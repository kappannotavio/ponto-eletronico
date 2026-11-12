// Cartao SD: historico de batidas (CSV) e arquivo de configuracao.
//
// O SD e um extra: se faltar ou der problema, o relogio continua batendo
// ponto normalmente (a fila e a lista de funcionarios ficam na memoria
// interna do ESP32).
//
//   /config.txt               Wi-Fi, servidor e chave (lido ao ligar)
//   /registros/AAAA-MM.csv    todas as batidas feitas neste relogio
#pragma once

#include <Arduino.h>

#include "rede.h"

void sdIniciar();
// Tenta montar o cartao de novo se ele foi colocado depois. Chamar no loop.
void sdManter();
bool sdPresente();
String sdDescricao();

// Le /config.txt. Devolve true se o arquivo existe (campos vazios = nao mudar).
bool sdLerConfig(ConfigRede& config);

// Acrescenta uma linha no historico do mes.
// situacao: "registrado", "offline", "enviado", "descartado", "recusado"...
void sdRegistrar(
  const String& cartao,
  const String& funcionario,
  const String& situacao,
  const String& detalhe
);
