#include "cartao_sd.h"

#include <SD.h>
#include <SPI.h>

#include "config.h"
#include "relogio.h"

static const char* ARQUIVO_CONFIG = "/config.txt";
static const char* ARQUIVO_EXEMPLO = "/config-exemplo.txt";
static const char* PASTA_REGISTROS = "/registros";

static bool sdOk = false;
static unsigned long proximaTentativa = 0;

static void criarExemploDeConfig() {
  if (SD.exists(ARQUIVO_CONFIG) || SD.exists(ARQUIVO_EXEMPLO)) {
    return;
  }

  File arquivo = SD.open(ARQUIVO_EXEMPLO, FILE_WRITE);
  if (!arquivo) {
    return;
  }

  arquivo.println("# Copie este arquivo como config.txt, preencha e coloque o cartao");
  arquivo.println("# de volta no relogio. Ele e lido sempre que o relogio liga.");
  arquivo.println("# Linhas comecando com # sao ignoradas. Deixe vazio o que nao quer mudar.");
  arquivo.println("#");
  arquivo.println("# Rede Wi-Fi de 2.4 GHz (o ESP32 nao conecta em redes 5 GHz).");
  arquivo.println("wifi_ssid=NomeDaRede");
  arquivo.println("wifi_senha=SenhaDaRede");
  arquivo.println("#");
  arquivo.println("# Endereco do PC com o backend. Use um IP fixo (reserva de DHCP no roteador).");
  arquivo.println("servidor=http://192.168.0.10:8080/api/v1");
  arquivo.println("#");
  arquivo.println("# Mesmo valor de DEVICE_API_KEY do backend.");
  arquivo.println("chave=dev-device-key");
  arquivo.close();
}

static bool montar() {
  // A frequencia baixa deixa o barramento dividido com o RFID mais estavel.
  if (!SD.begin(PINO_SD_CS, SPI, 4000000)) {
    SD.end();
    return false;
  }

  if (SD.cardType() == CARD_NONE) {
    SD.end();
    return false;
  }

  if (!SD.exists(PASTA_REGISTROS)) {
    SD.mkdir(PASTA_REGISTROS);
  }
  criarExemploDeConfig();

  return true;
}

void sdIniciar() {
  sdOk = montar();
  proximaTentativa = millis() + 30000;

  if (sdOk) {
    Serial.println("Cartao SD OK: " + sdDescricao());
  } else {
    Serial.println("Cartao SD NAO encontrado. O historico nao sera gravado no SD.");
  }
}

void sdManter() {
  if (sdOk || (long)(millis() - proximaTentativa) < 0) {
    return;
  }

  proximaTentativa = millis() + 30000;
  sdOk = montar();

  if (sdOk) {
    Serial.println("Cartao SD encontrado: " + sdDescricao());
  }
}

bool sdPresente() {
  return sdOk;
}

String sdDescricao() {
  if (!sdOk) {
    return "ausente";
  }

  uint64_t totalMb = SD.totalBytes() / (1024 * 1024);
  uint64_t usadoMb = SD.usedBytes() / (1024 * 1024);
  return String((uint32_t)usadoMb) + " MB usados de " + String((uint32_t)totalMb) + " MB";
}

bool sdLerConfig(ConfigRede& config) {
  if (!sdOk || !SD.exists(ARQUIVO_CONFIG)) {
    return false;
  }

  File arquivo = SD.open(ARQUIVO_CONFIG, FILE_READ);
  if (!arquivo) {
    return false;
  }

  while (arquivo.available()) {
    String linha = arquivo.readStringUntil('\n');
    linha.trim();

    if (linha.length() == 0 || linha.startsWith("#")) {
      continue;
    }

    int igual = linha.indexOf('=');
    if (igual < 0) {
      continue;
    }

    String chave = linha.substring(0, igual);
    String valor = linha.substring(igual + 1);
    chave.trim();
    chave.toLowerCase();
    valor.trim();

    if (chave == "wifi_ssid") {
      config.wifiSsid = valor;
    } else if (chave == "wifi_senha") {
      config.wifiSenha = valor;
    } else if (chave == "servidor") {
      config.apiUrl = valor;
    } else if (chave == "chave") {
      config.chave = valor;
    }
  }

  arquivo.close();
  return true;
}

// Campos do CSV nao podem ter ; nem quebra de linha.
static String campoCsv(const String& valor) {
  String limpo = valor;
  limpo.replace(";", ",");
  limpo.replace("\n", " ");
  limpo.replace("\r", " ");
  return limpo;
}

void sdRegistrar(
  const String& cartao,
  const String& funcionario,
  const String& situacao,
  const String& detalhe
) {
  if (!sdOk) {
    return;
  }

  String mes = mesAtual();
  String caminho = String(PASTA_REGISTROS) + "/" + (mes.length() > 0 ? mes : String("sem-data")) + ".csv";
  bool novo = !SD.exists(caminho);

  File arquivo = SD.open(caminho, FILE_APPEND);
  if (!arquivo) {
    // Cartao removido ou com defeito: tenta montar de novo mais tarde.
    Serial.println("Erro ao gravar no cartao SD.");
    SD.end();
    sdOk = false;
    return;
  }

  if (novo) {
    // Separador ; abre direto no Excel em portugues.
    arquivo.println("data;hora;cartao;funcionario;situacao;detalhe");
  }

  String data = dataAtual();
  String hora = horaAtual();

  arquivo.println(
    (data.length() > 0 ? data : String("?")) + ";" +
    (hora.length() > 0 ? hora : String("?")) + ";" +
    campoCsv(cartao) + ";" +
    campoCsv(funcionario) + ";" +
    campoCsv(situacao) + ";" +
    campoCsv(detalhe)
  );
  arquivo.close();
}
