#include <Arduino.h>
#include <WiFi.h>
#include <HTTPClient.h>
#include <Preferences.h>
#include <SPI.h>
#include <MFRC522.h>
#include <Keypad.h>
#include <ArduinoJson.h>
#include <time.h>
#include <sys/time.h>

// =====================================================
// 1. CONFIGURACAO
// =====================================================

// Valores padrao, prontos para o simulador Wokwi. No ESP32 real,
// configure pelo monitor serial (comandos wifi, servidor e chave):
// fica salvo na memoria e nao precisa recompilar.
const char* WIFI_SSID_PADRAO = "Wokwi-GUEST";
const char* WIFI_PASSWORD_PADRAO = "";

// No Wokwi (VS Code), host.wokwi.internal aponta para o seu computador.
const char* API_BASE_URL_PADRAO = "http://host.wokwi.internal:8080/api/v1";

// Deve ser igual a variavel DEVICE_API_KEY do backend.
const char* DEVICE_API_KEY_PADRAO = "dev-device-key";

// Fuso horario de Brasilia (UTC-3) no formato POSIX.
const char* TIMEZONE = "<-03>3";

const int RFID_SS = 5;
const int RFID_RST = 21;
const int BUZZER_PIN = 17;

const int PIN_LENGTH = 4;
const int MAX_PIN_ATTEMPTS = 3;
const int MAX_OFFLINE_QUEUE = 50;

const unsigned long PIN_TIMEOUT_MS = 15000;
const unsigned long CARD_READ_TIMEOUT_MS = 60000;
const unsigned long POLL_INTERVAL_MS = 2000;
const unsigned long POLL_BACKOFF_MS = 15000;
const unsigned long SYNC_INTERVAL_MS = 30000;
const unsigned long WIFI_RETRY_MS = 10000;

// =====================================================
// 2. OBJETOS DO HARDWARE
// =====================================================

MFRC522 rfid(RFID_SS, RFID_RST);
Preferences memoria;

// Teclado 4x4
const byte ROWS = 4;
const byte COLS = 4;

char keys[ROWS][COLS] = {
  {'1', '2', '3', 'A'},
  {'4', '5', '6', 'B'},
  {'7', '8', '9', 'C'},
  {'*', '0', '#', 'D'}
};

byte rowPins[ROWS] = {13, 12, 14, 27};
byte colPins[COLS] = {26, 25, 33, 32};

Keypad keypad = Keypad(
  makeKeymap(keys),
  rowPins,
  colPins,
  ROWS,
  COLS
);

// =====================================================
// 3. ESTADOS DO SISTEMA
// =====================================================
//
// AGUARDANDO_TAG      -> tela inicial: aproximar o cartao bate o ponto
// DIGITANDO_PIN       -> cartao lido, esperando a senha no teclado
// LENDO_TAG_CADASTRO  -> o painel pediu para ler uma tag nova
//                        (cadastro/edicao de funcionario)
// =====================================================

enum Estado {
  AGUARDANDO_TAG,
  DIGITANDO_PIN,
  LENDO_TAG_CADASTRO
};

Estado estado = AGUARDANDO_TAG;

String tagAtual = "";
String nomeAtual = "";
String pinDigitado = "";
String leituraId = "";
String tagSimulada = "";

// Configuracao em uso (padrao ou salva na memoria).
String wifiSsid = "";
String wifiSenha = "";
String apiBaseUrl = "";
String deviceApiKey = "";

int tentativasPin = 0;
bool wifiEstavaConectado = false;

unsigned long prazoEstado = 0;
unsigned long proximaConsulta = 0;
unsigned long proximaSincronizacao = 0;
unsigned long proximaTentativaWifi = 0;

// =====================================================
// 4. FUNCOES AUXILIARES
// =====================================================

void bip(int duracao = 100) {
  digitalWrite(BUZZER_PIN, HIGH);
  delay(duracao);
  digitalWrite(BUZZER_PIN, LOW);
}

void bipSucesso() {
  bip(150);
}

void bipSaida() {
  bip(100);
  delay(80);
  bip(100);
}

void bipErro() {
  bip(600);
}

// Compara instantes de millis() sem quebrar no overflow (~49 dias).
bool passou(unsigned long instante) {
  return (long)(millis() - instante) >= 0;
}

