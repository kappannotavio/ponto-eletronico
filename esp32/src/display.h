// LCD 16x2 com modulo I2C. Sem LCD ligado, as funcoes nao fazem nada
// (as mensagens continuam aparecendo no monitor serial).
#pragma once

#include <Arduino.h>

void displayIniciar();
bool displayPresente();

// Mostra duas linhas. Com duracaoMs > 0, volta sozinho para a tela
// inicial depois desse tempo (quando o relogio estiver ocioso).
void displayMostrar(const String& linha1, const String& linha2 = "", unsigned long duracaoMs = 0);

// Atualiza a tela inicial (hora e situacao). Chamar no loop.
void displayAtualizar(bool ocioso);

// Remove acentos (o LCD nao tem esses caracteres) e corta no tamanho da linha.
String textoParaLcd(const String& texto);
