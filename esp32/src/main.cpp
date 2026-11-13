// =====================================================
// PONTO ELETRONICO - ESP32
// =====================================================
//
// Relogio de ponto: o funcionario aproxima o cartao RFID, ve o nome no LCD,
// digita a senha de 4 numeros e o ponto vai para o backend. Sem conexao,
// o ponto fica guardado no relogio e e enviado quando o servidor voltar.
//
// Arquivos:
//   config.h            pinos, tempos e valores padrao
//   ponto.cpp           fluxo de bater ponto e de ler cartao para cadastro
//   funcionarios.cpp    lista local de funcionarios (nome + senha)
//   fila_offline.cpp    pontos guardados sem conexao (memoria interna)
//   cartao_sd.cpp       historico em CSV e config.txt no cartao SD
//   rede.cpp            Wi-Fi, configuracao salva e chamadas a API
//   relogio.cpp         hora: RTC DS3231, NTP e servidor
//   display.cpp         LCD 16x2 I2C
//   perifericos.cpp     leitor RFID, teclado e buzzer
//   comandos_serial.cpp comandos do monitor serial
// =====================================================

#include <Arduino.h>
#include <ArduinoJson.h>
#include <LittleFS.h>
#include <SPI.h>
#include <Wire.h>

#include "cartao_sd.h"
#include "comandos_serial.h"
#include "config.h"
#include "display.h"
#include "fila_offline.h"
#include "funcionarios.h"
#include "perifericos.h"
#include "ponto.h"
#include "rede.h"
#include "relogio.h"

static bool memoriaOk = false;
static bool wifiEstavaConectado = false;

static unsigned long proximoEnvioFila = 0;
static unsigned long proximaLista = 0;
static unsigned long proximaVerificacao = 0;

static bool passou(unsigned long instante) {
  return (long)(millis() - instante) >= 0;
}

static void acertarHoraPeloServidor() {
  JsonDocument resposta;
  if (api("GET", "/device/ping", "", &resposta) == 200) {
    relogioAcertarPeloServidor((time_t)(resposta["unixTime"] | 0LL));
  }
}

// Tarefas que so rodam com o relogio parado na tela inicial,
// para nao atrasar quem esta batendo o ponto.
static void tarefasEmSegundoPlano() {
  sdManter();

  if (passou(proximaVerificacao)) {
    proximaVerificacao = millis() + INTERVALO_VERIFICAR_HARDWARE;
    verificarRfid();
  }

  bool conectado = wifiConectado();
  if (conectado && !wifiEstavaConectado) {
    // Acabou de conectar: sincroniza tudo agora.
    proximoEnvioFila = millis();
    proximaLista = millis();
  }
  wifiEstavaConectado = conectado;

  if (!conectado || !memoriaOk) {
    return;
  }

  if (passou(proximaLista)) {
    proximaLista = millis() + INTERVALO_LISTA_FUNCIONARIOS;
    funcionariosSincronizar();
  }

  if (passou(proximoEnvioFila)) {
    proximoEnvioFila = millis() + INTERVALO_ENVIO_FILA;

    if (relogioPrecisaDoServidor()) {
      acertarHoraPeloServidor();
    }
    if (filaTamanho() > 0) {
      filaEnviar(false);
    }
  }
}

static void mostrarDiagnostico() {
  String linha1 = String("RFID ") + (rfidFuncionando() ? "ok" : "ERRO") +
                  " SD " + (sdPresente() ? "ok" : "--");
  String linha2 = String("RTC ") + (rtcPresente() ? "ok" : "--") +
                  " MEM " + (memoriaOk ? "ok" : "ERRO");

  displayMostrar(linha1, linha2, TEMPO_MENSAGEM);
}

void setup() {
  Serial.begin(115200);
  Serial.setTimeout(100);

  Serial.println();
  Serial.println("Inicializando ponto eletronico...");

  // CS do RFID e do SD em nivel alto antes de iniciar: nenhum dos dois
  // responde antes da hora.
  pinMode(PINO_RFID_SS, OUTPUT);
  digitalWrite(PINO_RFID_SS, HIGH);
  pinMode(PINO_SD_CS, OUTPUT);
  digitalWrite(PINO_SD_CS, HIGH);

  // SPI do RFID. O SD inicia o barramento dele em sdIniciar().
  SPI.begin(PINO_SPI_SCK, PINO_SPI_MISO, PINO_SPI_MOSI);
  Wire.begin(PINO_I2C_SDA, PINO_I2C_SCL);

  displayIniciar();
  displayMostrar("Ponto eletronico", "Iniciando...");

  relogioIniciar();

  // Memoria interna para a fila offline e a lista de funcionarios.
  // Na primeira vez a particao e formatada (true).
  memoriaOk = LittleFS.begin(true);
  if (!memoriaOk) {
    Serial.println("ERRO: memoria interna (LittleFS) nao montou. Pontos offline nao serao salvos.");
  }

  sdIniciar();

  redeIniciar();

  // config.txt no SD tem prioridade: e o jeito facil de configurar o Wi-Fi.
  ConfigRede doCartao;
  if (sdLerConfig(doCartao)) {
    if (salvarConfigRede(doCartao)) {
      Serial.println("Configuracao atualizada pelo config.txt do cartao SD.");
    }
  }

  perifericosIniciar();

  if (memoriaOk) {
    funcionariosIniciar();
    filaIniciar();
  }

  pontoIniciar();
  mostrarAjuda();
  mostrarDiagnostico();
}

void loop() {
  redeManter();
  relogioAtualizar();
  processarSerial();
  pontoAtualizar();

  bool ocioso = pontoOcioso();
  if (ocioso) {
    tarefasEmSegundoPlano();
  }

  displayAtualizar(ocioso);
}