bool wifiConectado() {
  return WiFi.status() == WL_CONNECTED;
}

bool relogioSincronizado() {
  // Qualquer data depois de 2023 indica que NTP ou o servidor acertaram o relogio.
  return time(nullptr) > 1700000000;
}

String obterDataHora() {
  if (!relogioSincronizado()) {
    return "HORA_NAO_SINCRONIZADA";
  }

  time_t agora = time(nullptr);
  struct tm info;
  localtime_r(&agora, &info);

  char buffer[25];
  strftime(buffer, sizeof(buffer), "%d/%m/%Y %H:%M:%S", &info);

  return String(buffer);
}

// Instante atual em ISO 8601 UTC, formato aceito pelo backend.
String instanteIsoUtc() {
  time_t agora = time(nullptr);
  struct tm info;
  gmtime_r(&agora, &info);

  char buffer[25];
  strftime(buffer, sizeof(buffer), "%Y-%m-%dT%H:%M:%SZ", &info);

  return String(buffer);
}

String horaDe(const String& isoComFuso) {
  // "2026-11-06T08:01:12-03:00" -> "08:01"
  if (isoComFuso.length() < 16) {
    return isoComFuso;
  }
  return isoComFuso.substring(11, 16);
}

String lerUID() {
  String uid = "";

  for (byte i = 0; i < rfid.uid.size; i++) {
    if (rfid.uid.uidByte[i] < 0x10) {
      uid += "0";
    }

    uid += String(rfid.uid.uidByte[i], HEX);

    if (i < rfid.uid.size - 1) {
      uid += ":";
    }
  }

  uid.toUpperCase();
  return uid;
}

void voltarAoInicio() {
  tagAtual = "";
  nomeAtual = "";
  pinDigitado = "";
  leituraId = "";
  tentativasPin = 0;
  estado = AGUARDANDO_TAG;

  Serial.println();
  Serial.println("Aproxime o cartao para bater o ponto.");
}

// =====================================================
// 5. COMUNICACAO COM O BACKEND
// =====================================================
//
// Todas as rotas do relogio ficam em /api/v1/device/*
// e exigem o header X-Device-Key.
//
// Retorna o status HTTP, ou um valor <= 0 quando nao foi
// possivel falar com o servidor (sem Wi-Fi, timeout...).
// =====================================================

int requisicaoApi(
  const char* metodo,
  const String& caminho,
  const String& corpo,
  JsonDocument* resposta
) {
  if (!wifiConectado()) {
    return -1;
  }

  HTTPClient http;
  http.setConnectTimeout(2000);
  http.setTimeout(4000);

  if (!http.begin(apiBaseUrl + caminho)) {
    return -1;
  }

  http.addHeader("X-Device-Key", deviceApiKey);

  if (corpo.length() > 0) {
    http.addHeader("Content-Type", "application/json");
  }

  int status = http.sendRequest(metodo, corpo);

  if (status > 0 && resposta != nullptr) {
    String texto = http.getString();

    if (texto.length() > 0) {
      deserializeJson(*resposta, texto);
    }
  }

  http.end();
  return status;
}

String mensagemDeErro(int status, JsonDocument& resposta) {
  if (status <= 0) {
    return "servidor indisponivel";
  }

  String mensagem = resposta["error"] | "";

  if (mensagem.length() == 0) {
    mensagem = "erro HTTP " + String(status);
  }

  return mensagem;
}

// Usa o horario do servidor quando o NTP nao responde
// (comum em redes simuladas ou bloqueadas).
void sincronizarRelogioPeloServidor() {
  if (relogioSincronizado()) {
    return;
  }

  JsonDocument resposta;

  if (requisicaoApi("GET", "/device/ping", "", &resposta) != 200) {
    return;
  }

  struct timeval agora = {};
  agora.tv_sec = (time_t)resposta["unixTime"].as<long long>();
  settimeofday(&agora, nullptr);

  Serial.println("Relogio acertado pelo servidor: " + obterDataHora());
}

// =====================================================
// 6. FILA OFFLINE
// =====================================================
//
// Se o backend estiver fora do ar, o ponto e guardado na
// NVS (chave "fila") e reenviado a cada 30 segundos.
// A senha so e validada quando a batida chega ao servidor.
//
// Estrutura salva:
// [
//   {
//     "cardTag": "AA:BB:CC:DD",
//     "pin": "1234",
//     "recordedAt": "2026-11-06T11:00:00Z"
//   }
// ]
// =====================================================

