#include "comandos_serial.h"

#include <ArduinoJson.h>
#include <WiFi.h>

#include "cartao_sd.h"
#include "config.h"
#include "display.h"
#include "fila_offline.h"
#include "funcionarios.h"
#include "perifericos.h"
#include "ponto.h"
#include "rede.h"
#include "relogio.h"

void mostrarAjuda() {
  Serial.println();
  Serial.println("================================");
  Serial.println("       PONTO ELETRONICO");
  Serial.println("================================");
  Serial.println("Uso normal: aproxime o cartao e");
  Serial.println("digite a senha no teclado.");
  Serial.println();
  Serial.println("Testes e diagnostico:");
  Serial.println("  tag AA:BB:CC:DD  simula um cartao");
  Serial.println("  pin 1234         simula a senha");
  Serial.println("  status           hardware, rede, hora e fila");
  Serial.println("  fila             lista pontos guardados sem conexao");
  Serial.println("  sync             envia a fila agora");
  Serial.println("  limpar-fila      apaga a fila (os pontos se perdem!)");
  Serial.println("  funcionarios     lista salva no relogio");
  Serial.println("  atualizar        baixa a lista de funcionarios agora");
  Serial.println("  hora             mostra data e hora");
  Serial.println("  ajuda            mostra esta tela");
  Serial.println();
  Serial.println("Configuracao (fica salva; tambem pode vir do config.txt no SD):");
  Serial.println("  wifi             pergunta nome e senha da rede");
  Serial.println("  servidor URL     ex.: servidor http://192.168.0.10:8080/api/v1");
  Serial.println("  chave CHAVE      DEVICE_API_KEY do backend");
  Serial.println("  config           mostra a configuracao atual");
  Serial.println("  reset-config     volta aos valores padrao");
  Serial.println("================================");
}

static String okOuFalta(bool ok) {
  return ok ? "OK" : "NAO ENCONTRADO";
}

void mostrarStatus() {
  JsonDocument resposta;
  int status = api("GET", "/device/ping", "", &resposta);

  String wifi = wifiConectado()
    ? "conectado (" + WiFi.localIP().toString() + ", sinal " + String(WiFi.RSSI()) + " dBm)"
    : String("desconectado");
  String servidor = status == 200 ? String("online") : "offline (" + mensagemDeErro(status, resposta) + ")";

  Serial.println();
  Serial.println("============ STATUS ============");
  Serial.println("Leitor RFID: " + okOuFalta(rfidFuncionando()));
  Serial.println("LCD: " + okOuFalta(displayPresente()));
  Serial.println("RTC DS3231: " + okOuFalta(rtcPresente()));
  Serial.println("Cartao SD: " + sdDescricao());
  Serial.println("Wi-Fi: " + configRede().wifiSsid + " - " + wifi);
  Serial.println("API: " + configRede().apiUrl);
  Serial.println("Servidor: " + servidor);
  Serial.println("Data/hora: " + dataHoraAtual() + " (fonte: " + relogioFonte() + ")");
  Serial.println(
    "Funcionarios na memoria: " +
    (listaCarregada() ? String(totalFuncionarios()) : String("lista ainda nao baixada"))
  );
  Serial.println("Fila offline: " + String(filaTamanho()) + " ponto(s)");
  Serial.println("================================");
}

static void mostrarConfiguracao() {
  const ConfigRede& config = configRede();

  Serial.println();
  Serial.println("========= CONFIGURACAO =========");
  Serial.println("Wi-Fi: " + config.wifiSsid);
  Serial.println("Servidor: " + config.apiUrl);
  Serial.println("Chave: " + config.chave);
  Serial.println("================================");
}

// Espera uma linha no Serial (usado so nos comandos de configuracao).
static String lerLinhaSerial(unsigned long timeoutMs) {
  unsigned long prazo = millis() + timeoutMs;

  while ((long)(millis() - prazo) < 0) {
    if (Serial.available()) {
      String linha = Serial.readStringUntil('\n');
      linha.trim();
      return linha;
    }
    delay(10);
  }

  return "";
}

static void configurarWifi() {
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

  ConfigRede nova;
  nova.wifiSsid = ssid;
  nova.wifiSenha = senha;
  salvarConfigRede(nova);
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
    } else if (!pontoOcioso()) {
      Serial.println("Termine a operacao atual ou cancele com D.");
    } else {
      simularCartao(argumento);
    }
  } else if (comando == "pin") {
    if (!aguardandoSenha()) {
      Serial.println("Nenhum cartao aguardando senha.");
    } else if (argumento.length() != (unsigned int)TAMANHO_SENHA) {
      Serial.println("A senha tem " + String(TAMANHO_SENHA) + " numeros.");
    } else {
      pontoDigitarSenha(argumento);
    }
  } else if (comando == "status") {
    mostrarStatus();
  } else if (comando == "fila") {
    filaListar();
  } else if (comando == "sync") {
    filaEnviar(true);
  } else if (comando == "limpar-fila") {
    filaApagar();
    Serial.println("Fila offline apagada.");
  } else if (comando == "funcionarios") {
    funcionariosListar();
  } else if (comando == "atualizar") {
    Serial.println(funcionariosSincronizar() ? "Lista conferida com o servidor." : "Servidor indisponivel.");
  } else if (comando == "wifi") {
    configurarWifi();
  } else if (comando == "servidor") {
    if (!argumento.startsWith("http://") && !argumento.startsWith("https://")) {
      Serial.println("Uso: servidor http://IP_DO_COMPUTADOR:8080/api/v1");
    } else {
      ConfigRede nova;
      nova.apiUrl = argumento;
      salvarConfigRede(nova);
      Serial.println("Servidor salvo: " + configRede().apiUrl);
      mostrarStatus();
    }
  } else if (comando == "chave") {
    if (argumento.length() == 0) {
      Serial.println("Uso: chave MINHA_CHAVE");
    } else {
      ConfigRede nova;
      nova.chave = argumento;
      salvarConfigRede(nova);
      Serial.println("Chave salva.");

      // Pontos retidos pela chave errada saem agora, sem esperar 30 s.
      if (filaTamanho() > 0) {
        filaEnviar(true);
      }
    }
  } else if (comando == "config") {
    mostrarConfiguracao();
  } else if (comando == "reset-config") {
    restaurarConfigPadrao();
    Serial.println("Configuracao padrao restaurada.");
  } else if (comando == "hora") {
    Serial.println("Data/hora: " + dataHoraAtual() + " (fonte: " + relogioFonte() + ")");
  } else if (comando == "ajuda" || comando == "help" || comando == "menu") {
    mostrarAjuda();
  } else {
    Serial.println("Comando desconhecido. Digite ajuda.");
  }
}
