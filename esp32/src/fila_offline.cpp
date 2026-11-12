#include "fila_offline.h"

#include <ArduinoJson.h>
#include <LittleFS.h>
#include <Preferences.h>

#include "cartao_sd.h"
#include "config.h"
#include "funcionarios.h"
#include "rede.h"
#include "relogio.h"

static const char* ARQUIVO_FILA = "/fila.jsonl";
static const char* ARQUIVO_TEMP = "/fila.tmp";

static int tamanho = 0;

static int contarLinhas() {
  File arquivo = LittleFS.open(ARQUIVO_FILA, FILE_READ);
  if (!arquivo) {
    return 0;
  }

  int linhas = 0;
  while (arquivo.available()) {
    String linha = arquivo.readStringUntil('\n');
    linha.trim();
    if (linha.length() > 0) {
      linhas++;
    }
  }

  arquivo.close();
  return linhas;
}

// Firmwares antigos guardavam a fila na NVS (chave "fila"). Move para o arquivo.
static void migrarFilaAntiga() {
  Preferences nvs;
  nvs.begin("ponto", false);

  if (!nvs.isKey("fila")) {
    nvs.end();
    return;
  }

  JsonDocument doc;
  String conteudo = nvs.getString("fila", "[]");

  if (!deserializeJson(doc, conteudo) && doc.is<JsonArray>()) {
    for (JsonObject ponto : doc.as<JsonArray>()) {
      filaAdicionar(ponto["cardTag"] | "", ponto["pin"] | "", ponto["recordedAt"] | "");
    }
    Serial.println("Fila antiga migrada: " + String(doc.as<JsonArray>().size()) + " ponto(s).");
  }

  nvs.remove("fila");
  nvs.end();
}

void filaIniciar() {
  // Se o relogio desligou no meio de uma regravacao da fila:
  // - os dois arquivos existem: a fila original esta inteira, descarta o temporario;
  // - so o temporario existe: ele ja estava completo, vira a fila.
  if (LittleFS.exists(ARQUIVO_FILA)) {
    if (LittleFS.exists(ARQUIVO_TEMP)) {
      LittleFS.remove(ARQUIVO_TEMP);
    }
  } else if (LittleFS.exists(ARQUIVO_TEMP)) {
    LittleFS.rename(ARQUIVO_TEMP, ARQUIVO_FILA);
  }

  tamanho = contarLinhas();
  migrarFilaAntiga();

  Serial.println("Fila offline: " + String(tamanho) + " ponto(s) pendente(s).");
}

int filaTamanho() {
  return tamanho;
}

bool filaCheia() {
  return tamanho >= MAX_FILA_OFFLINE;
}

bool filaAdicionar(const String& cardTag, const String& pin, const String& recordedAt) {
  if (filaCheia()) {
    return false;
  }

  JsonDocument doc;
  doc["cardTag"] = cardTag;
  doc["pin"] = pin;
  // Sem hora certa, o servidor usa a hora em que receber o ponto.
  if (recordedAt.length() > 0) {
    doc["recordedAt"] = recordedAt;
  }

  String linha;
  serializeJson(doc, linha);

  File arquivo = LittleFS.open(ARQUIVO_FILA, FILE_APPEND);
  if (!arquivo) {
    return false;
  }

  size_t escrito = arquivo.println(linha);
  arquivo.close();

  if (escrito < linha.length()) {
    return false;
  }

  tamanho++;
  return true;
}

// Copia o resto do arquivo aberto (a partir da posicao atual) para a nova fila.
static void regravarFila(File& origem, const String& linhaAtual) {
  File destino = LittleFS.open(ARQUIVO_TEMP, FILE_WRITE);
  if (!destino) {
    Serial.println("Erro ao regravar a fila offline.");
    return;
  }

  int restantes = 0;
  if (linhaAtual.length() > 0) {
    destino.println(linhaAtual);
    restantes++;
  }

  while (origem.available()) {
    String linha = origem.readStringUntil('\n');
    linha.trim();
    if (linha.length() > 0) {
      destino.println(linha);
      restantes++;
    }
  }

  destino.close();
  origem.close();

  LittleFS.remove(ARQUIVO_FILA);
  LittleFS.rename(ARQUIVO_TEMP, ARQUIVO_FILA);
  tamanho = restantes;
}