void carregarFila(JsonDocument& doc) {
  String conteudo = memoria.getString("fila", "[]");

  if (deserializeJson(doc, conteudo) || !doc.is<JsonArray>()) {
    doc.to<JsonArray>();
  }
}

void salvarFila(JsonDocument& doc) {
  String conteudo;
  serializeJson(doc, conteudo);

  if (memoria.putString("fila", conteudo) == 0 && conteudo.length() > 0) {
    Serial.println("Erro ao salvar a fila offline.");
  }
}

int tamanhoFila() {
  JsonDocument doc;
  carregarFila(doc);
  return doc.as<JsonArray>().size();
}

void enfileirarPonto(const String& cardTag, const String& pin) {
  JsonDocument doc;
  carregarFila(doc);

  JsonArray fila = doc.as<JsonArray>();

  // Descarta o registro mais antigo se o limite for atingido.
  if (fila.size() >= MAX_OFFLINE_QUEUE) {
    fila.remove(0);
  }

  JsonObject ponto = fila.add<JsonObject>();
  ponto["cardTag"] = cardTag;
  ponto["pin"] = pin;

  // Sem relogio sincronizado o servidor usa a hora em que receber o ponto.
  if (relogioSincronizado()) {
    ponto["recordedAt"] = instanteIsoUtc();
  }

  salvarFila(doc);
}

void enviarFila(bool detalhado) {
  JsonDocument doc;
  carregarFila(doc);

  JsonArray fila = doc.as<JsonArray>();

  if (fila.size() == 0) {
    if (detalhado) {
      Serial.println("Fila offline vazia.");
    }
    return;
  }

  Serial.println("Enviando " + String(fila.size()) + " ponto(s) da fila offline...");

  int enviados = 0;

  while (fila.size() > 0) {
    String corpo;
    serializeJson(fila[0], corpo);

    JsonDocument resposta;
    int status = requisicaoApi("POST", "/device/punches", corpo, &resposta);

    // Sem conexao ou erro do servidor: tenta de novo depois.
    if (status <= 0 || status >= 500) {
      break;
    }

    String tag = fila[0]["cardTag"].as<String>();

    if (status == 201) {
      enviados++;
      Serial.println(
        "  OK: " + resposta["employeeName"].as<String>() +
        " as " + horaDe(resposta["record"]["recordedAt"].as<String>())
      );
    } else {
      // 4xx nao vai mudar com nova tentativa (senha errada, duplicado...).
      Serial.println("  Descartado (" + tag + "): " + mensagemDeErro(status, resposta));
    }

    fila.remove(0);
    salvarFila(doc);
  }

  Serial.println(
    "Fila offline: " + String(enviados) + " enviado(s), " +
    String(fila.size()) + " pendente(s)."
  );
}

void listarFila() {
  JsonDocument doc;
  carregarFila(doc);

  JsonArray fila = doc.as<JsonArray>();

  Serial.println();
  Serial.println("======= FILA OFFLINE =======");

  if (fila.size() == 0) {
    Serial.println("Nenhum ponto pendente.");
  }

  for (JsonObject ponto : fila) {
    Serial.println(
      ponto["cardTag"].as<String>() + " | " +
      (ponto["recordedAt"] | "sem horario")
    );
  }

  Serial.println("============================");
}

// =====================================================
// 7. LEITURA RFID
// =====================================================

bool lerTag(String& uid) {
  // Tag digitada no Serial (comando "tag") para testar sem leitor.
  if (tagSimulada.length() > 0) {
    uid = tagSimulada;
    tagSimulada = "";
    Serial.println("UID simulado: " + uid);
    return true;
  }

  if (!rfid.PICC_IsNewCardPresent()) {
    return false;
  }

  if (!rfid.PICC_ReadCardSerial()) {
    return false;
  }

  uid = lerUID();

  rfid.PICC_HaltA();
  rfid.PCD_StopCrypto1();

  Serial.println("UID detectado: " + uid);

  return true;
}

// =====================================================
// 8. BATER PONTO
// =====================================================

