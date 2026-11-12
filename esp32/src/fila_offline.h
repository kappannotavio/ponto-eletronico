// Fila de pontos batidos sem conexao com o servidor.
//
// Fica na memoria flash interna (LittleFS, arquivo /fila.jsonl), uma batida
// por linha, e nao se perde se o relogio desligar. Cada linha tem o mesmo
// corpo de POST /device/punches:
//   {"cardTag":"AA:BB:CC:DD","pin":"1234","recordedAt":"2026-11-12T11:02:00Z"}
#pragma once

#include <Arduino.h>

void filaIniciar();

int filaTamanho();
bool filaCheia();

// Devolve false se a fila estiver cheia ou a memoria falhar.
bool filaAdicionar(const String& cardTag, const String& pin, const String& recordedAt);

// Envia ao servidor, da mais antiga para a mais nova. Para no primeiro erro
// de conexao (o resto fica para a proxima). Mostra o resumo se detalhado.
void filaEnviar(bool detalhado);

void filaListar();
void filaApagar();
