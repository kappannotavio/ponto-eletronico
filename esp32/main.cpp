
#include <WiFi.h>
#include <time.h>
#include <Preferences.h>
#include <SPI.h>
#include <MFRC522.h>
#include <Keypad.h>
#include <ArduinoJson.h>
#include <mbedtls/sha256.h>

// =====================================================
// 1. CONFIGURACAO
// =====================================================

// Wi-Fi opcional: usado somente para sincronizar o relogio.
// O menu, o cadastro e o ponto funcionam sem internet.
const char* WIFI_SSID = "Wokwi-GUEST";
const char* WIFI_PASSWORD = "";

// Fuso horario de Brasilia: UTC-3
const long GMT_OFFSET = -3 * 3600;
const int DAYLIGHT_OFFSET = 0;

const int RFID_SS = 5;
const int RFID_RST = 21;
const int BUZZER_PIN = 17;

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

enum Estado {
  MENU,
  LENDO_TAG_CADASTRO,
  DIGITANDO_SENHA_CADASTRO,
  LENDO_TAG_PONTO,
  LENDO_TAG_NOME,
  LENDO_TAG_TROCA_SENHA,
  DIGITANDO_NOVA_SENHA
};

Estado estado = MENU;

String nomeTemporario = "";
String uidTemporario = "";
String senhaDigitada = "";

const char* NOMES_TESTE[] = {
  "Maria Silva",
  "Joao Santos",
  "Ana Oliveira",
  "Pedro Souza",
  "Lucas Ferreira",
  "Julia Costa"
};

const int QUANTIDADE_NOMES =
  sizeof(NOMES_TESTE) / sizeof(NOMES_TESTE[0]);

// =====================================================
// 4. FUNCOES AUXILIARES
// =====================================================

void bip(int duracao = 100) {
  digitalWrite(BUZZER_PIN, HIGH);
  delay(duracao);
  digitalWrite(BUZZER_PIN, LOW);
}