void iniciarPonto(const String& tag) {
  tagAtual = tag;
  nomeAtual = "";

  JsonDocument resposta;
  int status = requisicaoApi("GET", "/device/employees/" + tag, "", &resposta);

  if (status == 200) {
    if (!resposta["isActive"].as<bool>()) {
      Serial.println("Funcionario desativado. Procure o RH.");
      bipErro();
      voltarAoInicio();
      return;
    }

    nomeAtual = resposta["fullName"].as<String>();
    Serial.println("Ola, " + nomeAtual + "!");
  } else if (status > 0) {
    Serial.println("Nao foi possivel bater o ponto: " + mensagemDeErro(status, resposta));
    bipErro();
    voltarAoInicio();
    return;
  } else {
    Serial.println("Servidor indisponivel. O ponto sera salvo offline.");
  }

  pinDigitado = "";
  tentativasPin = 0;
  prazoEstado = millis() + PIN_TIMEOUT_MS;
  estado = DIGITANDO_PIN;

  Serial.println("Digite a senha de 4 numeros no teclado.");
  Serial.println("* apaga | D cancela");

  bip(80);
}

void enviarPonto() {
  JsonDocument corpoDoc;
  corpoDoc["cardTag"] = tagAtual;
  corpoDoc["pin"] = pinDigitado;

  if (relogioSincronizado()) {
    corpoDoc["recordedAt"] = instanteIsoUtc();
  }

  String corpo;
  serializeJson(corpoDoc, corpo);

  Serial.println();

  JsonDocument resposta;
  int status = requisicaoApi("POST", "/device/punches", corpo, &resposta);

  if (status == 201) {
    bool saida = resposta["type"] == "exit";

    Serial.println("******** PONTO REGISTRADO ********");
    Serial.println("Usuario: " + resposta["employeeName"].as<String>());
    Serial.println(String("Tipo: ") + (saida ? "SAIDA" : "ENTRADA"));
    Serial.println("Hora: " + horaDe(resposta["record"]["recordedAt"].as<String>()));
    Serial.println("Batida do dia: " + String(resposta["punchNumber"].as<int>()));
    Serial.println("**********************************");

    if (saida) {
      bipSaida();
    } else {
      bipSucesso();
    }

    voltarAoInicio();
    return;
  }

  if (status == 401) {
    tentativasPin++;
    bipErro();

    if (tentativasPin < MAX_PIN_ATTEMPTS) {
      Serial.println(
        mensagemDeErro(status, resposta) + ". Tente novamente (" +
        String(tentativasPin) + "/" + String(MAX_PIN_ATTEMPTS) + ")."
      );
      pinDigitado = "";
      prazoEstado = millis() + PIN_TIMEOUT_MS;
      return;
    }

    Serial.println("Muitas tentativas. Aproxime o cartao novamente.");
    voltarAoInicio();
    return;
  }

  if (status > 0 && status < 500) {
    Serial.println("Ponto nao registrado: " + mensagemDeErro(status, resposta));
    bipErro();
    voltarAoInicio();
    return;
  }

  enfileirarPonto(tagAtual, pinDigitado);

  Serial.println("******** PONTO SALVO OFFLINE ********");
  Serial.println("Sem conexao com o servidor.");
  Serial.println("Data/hora: " + obterDataHora());
  Serial.println("Sera enviado quando a conexao voltar.");
  Serial.println("Pendentes na fila: " + String(tamanhoFila()));
  Serial.println("*************************************");

  bipSucesso();
  voltarAoInicio();
}

void processarTeclado() {
  char tecla = keypad.getKey();

  if (!tecla) {
    return;
  }

  // Qualquer tecla renova o tempo para digitar.
  prazoEstado = millis() + PIN_TIMEOUT_MS;

  if (tecla == 'D') {
    Serial.println();
    Serial.println("Operacao cancelada.");
    voltarAoInicio();
    return;
  }

  if (tecla == '*') {
    pinDigitado = "";
    Serial.println();
    Serial.println("Senha apagada.");
    return;
  }

  if (tecla >= '0' && tecla <= '9' && pinDigitado.length() < PIN_LENGTH) {
    pinDigitado += tecla;
    Serial.print("*");
  }

  // Confirma sozinho ao completar os 4 numeros (# tambem confirma).
  if (pinDigitado.length() == PIN_LENGTH) {
    enviarPonto();
  } else if (tecla == '#') {
    Serial.println();
    Serial.println("A senha tem " + String(PIN_LENGTH) + " numeros.");
  }
}

