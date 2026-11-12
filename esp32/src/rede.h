// Wi-Fi, configuracao salva (rede/servidor/chave) e chamadas a API.
#pragma once

#include <Arduino.h>
#include <ArduinoJson.h>

struct ConfigRede {
  String wifiSsid;
  String wifiSenha;
  String apiUrl;
  String chave;
};

// Codigos devolvidos por api() quando nao houve resposta HTTP.
const int API_SEM_WIFI = -1000;

void redeIniciar();
void redeManter();

bool wifiConectado();
// Ultima chamada a API teve resposta (Wi-Fi ok e servidor no ar).
bool servidorOnline();

// Configuracao em uso (salva na NVS; padroes em config.h).
const ConfigRede& configRede();
// Salva so os campos nao vazios. Reconecta o Wi-Fi se a rede mudou.
// Devolve true se algo mudou.
bool salvarConfigRede(const ConfigRede& nova);
void restaurarConfigPadrao();

// Faz uma requisicao para apiUrl + caminho com o header X-Device-Key.
// Devolve o status HTTP (> 0) ou <= 0 se nao foi possivel falar com o servidor.
// etagEnviado/etagRecebido sao opcionais (If-None-Match / ETag).
int api(
  const char* metodo,
  const String& caminho,
  const String& corpo = "",
  JsonDocument* resposta = nullptr,
  const String& etagEnviado = "",
  String* etagRecebido = nullptr,
  String* corpoBruto = nullptr
);

// Mensagem de erro do backend ({"error": "..."}) ou uma descricao do status.
String mensagemDeErro(int status, JsonDocument& resposta);

// 401 por chave do relogio errada (e nao por senha errada do funcionario).
bool chaveInvalida(int status, JsonDocument& resposta);
