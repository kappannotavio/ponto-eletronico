// Leitor RFID, teclado e buzzer.
#pragma once

#include <Arduino.h>

void perifericosIniciar();

// Le um cartao (ou o simulado pelo monitor serial). Devolve true e o UID
// no formato "AA:BB:CC:DD" quando ha cartao novo.
bool lerCartao(String& uid);

// Simula a leitura de um cartao (comando "tag" do monitor serial).
void simularCartao(const String& uid);

// Tecla pressionada ou '\0'.
char lerTecla();

// Confere se o leitor RFID responde e reinicia se travou.
bool rfidFuncionando();
void verificarRfid();

void bip(int duracaoMs = 100);
void bipEntrada();
void bipSaida();
void bipErro();
void bipAviso();