// =====================================================
// 9. LEITURA DE TAG PEDIDA PELO PAINEL
// =====================================================
//
// No cadastro de funcionario, o painel cria um pedido em
// POST /api/v1/card-reads. O relogio consulta os pedidos,
// le a proxima tag e devolve o UID para o painel.
// =====================================================

void consultarPedidoDeLeitura() {
  if (!passou(proximaConsulta)) {
    return;
  }

  JsonDocument resposta;
  int status = requisicaoApi("GET", "/device/card-reads/pending", "", &resposta);

  // Se o servidor nao responde, espera mais para nao travar o leitor.
  proximaConsulta = millis() + (status > 0 ? POLL_INTERVAL_MS : POLL_BACKOFF_MS);

  if (status == 200) {
    String id = resposta["id"].as<String>();

    if (estado == LENDO_TAG_CADASTRO && id == leituraId) {
      return;
    }

    leituraId = id;
    prazoEstado = millis() + CARD_READ_TIMEOUT_MS;
    estado = LENDO_TAG_CADASTRO;

    Serial.println();
    Serial.println("[PAINEL] Cadastro de cartao solicitado.");
    Serial.println("Aproxime a tag que sera vinculada ao funcionario.");

    bip(80);
    delay(80);
    bip(80);
  } else if (status == 204 && estado == LENDO_TAG_CADASTRO) {
    Serial.println("[PAINEL] Leitura cancelada ou expirada.");
    voltarAoInicio();
  }
}

void enviarTagCadastro(const String& tag) {
  JsonDocument corpoDoc;
  corpoDoc["cardTag"] = tag;

  String corpo;
  serializeJson(corpoDoc, corpo);

  JsonDocument resposta;
  int status = requisicaoApi("POST", "/device/card-reads/" + leituraId, corpo, &resposta);

  if (status == 200) {
    Serial.println("Tag " + tag + " enviada ao painel.");

    if (!resposta["assignedTo"].isNull()) {
      Serial.println(
        "Atencao: essa tag ja pertence a " +
        resposta["assignedTo"]["fullName"].as<String>() + "."
      );
    }

    bipSucesso();
  } else {
    Serial.println("Nao foi possivel enviar a tag: " + mensagemDeErro(status, resposta));
    bipErro();
  }

  voltarAoInicio();
}

// =====================================================
// 10. CONFIGURACAO, WI-FI E SINCRONIZACAO
// =====================================================
//
// Wi-Fi, servidor e chave ficam na NVS (namespace "ponto").
// Sem nada salvo, valem os valores padrao do topo do arquivo.
// =====================================================

void carregarConfiguracao() {
  wifiSsid = memoria.getString("wifi_ssid", WIFI_SSID_PADRAO);
  wifiSenha = memoria.getString("wifi_senha", WIFI_PASSWORD_PADRAO);
  apiBaseUrl = memoria.getString("api_url", API_BASE_URL_PADRAO);
  deviceApiKey = memoria.getString("api_chave", DEVICE_API_KEY_PADRAO);
}

void conectarWifi() {
  // Canal fixo acelera a conexao no Wokwi; em rede real o canal e automatico.
  int canal = wifiSsid == "Wokwi-GUEST" ? 6 : 0;

  WiFi.disconnect();
  WiFi.begin(wifiSsid.c_str(), wifiSenha.c_str(), canal);
  proximaTentativaWifi = millis() + WIFI_RETRY_MS;

  Serial.println("Conectando ao Wi-Fi \"" + wifiSsid + "\"...");
}

void manterWifi() {
  bool conectado = wifiConectado();

  if (conectado != wifiEstavaConectado) {
    wifiEstavaConectado = conectado;

    if (conectado) {
      Serial.println("Wi-Fi conectado. IP: " + WiFi.localIP().toString());
      proximaSincronizacao = millis();
    } else {
      Serial.println("Wi-Fi desconectado. Pontos serao salvos offline.");
    }
  }

  if (!conectado && passou(proximaTentativaWifi)) {
    WiFi.reconnect();
    proximaTentativaWifi = millis() + WIFI_RETRY_MS;
  }
}

void sincronizarEmSegundoPlano() {
  if (!wifiConectado() || !passou(proximaSincronizacao)) {
    return;
  }

  proximaSincronizacao = millis() + SYNC_INTERVAL_MS;

  sincronizarRelogioPeloServidor();

  if (tamanhoFila() > 0) {
    enviarFila(false);
  }
}