void mostrarMenu() {
  Serial.println();
  Serial.println("================================");
  Serial.println("       PONTO ELETRONICO");
  Serial.println("================================");
  Serial.println("1 - Cadastrar novo usuario");
  Serial.println("2 - Bater ponto");
  Serial.println("3 - Listar usuarios");
  Serial.println("4 - Alterar nome");
  Serial.println("5 - Trocar senha");
  Serial.println("6 - Consultar historico de pontos");
  Serial.println("7 - Mostrar data e hora");
  Serial.println("================================");
  Serial.println("Digite uma opcao e pressione Enter:");
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

String calcularHash(const String& uid, const String& senha) {
  // Demonstracao local. SHA-256 simples nao e adequado
  // para proteger senhas em um sistema de producao.
  String entrada = uid + ":" + senha;

  unsigned char hash[32];

  mbedtls_sha256(
    (const unsigned char*)entrada.c_str(),
    entrada.length(),
    hash,
    0
  );

  char resultado[65];

  for (int i = 0; i < 32; i++) {
    sprintf(&resultado[i * 2], "%02x", hash[i]);
  }

  resultado[64] = '\0';
  return String(resultado);
}

String obterDataHora() {
  struct tm info;

  if (getLocalTime(&info, 100)) {
    char buffer[25];

    strftime(
      buffer,
      sizeof(buffer),
      "%d/%m/%Y %H:%M:%S",
      &info
    );

    return String(buffer);
  }

  // Sem NTP/RTC, millis nao representa data real.
  return "HORA_NAO_SINCRONIZADA";
}

// =====================================================
// 5. ARMAZENAMENTO PERSISTENTE
// =====================================================
//
// Preferences/NVS substitui LittleFS neste exemplo.
// Os usuarios continuam salvos depois de reiniciar.
//
// Estrutura salva na chave "usuarios":
// [
//   {
//     "uid": "AA:BB:CC:DD",
//     "nome": "Maria Silva",
//     "hash": "..."
//   }
// ]
// =====================================================

bool carregarUsuarios(JsonDocument& doc) {
  String conteudo = memoria.getString("usuarios", "[]");

  DeserializationError erro =
    deserializeJson(doc, conteudo);

  if (erro) {
    Serial.println("Erro ao ler os usuarios salvos.");
    return false;
  }

  return true;
}

bool salvarUsuarios(JsonDocument& doc) {
  String conteudo;
  serializeJson(doc, conteudo);

  size_t gravados = memoria.putString(
    "usuarios",
    conteudo
  );

  if (gravados == 0 && conteudo.length() > 0) {
    Serial.println("Erro ao salvar usuarios.");
    return false;
  }

  return true;
}

int buscarUsuario(JsonDocument& doc, const String& uid) {
  JsonArray usuarios = doc.as<JsonArray>();

  for (int i = 0; i < usuarios.size(); i++) {
    if (usuarios[i]["uid"].as<String>() == uid) {
      return i;
    }
  }

  return -1;
}

bool usuarioExiste(const String& uid) {
  JsonDocument doc;

  if (!carregarUsuarios(doc)) {
    return false;
  }

  return buscarUsuario(doc, uid) >= 0;
}

bool salvarUsuario(
  const String& uid,
  const String& nome,
  const String& hash
) {
  JsonDocument doc;

  if (!carregarUsuarios(doc)) {
    return false;
  }

  JsonArray usuarios = doc.as<JsonArray>();
  int indice = buscarUsuario(doc, uid);

  JsonObject usuario;

  if (indice >= 0) {
    // Atualiza o cadastro existente para o mesmo UID.
    usuario = usuarios[indice].as<JsonObject>();
  } else {
    usuario = usuarios.add<JsonObject>();
  }

  usuario["uid"] = uid;
  usuario["nome"] = nome;
  usuario["hash"] = hash;

  return salvarUsuarios(doc);
}

bool obterUsuario(
  const String& uid,
  String& nome,
  String& hash
) {
  JsonDocument doc;

  if (!carregarUsuarios(doc)) {
    return false;
  }

  int indice = buscarUsuario(doc, uid);

  if (indice < 0) {
    return false;
  }

  nome = doc[indice]["nome"].as<String>();
  hash = doc[indice]["hash"].as<String>();

  return true;
}

void listarUsuarios() {
  JsonDocument doc;

  if (!carregarUsuarios(doc)) {
    return;
  }

  JsonArray usuarios = doc.as<JsonArray>();

  Serial.println();
  Serial.println("===== USUARIOS CADASTRADOS =====");

  if (usuarios.size() == 0) {
    Serial.println("Nenhum usuario cadastrado.");
  }

  for (JsonObject usuario : usuarios) {
    Serial.println(
      usuario["nome"].as<String>() +
      " | UID: " +
      usuario["uid"].as<String>()
    );
  }

  Serial.println("================================");
}

// =====================================================
// 6. HISTORICO LOCAL DE PONTOS
// =====================================================
//
// O historico e salvo na chave "pontos".
// Mantemos os ultimos 30 registros neste prototipo
// para limitar o uso da memoria NVS.
// =====================================================

void registrarPonto(
  const String& uid,
  const String& nome
) {
  JsonDocument doc;

  String conteudo = memoria.getString("pontos", "[]");

  if (deserializeJson(doc, conteudo)) {
    doc.to<JsonArray>();
  }

  JsonArray pontos = doc.as<JsonArray>();

  // Remove o registro mais antigo se o limite for atingido.
  if (pontos.size() >= 30) {
    pontos.remove(0);
  }

  JsonObject ponto = pontos.add<JsonObject>();

  ponto["uid"] = uid;
  ponto["nome"] = nome;
  ponto["data_hora"] = obterDataHora();
  ponto["uptime_ms"] = millis();

  String novoConteudo;
  serializeJson(doc, novoConteudo);

  if (memoria.putString("pontos", novoConteudo) == 0 &&
      novoConteudo.length() > 0) {
    Serial.println("Erro ao salvar o historico.");
    return;
  }

  Serial.println();
  Serial.println("******** PONTO REGISTRADO ********");
  Serial.println("Usuario: " + nome);
  Serial.println("UID: " + uid);
  Serial.println("Data/hora: " + obterDataHora());
  Serial.println("**********************************");

  bip(200);
}

void consultarHistorico() {
  JsonDocument doc;

  String conteudo = memoria.getString("pontos", "[]");

  if (deserializeJson(doc, conteudo)) {
    Serial.println("Erro ao ler o historico.");
    return;
  }

  JsonArray pontos = doc.as<JsonArray>();

  Serial.println();
  Serial.println("======= HISTORICO DE PONTOS =======");

  if (pontos.size() == 0) {
    Serial.println("Nenhum ponto registrado.");
  }

  for (JsonObject ponto : pontos) {
    Serial.println(
      ponto["nome"].as<String>() +
      " | " +
      ponto["data_hora"].as<String>() +
      " | UID: " +
      ponto["uid"].as<String>()
    );
  }

  Serial.println("===================================");
}

// =====================================================
// 7. LEITURA RFID
// =====================================================

bool lerTag(String& uid) {
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
// 8. INICIAR OPERACOES
// =====================================================

void iniciarCadastro() {
  int indice = random(QUANTIDADE_NOMES);

  nomeTemporario = NOMES_TESTE[indice];

  Serial.println();
  Serial.println("[SIMULACAO DE DASHBOARD]");
  Serial.println("Solicitacao recebida para cadastrar:");
  Serial.println("Nome: " + nomeTemporario);
  Serial.println("Aproxime uma tag RFID.");

  estado = LENDO_TAG_CADASTRO;
}

void iniciarPonto() {
  Serial.println();
  Serial.println("Modo de registro de ponto.");
  Serial.println("Aproxime a tag RFID.");

  estado = LENDO_TAG_PONTO;
}

void iniciarAlteracaoNome() {
  Serial.println();
  Serial.println("Modo de alteracao de nome.");
  Serial.println("Aproxime a tag do usuario.");

  estado = LENDO_TAG_NOME;
}

void iniciarTrocaSenha() {
  Serial.println();
  Serial.println("Modo de troca de senha.");
  Serial.println("Aproxime a tag do usuario.");

  estado = LENDO_TAG_TROCA_SENHA;
}

// =====================================================
// 9. PROCESSAR MENU SERIAL
// =====================================================

void processarMenu() {
  if (!Serial.available()) {
    return;
  }

  String opcao = Serial.readStringUntil('\n');
  opcao.trim();

  if (opcao == "1") {
    iniciarCadastro();
  } else if (opcao == "2") {
    iniciarPonto();
  } else if (opcao == "3") {
    listarUsuarios();
    mostrarMenu();
  } else if (opcao == "4") {
    iniciarAlteracaoNome();
  } else if (opcao == "5") {
    iniciarTrocaSenha();
  } else if (opcao == "6") {
    consultarHistorico();
    mostrarMenu();
  } else if (opcao == "7") {
    Serial.println("Data/hora: " + obterDataHora());
    mostrarMenu();
  } else {
    Serial.println("Opcao invalida.");
    mostrarMenu();
  }
}

// =====================================================
// 10. CADASTRO DE USUARIO
// =====================================================

void processarCadastro() {
  String uid;

  if (!lerTag(uid)) {
    return;
  }

  if (usuarioExiste(uid)) {
    Serial.println(
      "Esta tag ja esta cadastrada. "
      "Use Alterar nome ou Trocar senha."
    );

    bip(400);
    estado = MENU;
    mostrarMenu();
    return;
  }

  uidTemporario = uid;
  senhaDigitada = "";

  Serial.println("Nome: " + nomeTemporario);
  Serial.println("Agora digite a senha pelo teclado.");
  Serial.println("* apaga | # confirma");

  estado = DIGITANDO_SENHA_CADASTRO;

  bip(100);
}

// =====================================================
// 11. ALTERAR NOME
// =====================================================

void processarAlteracaoNome() {
  String uid;

  if (!lerTag(uid)) {
    return;
  }

  String nomeAntigo;
  String hashAntigo;

  if (!obterUsuario(uid, nomeAntigo, hashAntigo)) {
    Serial.println("Usuario nao encontrado.");
    estado = MENU;
    mostrarMenu();
    return;
  }

  int indice = random(QUANTIDADE_NOMES);
  String novoNome = NOMES_TESTE[indice];

  // Evita que o nome novo seja igual ao antigo.
  if (novoNome == nomeAntigo) {
    novoNome = "Usuario Atualizado";
  }

  if (salvarUsuario(uid, novoNome, hashAntigo)) {
    Serial.println("Nome anterior: " + nomeAntigo);
    Serial.println("Novo nome: " + novoNome);
    Serial.println("Alteracao salva na memoria local.");
    bip(200);
  } else {
    Serial.println("Erro ao alterar nome.");
  }

  estado = MENU;
  mostrarMenu();
}

// =====================================================
// 12. TROCAR SENHA
// =====================================================

void processarInicioTrocaSenha() {
  String uid;

  if (!lerTag(uid)) {
    return;
  }

  String nome;
  String hash;

  if (!obterUsuario(uid, nome, hash)) {
    Serial.println("Usuario nao encontrado.");
    estado = MENU;
    mostrarMenu();
    return;
  }

  uidTemporario = uid;
  senhaDigitada = "";

  Serial.println("Usuario: " + nome);
  Serial.println("Digite a nova senha.");
  Serial.println("* apaga | # confirma");

  estado = DIGITANDO_NOVA_SENHA;
}

// =====================================================
// 13. TECLADO: CADASTRO E TROCA DE SENHA
// =====================================================

void processarSenha() {
  char tecla = keypad.getKey();

  if (!tecla) {
    return;
  }

  if (tecla == '*') {
    senhaDigitada = "";
    Serial.println();
    Serial.println("Senha apagada.");
    return;
  }

  if (tecla >= '0' && tecla <= '9') {
    if (senhaDigitada.length() < 16) {
      senhaDigitada += tecla;
      Serial.print("*");
    }
    return;
  }

  if (tecla != '#') {
    return;
  }

  if (senhaDigitada.length() == 0) {
    Serial.println();
    Serial.println("Digite uma senha primeiro.");
    return;
  }

  String hash = calcularHash(
    uidTemporario,
    senhaDigitada
  );

  bool sucesso = false;

  if (estado == DIGITANDO_SENHA_CADASTRO) {
    sucesso = salvarUsuario(
      uidTemporario,
      nomeTemporario,
      hash
    );
  } else if (estado == DIGITANDO_NOVA_SENHA) {
    String nomeAtual;
    String hashAtual;

    if (obterUsuario(uidTemporario, nomeAtual, hashAtual)) {
      sucesso = salvarUsuario(
        uidTemporario,
        nomeAtual,
        hash
      );
    }
  }

  senhaDigitada = "";

  if (sucesso) {
    Serial.println();
    Serial.println("Operacao concluida com sucesso!");
    Serial.println("Dados salvos na memoria persistente.");
    bip(200);
  } else {
    Serial.println();
    Serial.println("Falha ao salvar os dados.");
    bip(500);
  }

  estado = MENU;
  mostrarMenu();
}

// =====================================================
// 14. REGISTRAR PONTO
// =====================================================

void processarPonto() {
  String uid;

  if (!lerTag(uid)) {
    return;
  }

  String nome;
  String hash;

  if (!obterUsuario(uid, nome, hash)) {
    Serial.println("Tag nao cadastrada.");
    bip(400);
    estado = MENU;
    mostrarMenu();
    return;
  }

  // Este menu de teste registra ponto apenas com a tag.
  // Em producao, adicione a validacao de senha se exigida.
  registrarPonto(uid, nome);

  estado = MENU;
  mostrarMenu();
}

// =====================================================
// 15. SETUP
// =====================================================

void setup() {
  Serial.begin(115200);

  // Permite que readStringUntil use timeout curto.
  Serial.setTimeout(100);

  pinMode(BUZZER_PIN, OUTPUT);
  digitalWrite(BUZZER_PIN, LOW);

  randomSeed(esp_random());

  SPI.begin();
  rfid.PCD_Init();

  // Namespace persistente.
  memoria.begin("ponto", false);

  Serial.println();
  Serial.println("Inicializando ponto eletronico...");

  // O sistema nao fica preso esperando Wi-Fi.
  WiFi.mode(WIFI_STA);
  WiFi.begin(WIFI_SSID, WIFI_PASSWORD);

  configTime(
    GMT_OFFSET,
    DAYLIGHT_OFFSET,
    "pool.ntp.org",
    "time.google.com"
  );

  Serial.println("Sistema pronto.");
  Serial.println("O cadastro e os pontos funcionam offline.");

  mostrarMenu();
}

// =====================================================
// 16. LOOP PRINCIPAL
// =====================================================

void loop() {
  switch (estado) {
    case MENU:
      processarMenu();
      break;

    case LENDO_TAG_CADASTRO:
      processarCadastro();
      break;

    case LENDO_TAG_PONTO:
      processarPonto();
      break;

    case LENDO_TAG_NOME:
      processarAlteracaoNome();
      break;

    case LENDO_TAG_TROCA_SENHA:
      processarInicioTrocaSenha();
      break;

    case DIGITANDO_SENHA_CADASTRO:
    case DIGITANDO_NOVA_SENHA:
      processarSenha();
      break;
  }
}
