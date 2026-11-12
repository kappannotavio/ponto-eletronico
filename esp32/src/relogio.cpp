#include "relogio.h"

#include <RTClib.h>
#include <Wire.h>
#include <esp_sntp.h>
#include <sys/time.h>

#include "config.h"

// Qualquer data depois de nov/2023 indica hora valida.
static const time_t HORA_VALIDA_MINIMA = 1700000000;

static RTC_DS3231 rtc;
static bool rtcOk = false;

enum FonteHora { FONTE_NENHUMA, FONTE_RTC, FONTE_SERVIDOR, FONTE_NTP };
static FonteHora fonte = FONTE_NENHUMA;

static volatile bool ntpAcabouDeSincronizar = false;
static unsigned long ultimoAcertoServidor = 0;
static bool acertouPeloServidor = false;

static void aoSincronizarNtp(struct timeval*) {
  // Roda na task do SNTP: so marca, o loop principal grava no RTC.
  ntpAcabouDeSincronizar = true;
}

static void gravarRtc() {
  if (!rtcOk || !relogioSincronizado()) {
    return;
  }
  // O RTC guarda UTC; o fuso e aplicado so na hora de mostrar.
  rtc.adjust(DateTime((uint32_t)time(nullptr)));
}

void relogioIniciar() {
  // Define o fuso e liga o NTP (so funciona quando houver internet).
  sntp_set_time_sync_notification_cb(aoSincronizarNtp);
  configTzTime(FUSO_HORARIO, "pool.ntp.org", "time.google.com");

  rtcOk = rtc.begin(&Wire);

  if (!rtcOk) {
    Serial.println("RTC DS3231 NAO encontrado (I2C 0x68). Sem rede, a hora se perde ao reiniciar.");
    return;
  }

  if (rtc.lostPower()) {
    Serial.println("RTC sem hora (bateria fraca ou primeira vez). Sera acertado pela rede.");
    return;
  }

  uint32_t agora = rtc.now().unixtime();
  if ((time_t)agora < HORA_VALIDA_MINIMA) {
    Serial.println("RTC com data invalida. Sera acertado pela rede.");
    return;
  }

  struct timeval tv = {};
  tv.tv_sec = (time_t)agora;
  settimeofday(&tv, nullptr);
  fonte = FONTE_RTC;

  Serial.println("Hora carregada do RTC: " + dataHoraAtual());
}

void relogioAtualizar() {
  if (!ntpAcabouDeSincronizar) {
    return;
  }

  ntpAcabouDeSincronizar = false;
  fonte = FONTE_NTP;
  gravarRtc();
}

bool relogioSincronizado() {
  return time(nullptr) > HORA_VALIDA_MINIMA;
}

bool rtcPresente() {
  return rtcOk;
}

bool relogioPrecisaDoServidor() {
  if (fonte == FONTE_NTP) {
    return false;
  }
  if (!relogioSincronizado() || !acertouPeloServidor) {
    return true;
  }
  return millis() - ultimoAcertoServidor >= INTERVALO_HORA_SERVIDOR;
}

void relogioAcertarPeloServidor(time_t unixUtc) {
  if (unixUtc < HORA_VALIDA_MINIMA || fonte == FONTE_NTP) {
    return;
  }

  struct timeval tv = {};
  tv.tv_sec = unixUtc;
  settimeofday(&tv, nullptr);

  fonte = FONTE_SERVIDOR;
  acertouPeloServidor = true;
  ultimoAcertoServidor = millis();
  gravarRtc();

  Serial.println("Hora acertada pelo servidor: " + dataHoraAtual());
}

String relogioFonte() {
  switch (fonte) {
    case FONTE_NTP: return "NTP";
    case FONTE_SERVIDOR: return "servidor";
    case FONTE_RTC: return "RTC";
    default: return "nenhuma";
  }
}

static String formatar(const char* formato, bool utc) {
  if (!relogioSincronizado()) {
    return "";
  }

  time_t agora = time(nullptr);
  struct tm info;
  if (utc) {
    gmtime_r(&agora, &info);
  } else {
    localtime_r(&agora, &info);
  }

  char buffer[32];
  strftime(buffer, sizeof(buffer), formato, &info);
  return String(buffer);
}

String dataHoraAtual() {
  String texto = formatar("%d/%m/%Y %H:%M:%S", false);
  return texto.length() > 0 ? texto : String("HORA NAO SINCRONIZADA");
}

String horaAtual() {
  return formatar("%H:%M:%S", false);
}

String horaMinutoAtual() {
  return formatar("%H:%M", false);
}

String dataAtual() {
  return formatar("%Y-%m-%d", false);
}

String mesAtual() {
  return formatar("%Y-%m", false);
}

String instanteIsoUtc() {
  return formatar("%Y-%m-%dT%H:%M:%SZ", true);
}

String horaDeIso(const String& iso) {
  if (iso.length() < 16) {
    return iso;
  }
  return iso.substring(11, 16);
}