// =====================================================
// 11. COMANDOS SERIAL (DESENVOLVIMENTO / SIMULACAO)
// =====================================================

void mostrarAjuda() {
  Serial.println();
  Serial.println("================================");
  Serial.println("       PONTO ELETRONICO");
  Serial.println("================================");
  Serial.println("Uso normal: aproxime o cartao e");
  Serial.println("digite a senha no teclado.");
  Serial.println();
  Serial.println("Comandos pelo Serial:");
  Serial.println("  tag AA:BB:CC:DD  simula um cartao");
  Serial.println("  pin 1234         simula a senha");
  Serial.println("  status           Wi-Fi, servidor e fila");
  Serial.println("  fila             lista pontos offline");
  Serial.println("  sync             envia a fila agora");
  Serial.println("  limpar-fila      apaga a fila offline");
  Serial.println("  hora             mostra data e hora");
  Serial.println("  ajuda            mostra esta tela");
  Serial.println();
  Serial.println("Configuracao (fica salva):");
  Serial.println("  wifi             pergunta nome e senha da rede");
  Serial.println("  servidor URL     ex.: servidor http://192.168.0.10:8080/api/v1");
  Serial.println("  chave CHAVE      DEVICE_API_KEY do backend");
  Serial.println("  config           mostra a configuracao atual");
  Serial.println("  reset-config     volta aos valores padrao");
  Serial.println("================================");
}

// Espera uma linha no Serial (usado so nos comandos de configuracao).
String lerLinhaSerial(unsigned long timeoutMs) {
  unsigned long prazo = millis() + timeoutMs;

  while (!passou(prazo)) {
    if (Serial.available()) {
      String linha = Serial.readStringUntil('\n');
      linha.trim();
      return linha;
    }
    delay(10);
  }

  return "";
}

void configurarWifi() {
  Serial.print("Nome da rede (SSID): ");
  String ssid = lerLinhaSerial(60000);
  Serial.println(ssid);

  if (ssid.length() == 0) {
    Serial.println("Cancelado.");
    return;
  }

  Serial.print("Senha (vazio se a rede for aberta): ");
  String senha = lerLinhaSerial(60000);
  Serial.println(senha.length() > 0 ? "********" : "(sem senha)");

  memoria.putString("wifi_ssid", ssid);
  memoria.putString("wifi_senha", senha);

  carregarConfiguracao();
  conectarWifi();
}

void mostrarConfiguracao() {
  Serial.println();
  Serial.println("========= CONFIGURACAO =========");
  Serial.println("Wi-Fi: " + wifiSsid);
  Serial.println("Servidor: " + apiBaseUrl);
  Serial.println("Chave: " + deviceApiKey);
  Serial.println("================================");
}

void mostrarStatus() {
  JsonDocument resposta;
  int status = requisicaoApi("GET", "/device/ping", "", &resposta);

  Serial.println();
  Serial.println("============ STATUS ============");
  String wifi = wifiConectado()
    ? "conectado (" + WiFi.localIP().toString() + ")"
    : String("desconectado");
  String servidor = status == 200
    ? String("online")
    : "offline (" + mensagemDeErro(status, resposta) + ")";

  Serial.println("Wi-Fi: " + wifiSsid + " - " + wifi);
  Serial.println("API: " + apiBaseUrl);
  Serial.println("Servidor: " + servidor);
  Serial.println("Data/hora: " + obterDataHora());
  Serial.println("Fila offline: " + String(tamanhoFila()) + " ponto(s)");
  Serial.println("================================");
}

