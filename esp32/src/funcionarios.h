// Copia local da lista de funcionarios (GET /device/employees).
//
// Com ela o relogio mostra o nome na hora, recusa cartao desconhecido ou
// desativado e confere a senha mesmo sem conexao. Fica na memoria flash
// interna (LittleFS, /funcionarios.json) e e atualizada a cada minuto;
// o servidor responde 304 quando nada mudou.
#pragma once

#include <Arduino.h>

struct Funcionario {
  String cartao;
  String nome;
  bool ativo = false;
  // "salt$sha256(salt:pin)" em hexadecimal, igual ao backend.
  // Vazio quando o funcionario veio da consulta individual (sem senha).
  String hashSenha;
};

void funcionariosIniciar();

// Baixa a lista se mudou. Devolve false se o servidor nao respondeu.
bool funcionariosSincronizar();

// true se ja existe uma lista (baixada agora ou em outro dia).
bool listaCarregada();
int totalFuncionarios();

bool buscarFuncionario(const String& cartao, Funcionario& funcionario);
String nomeDoFuncionario(const String& cartao);

bool senhaConfere(const Funcionario& funcionario, const String& senha);

void funcionariosListar();
