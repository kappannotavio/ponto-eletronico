// Fluxo do relogio: bater ponto (cartao + senha) e ler cartao para cadastro.
//
// Estados:
//   AGUARDANDO_CARTAO  tela inicial: aproximar o cartao comeca a batida
//   DIGITANDO_SENHA    cartao lido, esperando os 4 numeros no teclado
//   LENDO_CADASTRO     o painel pediu para ler um cartao novo
#pragma once

#include <Arduino.h>

void pontoIniciar();

// Chamar no loop: le cartao/teclado e cuida dos tempos limite.
void pontoAtualizar();

bool pontoOcioso();

// Usados pelos comandos do monitor serial.
bool aguardandoSenha();
void pontoDigitarSenha(const String& senha);