void processarSerial() {
  if (!Serial.available()) {
    return;
  }

  String linha = Serial.readStringUntil('\n');
  linha.trim();

  if (linha.length() == 0) {
    return;
  }

  int espaco = linha.indexOf(' ');
  String comando = espaco < 0 ? linha : linha.substring(0, espaco);
  String argumento = espaco < 0 ? "" : linha.substring(espaco + 1);

  comando.toLowerCase();
  argumento.trim();

  if (comando == "tag") {
    argumento.toUpperCase();

    if (argumento.length() == 0) {
      Serial.println("Uso: tag AA:BB:CC:DD");
    } else if (estado == DIGITANDO_PIN) {
      Serial.println("Termine a senha atual ou cancele com D.");
    } else {
      tagSimulada = argumento;
    }
  } else if (comando == "pin") {
    if (estado != DIGITANDO_PIN) {
      Serial.println("Nenhum cartao aguardando senha.");
    } else if (argumento.length() != PIN_LENGTH) {
      Serial.println("A senha tem " + String(PIN_LENGTH) + " numeros.");
    } else {
      pinDigitado = argumento;
      enviarPonto();
    }
  } else if (comando == "status") {
    mostrarStatus();
  } else if (comando == "fila") {
    listarFila();
  } else if (comando == "sync") {
    enviarFila(true);
  } else if (comando == "limpar-fila") {
    memoria.remove("fila");
    Serial.println("Fila offline apagada.");
  } else if (comando == "wifi") {
    configurarWifi();
  } else if (comando == "servidor") {
    if (!argumento.startsWith("http://") && !argumento.startsWith("https://")) {
      Serial.println("Uso: servidor http://IP_DO_COMPUTADOR:8080/api/v1");
    } else {
      while (argumento.endsWith("/")) {
        argumento.remove(argumento.length() - 1);
      }
      memoria.putString("api_url", argumento);
      carregarConfiguracao();
      Serial.println("Servidor salvo: " + apiBaseUrl);
      mostrarStatus();
    }
  } else if (comando == "chave") {
    if (argumento.length() == 0) {
      Serial.println("Uso: chave MINHA_CHAVE");
    } else {
      memoria.putString("api_chave", argumento);
      carregarConfiguracao();
      Serial.println("Chave salva.");
    }
  } else if (comando == "config") {
    mostrarConfiguracao();
  } else if (comando == "reset-config") {
    memoria.remove("wifi_ssid");
    memoria.remove("wifi_senha");
    memoria.remove("api_url");
    memoria.remove("api_chave");
    carregarConfiguracao();
    Serial.println("Configuracao padrao restaurada.");
    conectarWifi();
  } else if (comando == "hora") {
    Serial.println("Data/hora: " + obterDataHora());
  } else if (comando == "ajuda" || comando == "help" || comando == "menu") {
    mostrarAjuda();
  } else {
    Serial.println("Comando desconhecido. Digite ajuda.");
  }
}

// =====================================================
// 12. SETUP
// =====================================================

void setup() {
  Serial.begin(115200);

  // Permite que readStringUntil use timeout curto.
  Serial.setTimeout(100);

  pinMode(BUZZER_PIN, OUTPUT);
  digitalWrite(BUZZER_PIN, LOW);

  SPI.begin();
  rfid.PCD_Init();

  // Namespace persistente.
  memoria.begin("ponto", false);

  Serial.println();
  Serial.println("Inicializando ponto eletronico...");

  carregarConfiguracao();

  // O sistema nao fica preso esperando Wi-Fi.
  WiFi.mode(WIFI_STA);
  conectarWifi();

  configTzTime(TIMEZONE, "pool.ntp.org", "time.google.com");

  Serial.println("Servidor: " + apiBaseUrl);
  Serial.println("Sem conexao, os pontos ficam salvos e sao enviados depois.");

  mostrarAjuda();
  voltarAoInicio();
}

// =====================================================
// 13. LOOP PRINCIPAL
// =====================================================

void loop() {
  manterWifi();
  processarSerial();

  String uid;

  switch (estado) {
    case AGUARDANDO_TAG:
      if (lerTag(uid)) {
        iniciarPonto(uid);
        break;
      }

      consultarPedidoDeLeitura();
      sincronizarEmSegundoPlano();
      break;

    case DIGITANDO_PIN:
      processarTeclado();

      if (estado == DIGITANDO_PIN && passou(prazoEstado)) {
        Serial.println();
        Serial.println("Tempo esgotado.");
        bipErro();
        voltarAoInicio();
      }
      break;

    case LENDO_TAG_CADASTRO:
      if (lerTag(uid)) {
        enviarTagCadastro(uid);
        break;
      }

      consultarPedidoDeLeitura();

      if (estado == LENDO_TAG_CADASTRO && passou(prazoEstado)) {
        Serial.println("[PAINEL] Nenhuma tag lida a tempo.");
        voltarAoInicio();
      }
      break;
  }
}
