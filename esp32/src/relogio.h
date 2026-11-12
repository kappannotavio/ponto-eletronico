// Data e hora: RTC DS3231 (bateria), NTP (internet) e hora do servidor.
//
// Ordem de confianca:
//   1. NTP, quando ha internet;
//   2. hora do backend (GET /device/ping), quando nao ha NTP;
//   3. RTC DS3231, que mantem a hora com o ESP desligado ou sem rede.
// Sempre que a hora vem da rede, o RTC e acertado.
#pragma once

#include <Arduino.h>
#include <time.h>

void relogioIniciar();
void relogioAtualizar();

bool relogioSincronizado();
bool rtcPresente();

// true quando vale a pena pedir a hora ao servidor (sem NTP ha muito tempo).
bool relogioPrecisaDoServidor();
void relogioAcertarPeloServidor(time_t unixUtc);

// De onde veio a hora atual: "NTP", "servidor", "RTC" ou "nenhuma".
String relogioFonte();

String dataHoraAtual();   // "12/11/2026 14:05:32"
String horaAtual();       // "14:05:32"
String horaMinutoAtual(); // "14:05"
String dataAtual();       // "2026-11-12"
String mesAtual();        // "2026-11"
String instanteIsoUtc();  // "2026-11-12T17:05:32Z"

// "2026-11-06T08:01:12-03:00" -> "08:01"
String horaDeIso(const String& iso);
