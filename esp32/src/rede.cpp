#include "rede.h"

#include <HTTPClient.h>
#include <Preferences.h>
#include <WiFi.h>

#include "config.h"

static Preferences memoria;
static ConfigRede config;

static bool wifiEstavaConectado = false;
static bool ultimaRespostaOk = false;
static unsigned long proximaTentativaWifi = 0;

static void carregarConfig() {
  config.wifiSsid = memoria.getString("wifi_ssid", WIFI_SSID_PADRAO);
  config.wifiSenha = memoria.getString("wifi_senha", WIFI_SENHA_PADRAO);
  config.apiUrl = memoria.getString("api_url", API_URL_PADRAO);
  config.chave = memoria.getString("api_chave", CHAVE_PADRAO);
}

static void conectarWifi() {
  // Canal fixo acelera a conexao no Wokwi; em rede real o canal e automatico.
  int canal = config.wifiSsid == "Wokwi-GUEST" ? 6 : 0;

  WiFi.disconnect();
  WiFi.begin(config.wifiSsid.c_str(), config.wifiSenha.c_str(), canal);
  proximaTentativaWifi = millis() + INTERVALO_TENTAR_WIFI;

  Serial.println("Conectando ao Wi-Fi \"" + config.wifiSsid + "\"...");
}

void redeIniciar() {
  memoria.begin("ponto", false);
  carregarConfig();

  WiFi.mode(WIFI_STA);
  WiFi.setAutoReconnect(true);
  conectarWifi();
}

void redeManter() {
  bool conectado = wifiConectado();

  if (conectado != wifiEstavaConectado) {
    wifiEstavaConectado = conectado;

    if (conectado) {
      Serial.println("Wi-Fi conectado. IP: " + WiFi.localIP().toString());
    } else {
      ultimaRespostaOk = false;
      Serial.println("Wi-Fi desconectado. Os pontos serao guardados no relogio.");
    }
  }

  // Recomeca a conexao do zero (disconnect + begin). WiFi.reconnect() no meio
  // de uma tentativa gera o erro "wifi:sta is connecting, return error".
  if (!conectado && (long)(millis() - proximaTentativaWifi) >= 0) {
    conectarWifi();
  }
}

bool wifiConectado() {
  return WiFi.status() == WL_CONNECTED;
}

bool servidorOnline() {
  return wifiConectado() && ultimaRespostaOk;
}

const ConfigRede& configRede() {
  return config;
}

static bool salvarCampo(const char* chave, const String& valor, String& atual) {
  if (valor.length() == 0 || valor == atual) {
    return false;
  }
  memoria.putString(chave, valor);
  atual = valor;
  return true;
}

bool salvarConfigRede(const ConfigRede& nova) {
  String apiUrl = nova.apiUrl;
  while (apiUrl.endsWith("/")) {
    apiUrl.remove(apiUrl.length() - 1);
  }

  bool redeMudou = salvarCampo("wifi_ssid", nova.wifiSsid, config.wifiSsid);
  // A senha vazia e valida (rede aberta) quando o SSID foi informado junto.
  if (nova.wifiSsid.length() > 0 && nova.wifiSenha != config.wifiSenha) {
    memoria.putString("wifi_senha", nova.wifiSenha);
    config.wifiSenha = nova.wifiSenha;
    redeMudou = true;
  }

  bool servidorMudou = salvarCampo("api_url", apiUrl, config.apiUrl);
  bool chaveMudou = salvarCampo("api_chave", nova.chave, config.chave);

  if (redeMudou) {
    conectarWifi();
  }

  return redeMudou || servidorMudou || chaveMudou;
}

void restaurarConfigPadrao() {
  memoria.remove("wifi_ssid");
  memoria.remove("wifi_senha");
  memoria.remove("api_url");
  memoria.remove("api_chave");
  carregarConfig();
  conectarWifi();
}

int api(
  const char* metodo,
  const String& caminho,
  const String& corpo,
  JsonDocument* resposta,
  const String& etagEnviado,
  String* etagRecebido,
  String* corpoBruto
) {
  if (!wifiConectado()) {
    return API_SEM_WIFI;
  }

  HTTPClient http;
  http.setConnectTimeout(2000);
  http.setTimeout(4000);

  if (!http.begin(config.apiUrl + caminho)) {
    ultimaRespostaOk = false;
    return -1;
  }

  http.addHeader("X-Device-Key", config.chave);

  if (corpo.length() > 0) {
    http.addHeader("Content-Type", "application/json");
  }
  if (etagEnviado.length() > 0) {
    http.addHeader("If-None-Match", etagEnviado);
  }

  const char* cabecalhos[] = {"ETag"};
  if (etagRecebido != nullptr) {
    http.collectHeaders(cabecalhos, 1);
  }

  int status = http.sendRequest(metodo, corpo);
  ultimaRespostaOk = status > 0;

  if (status > 0) {
    if (etagRecebido != nullptr) {
      *etagRecebido = http.header("ETag");
    }

    if (resposta != nullptr || corpoBruto != nullptr) {
      String texto = http.getString();

      if (resposta != nullptr && texto.length() > 0) {
        deserializeJson(*resposta, texto);
      }
      if (corpoBruto != nullptr) {
        *corpoBruto = texto;
      }
    }
  }

  http.end();
  return status;
}

String mensagemDeErro(int status, JsonDocument& resposta) {
  if (status == API_SEM_WIFI) {
    return "sem Wi-Fi";
  }
  if (status <= 0) {
    return "servidor indisponivel";
  }

  String mensagem = resposta["error"] | "";
  if (mensagem.length() == 0) {
    mensagem = "erro HTTP " + String(status);
  }
  return mensagem;
}

bool chaveInvalida(int status, JsonDocument& resposta) {
  return status == 401 && resposta["code"] == "invalid_device_key";
}