void filaEnviar(bool detalhado) {
  if (tamanho == 0) {
    if (detalhado) {
      Serial.println("Fila offline vazia.");
    }
    return;
  }

  File arquivo = LittleFS.open(ARQUIVO_FILA, FILE_READ);
  if (!arquivo) {
    tamanho = 0;
    return;
  }

  Serial.println("Enviando " + String(tamanho) + " ponto(s) da fila offline...");

  int enviados = 0;
  int descartados = 0;
  String pendente = "";

  while (arquivo.available()) {
    String linha = arquivo.readStringUntil('\n');
    linha.trim();
    if (linha.length() == 0) {
      continue;
    }

    JsonDocument ponto;
    if (deserializeJson(ponto, linha)) {
      descartados++;
      Serial.println("  Linha invalida descartada: " + linha);
      continue;
    }

    String tag = ponto["cardTag"] | "";
    String horario = ponto["recordedAt"] | "";
    String nome = nomeDoFuncionario(tag);

    JsonDocument resposta;
    int status = api("POST", "/device/punches", linha, &resposta);

    // Sem conexao, erro do servidor ou chave do relogio errada: tenta depois.
    if (status <= 0 || status >= 500 || chaveInvalida(status, resposta)) {
      if (chaveInvalida(status, resposta)) {
        Serial.println("  X-Device-Key invalido: fila mantida. Confira a chave.");
      }
      pendente = linha;
      break;
    }

    if (status == 201) {
      enviados++;
      String nomeServidor = resposta["employeeName"] | nome.c_str();
      String tipo = resposta["type"] == "exit" ? "saida" : "entrada";
      String hora = horaDeIso(resposta["record"]["recordedAt"] | "");

      Serial.println("  OK: " + nomeServidor + " " + tipo + " as " + hora);
      sdRegistrar(tag, nomeServidor, "enviado", tipo + " batida as " + hora);
    } else if (status == 409) {
      // Ja estava no servidor (ex.: o relogio desligou antes de tirar da fila).
      enviados++;
      Serial.println("  Ja registrado: " + tag + " " + horario);
      sdRegistrar(tag, nome, "enviado", "ja estava registrado (batida de " + horario + ")");
    } else {
      // 4xx nao muda tentando de novo (senha trocada, desativado...).
      descartados++;
      String motivo = mensagemDeErro(status, resposta);

      Serial.println("  Descartado (" + tag + "): " + motivo);
      sdRegistrar(tag, nome, "descartado", motivo + " (batida de " + horario + ")");
    }
  }

  // Grava so o que nao foi enviado (a linha que falhou e as seguintes).
  regravarFila(arquivo, pendente);

  Serial.println(
    "Fila offline: " + String(enviados) + " enviado(s), " +
    String(descartados) + " descartado(s), " +
    String(tamanho) + " pendente(s)."
  );
}

void filaListar() {
  Serial.println();
  Serial.println("======= FILA OFFLINE =======");

  File arquivo = LittleFS.open(ARQUIVO_FILA, FILE_READ);
  if (!arquivo || tamanho == 0) {
    Serial.println("Vazia.");
  } else {
    while (arquivo.available()) {
      String linha = arquivo.readStringUntil('\n');
      linha.trim();
      if (linha.length() == 0) {
        continue;
      }

      JsonDocument ponto;
      if (deserializeJson(ponto, linha)) {
        continue;
      }

      String tag = ponto["cardTag"] | "";
      Serial.println(
        "- " + tag + " (" + nomeDoFuncionario(tag) + ") " +
        (ponto["recordedAt"] | "sem horario")
      );
    }
  }

  if (arquivo) {
    arquivo.close();
  }
  Serial.println("Total: " + String(tamanho));
  Serial.println("============================");
}

void filaApagar() {
  LittleFS.remove(ARQUIVO_FILA);
  LittleFS.remove(ARQUIVO_TEMP);
  tamanho = 0;
}
