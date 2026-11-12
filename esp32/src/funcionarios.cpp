#include "funcionarios.h"

#include <ArduinoJson.h>
#include <LittleFS.h>
#include <mbedtls/md.h>
#include <vector>

#include "rede.h"

static const char* ARQUIVO_LISTA = "/funcionarios.json";
static const char* ARQUIVO_ETAG = "/funcionarios.etag";

static std::vector<Funcionario> lista;
static bool carregada = false;
static String etagAtual = "";

// Le {"employees":[{"cardTag","fullName","isActive","pinHash"}]}.
static bool interpretarLista(const String& json, std::vector<Funcionario>& destino) {
  JsonDocument doc;
  if (deserializeJson(doc, json)) {
    return false;
  }

  JsonArray itens = doc["employees"].as<JsonArray>();
  if (itens.isNull()) {
    return false;
  }

  destino.clear();
  destino.reserve(itens.size());

  for (JsonObject item : itens) {
    Funcionario funcionario;
    funcionario.cartao = item["cardTag"] | "";
    funcionario.nome = item["fullName"] | "";
    funcionario.ativo = item["isActive"] | false;
    funcionario.hashSenha = item["pinHash"] | "";
    funcionario.cartao.toUpperCase();

    if (funcionario.cartao.length() > 0) {
      destino.push_back(funcionario);
    }
  }

  return true;
}

static String lerArquivo(const char* caminho) {
  File arquivo = LittleFS.open(caminho, FILE_READ);
  if (!arquivo) {
    return "";
  }
  String conteudo = arquivo.readString();
  arquivo.close();
  return conteudo;
}

static bool gravarArquivo(const char* caminho, const String& conteudo) {
  // Grava num temporario e renomeia: se desligar no meio, o arquivo antigo fica.
  String temporario = String(caminho) + ".tmp";

  File arquivo = LittleFS.open(temporario, FILE_WRITE);
  if (!arquivo) {
    return false;
  }
  size_t escrito = arquivo.print(conteudo);
  arquivo.close();

  if (escrito != conteudo.length()) {
    LittleFS.remove(temporario);
    return false;
  }

  LittleFS.remove(caminho);
  return LittleFS.rename(temporario, caminho);
}

void funcionariosIniciar() {
  String json = lerArquivo(ARQUIVO_LISTA);

  if (json.length() > 0 && interpretarLista(json, lista)) {
    carregada = true;
    etagAtual = lerArquivo(ARQUIVO_ETAG);
    Serial.println("Lista de funcionarios carregada da memoria: " + String(lista.size()));
  } else {
    Serial.println("Sem lista de funcionarios salva. Sera baixada do servidor.");
  }
}

bool funcionariosSincronizar() {
  String etagNovo;
  String corpo;

  int status = api("GET", "/device/employees", "", nullptr, carregada ? etagAtual : String(""), &etagNovo, &corpo);

  if (status == 304) {
    return true;
  }
  if (status != 200) {
    return false;
  }

  std::vector<Funcionario> nova;
  if (!interpretarLista(corpo, nova)) {
    Serial.println("Lista de funcionarios invalida recebida do servidor.");
    return false;
  }

  bool primeira = !carregada;
  lista.swap(nova);
  carregada = true;
  etagAtual = etagNovo;

  if (!gravarArquivo(ARQUIVO_LISTA, corpo) || !gravarArquivo(ARQUIVO_ETAG, etagNovo)) {
    Serial.println("Erro ao salvar a lista de funcionarios na memoria.");
  }

  Serial.println(
    String(primeira ? "Lista de funcionarios baixada: " : "Lista de funcionarios atualizada: ") +
    String(lista.size())
  );
  return true;
}

bool listaCarregada() {
  return carregada;
}

int totalFuncionarios() {
  return lista.size();
}

bool buscarFuncionario(const String& cartao, Funcionario& funcionario) {
  String procurado = cartao;
  procurado.toUpperCase();

  for (const Funcionario& item : lista) {
    if (item.cartao == procurado) {
      funcionario = item;
      return true;
    }
  }
  return false;
}

String nomeDoFuncionario(const String& cartao) {
  Funcionario funcionario;
  return buscarFuncionario(cartao, funcionario) ? funcionario.nome : String("?");
}

static String sha256Hex(const String& texto) {
  uint8_t resumo[32];
  mbedtls_md(
    mbedtls_md_info_from_type(MBEDTLS_MD_SHA256),
    (const unsigned char*)texto.c_str(),
    texto.length(),
    resumo
  );

  static const char* hex = "0123456789abcdef";
  String resultado = "";
  resultado.reserve(64);
  for (uint8_t byte : resumo) {
    resultado += hex[byte >> 4];
    resultado += hex[byte & 0x0F];
  }
  return resultado;
}

// Mesmo calculo do backend (service/pin.go): sha256(salt + ":" + pin).
bool senhaConfere(const Funcionario& funcionario, const String& senha) {
  int separador = funcionario.hashSenha.indexOf('$');
  if (separador <= 0) {
    return false;
  }

  String salt = funcionario.hashSenha.substring(0, separador);
  String esperado = funcionario.hashSenha.substring(separador + 1);

  String calculado = sha256Hex(salt + ":" + senha);

  // Comparacao sem sair no primeiro caractere diferente.
  if (calculado.length() != esperado.length()) {
    return false;
  }
  uint8_t diferenca = 0;
  for (unsigned int i = 0; i < calculado.length(); i++) {
    diferenca |= calculado[i] ^ esperado[i];
  }
  return diferenca == 0;
}

void funcionariosListar() {
  Serial.println();
  Serial.println("======= FUNCIONARIOS =======");

  if (!carregada) {
    Serial.println("Lista ainda nao baixada.");
  }

  for (const Funcionario& item : lista) {
    Serial.println("- " + item.cartao + "  " + item.nome + (item.ativo ? "" : "  (desativado)"));
  }

  Serial.println("Total: " + String(lista.size()));
  Serial.println("============================");
}
