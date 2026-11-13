// =====================================================
// CONFIGURACAO GERAL DO RELOGIO DE PONTO
// =====================================================
//
// Pinos conforme o esquema de ligacao (ESP32 DevKit):
//
//   RFID (MFRC522)   SDA->D5  SCK->D18  MOSI->D23  MISO->D19  RST->D27  3.3V
//   Cartao SD        CS->D4   SCK->D14  MOSI->D13  MISO->D15            3.3V
//   LCD 16x2 I2C     SDA->D21 SCL->D22                                  VIN (5V)
//   RTC DS3231       SDA->D21 SCL->D22                                  3.3V
//   Teclado 4x4      R1->VP(36) R2->VN(39) R3->D34 R4->D35
//                    C1->D32 C2->D33 C3->D25 C4->D26
//                    resistor de 10k de cada linha (36, 39, 34, 35) para 3V3
//   Buzzer (opc.)    D17
//
// O RFID usa o barramento SPI padrao (VSPI) e o SD o segundo (HSPI),
// cada um nos seus pinos, para um nao atrapalhar o outro.
// O LCD e o RTC dividem o mesmo barramento I2C; cada um tem seu endereco.
// =====================================================

#pragma once

#include <Arduino.h>

// ---------- Pinos ----------

// RFID (SPI padrao / VSPI)
const uint8_t PINO_SPI_SCK = 18;
const uint8_t PINO_SPI_MISO = 19;
const uint8_t PINO_SPI_MOSI = 23;
const uint8_t PINO_RFID_SS = 5;
const uint8_t PINO_RFID_RST = 27;

// Cartao SD (segundo SPI / HSPI)
const uint8_t PINO_SD_SCK = 14;
const uint8_t PINO_SD_MISO = 15;
const uint8_t PINO_SD_MOSI = 13;
const uint8_t PINO_SD_CS = 4;

const uint8_t PINO_I2C_SDA = 21;
const uint8_t PINO_I2C_SCL = 22;

// Buzzer ativo (opcional). Sem buzzer ligado, o pino fica sem uso.
const uint8_t PINO_BUZZER = 17;

// As linhas do teclado estao nos pinos 34-39, que so funcionam como entrada
// e nao tem pull-up interno: por isso os resistores de 10k para 3V3.
// As colunas precisam ser pinos de saida.
const uint8_t TECLADO_LINHAS = 4;
const uint8_t TECLADO_COLUNAS = 4;
const uint8_t PINOS_LINHAS[TECLADO_LINHAS] = {36, 39, 34, 35};
const uint8_t PINOS_COLUNAS[TECLADO_COLUNAS] = {32, 33, 25, 26};

// ---------- Display ----------

const uint8_t LCD_COLUNAS = 16;
const uint8_t LCD_LINHAS = 2;
// Enderecos comuns do modulo I2C do LCD (PCF8574 e PCF8574A).
// O firmware testa os dois e usa o que responder.
const uint8_t LCD_ENDERECOS[] = {0x27, 0x3F};

// ---------- Rede (valores padrao) ----------
//
// Os valores reais vem do arquivo config.txt no cartao SD ou dos comandos
// do monitor serial (wifi, servidor, chave) e ficam salvos na memoria.
// Os padroes abaixo servem para a simulacao no Wokwi.

const char* const WIFI_SSID_PADRAO = "Wokwi-GUEST";
const char* const WIFI_SENHA_PADRAO = "";
const char* const API_URL_PADRAO = "http://host.wokwi.internal:8080/api/v1";
const char* const CHAVE_PADRAO = "dev-device-key";

// Fuso horario (POSIX): Brasilia, UTC-3, sem horario de verao.
const char* const FUSO_HORARIO = "<-03>3";

// ---------- Regras do ponto ----------

const int TAMANHO_SENHA = 4;
const int MAX_TENTATIVAS_SENHA = 3;

// Pontos guardados sem conexao (memoria interna). ~80 bytes cada.
const int MAX_FILA_OFFLINE = 2000;

// ---------- Tempos (ms) ----------

const unsigned long TEMPO_DIGITAR_SENHA = 15000;
const unsigned long TEMPO_LEITURA_CADASTRO = 60000;
const unsigned long TEMPO_MENSAGEM = 3000;
const unsigned long INTERVALO_PEDIDO_CADASTRO = 2000;
const unsigned long INTERVALO_SEM_SERVIDOR = 15000;
const unsigned long INTERVALO_ENVIO_FILA = 30000;
const unsigned long INTERVALO_LISTA_FUNCIONARIOS = 60000;
const unsigned long INTERVALO_TENTAR_WIFI = 10000;
const unsigned long INTERVALO_VERIFICAR_HARDWARE = 60000;
// Sem NTP, acerta a hora pelo servidor nesse intervalo (corrige o RTC).
const unsigned long INTERVALO_HORA_SERVIDOR = 3600000UL;
