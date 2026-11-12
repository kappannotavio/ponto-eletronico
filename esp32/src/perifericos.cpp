#include "perifericos.h"

#include <Keypad.h>
#include <MFRC522.h>
#include <SPI.h>

#include "config.h"

static MFRC522 rfid(PINO_RFID_SS, PINO_RFID_RST);

static char teclas[TECLADO_LINHAS][TECLADO_COLUNAS] = {
  {'1', '2', '3', 'A'},
  {'4', '5', '6', 'B'},
  {'7', '8', '9', 'C'},
  {'*', '0', '#', 'D'}
};

// A biblioteca Keypad recebe ponteiros nao-const.
static byte pinosLinhas[TECLADO_LINHAS];
static byte pinosColunas[TECLADO_COLUNAS];

static Keypad* teclado = nullptr;

static String cartaoSimulado = "";
static bool rfidOk = false;

static bool lerVersaoRfid(byte& versao) {
  versao = rfid.PCD_ReadRegister(MFRC522::VersionReg);
  // 0x00 ou 0xFF: nada respondendo no SPI (fio solto ou sem energia).
  return versao != 0x00 && versao != 0xFF;
}

void perifericosIniciar() {
  pinMode(PINO_BUZZER, OUTPUT);
  digitalWrite(PINO_BUZZER, LOW);

  for (int i = 0; i < TECLADO_LINHAS; i++) {
    pinosLinhas[i] = PINOS_LINHAS[i];
  }
  for (int i = 0; i < TECLADO_COLUNAS; i++) {
    pinosColunas[i] = PINOS_COLUNAS[i];
  }

  teclado = new Keypad(makeKeymap(teclas), pinosLinhas, pinosColunas, TECLADO_LINHAS, TECLADO_COLUNAS);

  // SPI.begin() ja foi chamado em main.cpp (barramento dividido com o SD).
  rfid.PCD_Init();

  byte versao;
  rfidOk = lerVersaoRfid(versao);

  if (rfidOk) {
    Serial.printf("Leitor RFID OK (versao 0x%02X)\n", versao);
  } else {
    Serial.println("Leitor RFID NAO encontrado. Confira os fios (SDA=D5, RST=D27, 3.3V).");
  }
}

bool rfidFuncionando() {
  return rfidOk;
}

// O MFRC522 as vezes trava depois de muito tempo ligado ou de um pico de
// energia. Se parar de responder, reinicia o leitor.
void verificarRfid() {
  byte versao;

  if (lerVersaoRfid(versao)) {
    if (!rfidOk) {
      Serial.println("Leitor RFID voltou a responder.");
    }
    rfidOk = true;
    return;
  }

  rfid.PCD_Init();
  bool voltou = lerVersaoRfid(versao);

  if (rfidOk && !voltou) {
    Serial.println("Leitor RFID parou de responder.");
  }
  rfidOk = voltou;
}

static String uidLido() {
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

void simularCartao(const String& uid) {
  cartaoSimulado = uid;
}

bool lerCartao(String& uid) {
  if (cartaoSimulado.length() > 0) {
    uid = cartaoSimulado;
    cartaoSimulado = "";
    Serial.println("Cartao simulado: " + uid);
    return true;
  }

  if (!rfid.PICC_IsNewCardPresent() || !rfid.PICC_ReadCardSerial()) {
    return false;
  }

  uid = uidLido();

  rfid.PICC_HaltA();
  rfid.PCD_StopCrypto1();

  Serial.println("Cartao lido: " + uid);
  return true;
}

char lerTecla() {
  return teclado->getKey();
}

void bip(int duracaoMs) {
  digitalWrite(PINO_BUZZER, HIGH);
  delay(duracaoMs);
  digitalWrite(PINO_BUZZER, LOW);
}

void bipEntrada() {
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

void bipAviso() {
  bip(80);
  delay(80);
  bip(80);
}
