#include "ponto.h"

#include <ArduinoJson.h>

#include "cartao_sd.h"
#include "config.h"
#include "display.h"
#include "fila_offline.h"
#include "funcionarios.h"
#include "perifericos.h"
#include "rede.h"
#include "relogio.h"

enum Estado {
  AGUARDANDO_CARTAO,
  DIGITANDO_SENHA,
  LENDO_CADASTRO
};

static Estado estado = AGUARDANDO_CARTAO;

// Batida em andamento.
static Funcionario atual;
static bool funcionarioConhecido = false;
static String senha = "";
static int tentativas = 0;
static unsigned long prazo = 0;

// Leitura de cartao pedida pelo painel.
static String leituraId = "";
static unsigned long proximaConsulta = 0;

static bool passou(unsigned long instante) {
  return (long)(millis() - instante) >= 0;
}

static String nomeAtual() {
  return funcionarioConhecido && atual.nome.length() > 0 ? atual.nome : atual.cartao;
}

static void voltarAoInicio() {
  atual = Funcionario();
  funcionarioConhecido = false;
  senha = "";
  tentativas = 0;
  leituraId = "";
  estado = AGUARDANDO_CARTAO;

  Serial.println();
  Serial.println("Aproxime o cartao para bater o ponto.");
}

// Mostra a mensagem por alguns segundos e volta para a tela inicial.
static void encerrar(const String& linha1, const String& linha2) {
  displayMostrar(linha1, linha2, TEMPO_MENSAGEM);
  voltarAoInicio();
}

static void recusar(const String& linha1, const String& linha2, const String& motivo) {
  Serial.println("Ponto nao registrado: " + motivo);
  sdRegistrar(atual.cartao, funcionarioConhecido ? atual.nome : String(""), "recusado", motivo);
  bipErro();
  encerrar(linha1, linha2);
}

// "ENTRADA    08:02": texto a esquerda e hora a direita da linha do LCD.
static String linhaComHora(const String& texto, const String& hora) {
  if (hora.length() == 0) {
    return texto;
  }

  String linha = texto;
  do {
    linha += ' ';
  } while (linha.length() + hora.length() < LCD_COLUNAS);

  return linha + hora;
}

static void mostrarSenhaDigitada() {
  String asteriscos = "";
  for (unsigned int i = 0; i < senha.length(); i++) {
    asteriscos += '*';
  }
  displayMostrar(nomeAtual(), "Senha: " + asteriscos);
}

// =====================================================
// BATER PONTO
// =====================================================

static void iniciarBatida(const String& cartao) {
  atual = Funcionario();
  atual.cartao = cartao;
  funcionarioConhecido = buscarFuncionario(cartao, atual);

  if (!funcionarioConhecido) {
    // Pode ser alguem cadastrado ha menos de 1 minuto: pergunta ao servidor.
    JsonDocument resposta;
    int status = api("GET", "/device/employees/" + cartao, "", &resposta);

    if (status == 200) {
      atual.nome = resposta["fullName"] | "";
      atual.ativo = resposta["isActive"] | false;
      funcionarioConhecido = true;
    } else if (status == 404) {
      recusar("Cartao nao", "cadastrado", "cartao nao cadastrado");
      return;
    } else if (chaveInvalida(status, resposta)) {
      recusar("Chave do relogio", "invalida", "X-Device-Key invalido (confira a chave)");
      return;
    } else if (status > 0 && status < 500) {
      recusar("Erro no servidor", mensagemDeErro(status, resposta), mensagemDeErro(status, resposta));
      return;
    } else if (listaCarregada()) {
      // Sem servidor, mas a lista local diz que esse cartao nao existe.
      recusar("Cartao nao", "cadastrado", "cartao nao cadastrado (sem conexao)");
      return;
    } else {
      // Relogio novo, sem lista e sem servidor: aceita e o servidor confere depois.
      Serial.println("Sem lista de funcionarios e sem servidor: a senha sera conferida depois.");
    }
  }

  if (funcionarioConhecido && !atual.ativo) {
    recusar("Funcionario", "desativado", "funcionario desativado");
    return;
  }

  if (funcionarioConhecido) {
    Serial.println("Ola, " + atual.nome + "!");
  }
  Serial.println("Digite a senha de 4 numeros. * apaga | D cancela");

  senha = "";
  tentativas = 0;
  prazo = millis() + TEMPO_DIGITAR_SENHA;
  estado = DIGITANDO_SENHA;

  mostrarSenhaDigitada();
  bip(80);
}

static void senhaIncorreta() {
  tentativas++;
  bipErro();

  if (tentativas >= MAX_TENTATIVAS_SENHA) {
    Serial.println("Senha incorreta. Muitas tentativas.");
    sdRegistrar(atual.cartao, atual.nome, "recusado", "senha incorreta " + String(tentativas) + "x");
    encerrar("Senha incorreta", "Tente mais tarde");
    return;
  }

  Serial.println(
    "Senha incorreta. Tente novamente (" + String(tentativas) + "/" +
    String(MAX_TENTATIVAS_SENHA) + ")."
  );

  senha = "";
  prazo = millis() + TEMPO_DIGITAR_SENHA;
  displayMostrar("Senha incorreta", "Tente de novo " + String(tentativas) + "/" + String(MAX_TENTATIVAS_SENHA));
}

static void pontoRegistrado(JsonDocument& resposta) {
  bool saida = resposta["type"] == "exit";
  String nome = resposta["employeeName"] | nomeAtual().c_str();
  String hora = horaDeIso(resposta["record"]["recordedAt"] | "");
  int numero = resposta["punchNumber"] | 0;
  String tipo = saida ? "SAIDA" : "ENTRADA";

  Serial.println("******** PONTO REGISTRADO ********");
  Serial.println("Funcionario: " + nome);
  Serial.println("Tipo: " + tipo);
  Serial.println("Hora: " + hora);
  Serial.println("Batida do dia: " + String(numero));
  Serial.println("**********************************");

  sdRegistrar(atual.cartao, nome, "registrado", (saida ? "saida" : "entrada") + String(" (batida ") + String(numero) + " do dia)");

  if (saida) {
    bipSaida();
  } else {
    bipEntrada();
  }

  encerrar(linhaComHora(tipo, hora), nome);
}

static void salvarOffline(const String& registradoEm) {
  // Com a lista local, a senha e conferida aqui mesmo. Sem ela (relogio novo),
  // o servidor confere quando a fila for enviada.
  bool conferida = funcionarioConhecido && atual.hashSenha.length() > 0;

  if (conferida && !senhaConfere(atual, senha)) {
    senhaIncorreta();
    return;
  }

  if (!filaAdicionar(atual.cartao, senha, registradoEm)) {
    String motivo = filaCheia() ? "memoria cheia" : "erro ao gravar na memoria";
    Serial.println("PONTO NAO SALVO: " + motivo + ". Avise o RH.");
    sdRegistrar(atual.cartao, atual.nome, "perdido", motivo);
    bipErro();
    encerrar("NAO SALVO", filaCheia() ? "Memoria cheia" : "Erro de memoria");
    return;
  }

  String hora = horaMinutoAtual();

  Serial.println("******** PONTO SALVO NO RELOGIO ********");
  Serial.println("Funcionario: " + nomeAtual());
  Serial.println("Data/hora: " + dataHoraAtual());
  Serial.println("Sem conexao: sera enviado quando o servidor voltar.");
  Serial.println("Pendentes: " + String(filaTamanho()));
  Serial.println("****************************************");

  sdRegistrar(
    atual.cartao, atual.nome, "offline",
    conferida ? "senha conferida no relogio" : "senha sera conferida pelo servidor"
  );

  bipEntrada();

  // A tela inicial mostra "OFF" e quantos pontos estao guardados.
  encerrar(linhaComHora("PONTO OK", hora), nomeAtual());
}

static void enviarBatida() {
  String registradoEm = instanteIsoUtc();

  JsonDocument corpoDoc;
  corpoDoc["cardTag"] = atual.cartao;
  corpoDoc["pin"] = senha;
  if (registradoEm.length() > 0) {
    corpoDoc["recordedAt"] = registradoEm;
  }

  String corpo;
  serializeJson(corpoDoc, corpo);

  Serial.println();
  displayMostrar(nomeAtual(), "Registrando...");

  JsonDocument resposta;
  int status = api("POST", "/device/punches", corpo, &resposta);

  if (status == 201) {
    pontoRegistrado(resposta);
    return;
  }

  if (chaveInvalida(status, resposta)) {
    // Configuracao errada no relogio: guarda para nao perder a batida.
    Serial.println("X-Device-Key invalido. Confira a chave (comando: chave).");
    salvarOffline(registradoEm);
    return;
  }

  if (status == 401) {
    senhaIncorreta();
    return;
  }

  if (status > 0 && status < 500) {
    String motivo = mensagemDeErro(status, resposta);
    recusar("Nao registrado", motivo, motivo);
    return;
  }

  // Sem Wi-Fi, servidor fora do ar ou erro interno.
  salvarOffline(registradoEm);
}

static void processarTeclado() {
  char tecla = lerTecla();
  if (!tecla) {
    return;
  }

  // Qualquer tecla renova o tempo para digitar.
  prazo = millis() + TEMPO_DIGITAR_SENHA;

  if (tecla == 'D') {
    Serial.println();
    Serial.println("Operacao cancelada.");
    encerrar("Cancelado", "");
    return;
  }

  if (tecla == '*') {
    senha = "";
    Serial.println();
    Serial.println("Senha apagada.");
    mostrarSenhaDigitada();
    return;
  }

  if (tecla >= '0' && tecla <= '9' && senha.length() < (unsigned int)TAMANHO_SENHA) {
    senha += tecla;
    Serial.print("*");
    mostrarSenhaDigitada();
  }

  // Confirma sozinho ao completar os 4 numeros (# tambem confirma).
  if (senha.length() == (unsigned int)TAMANHO_SENHA) {
    enviarBatida();
  } else if (tecla == '#') {
    Serial.println();
    Serial.println("A senha tem " + String(TAMANHO_SENHA) + " numeros.");
  }
}

// =====================================================
// LEITURA DE CARTAO PEDIDA PELO PAINEL
// =====================================================
//
// No cadastro de funcionario, o painel cria um pedido em
// POST /api/v1/card-reads. O relogio consulta os pedidos,
// le o proximo cartao e devolve o UID para o painel.
// =====================================================

static void consultarPedidoDeLeitura() {
  if (!passou(proximaConsulta) || !wifiConectado()) {
    return;
  }

  JsonDocument resposta;
  int status = api("GET", "/device/card-reads/pending", "", &resposta);

  // Se o servidor nao responde, espera mais para nao travar o leitor.
  proximaConsulta = millis() + (status > 0 ? INTERVALO_PEDIDO_CADASTRO : INTERVALO_SEM_SERVIDOR);

  if (status == 200) {
    String id = resposta["id"] | "";

    if (estado == LENDO_CADASTRO && id == leituraId) {
      return;
    }

    leituraId = id;
    prazo = millis() + TEMPO_LEITURA_CADASTRO;
    estado = LENDO_CADASTRO;

    Serial.println();
    Serial.println("[PAINEL] Cadastro de cartao solicitado.");
    Serial.println("Aproxime o cartao que sera vinculado ao funcionario.");

    displayMostrar("Cadastro: aprox.", "o cartao novo");
    bipAviso();
  } else if (status == 204 && estado == LENDO_CADASTRO) {
    Serial.println("[PAINEL] Leitura cancelada ou expirada.");
    encerrar("Cadastro", "cancelado");
  }
}

static void enviarCartaoCadastro(const String& cartao) {
  JsonDocument corpoDoc;
  corpoDoc["cardTag"] = cartao;

  String corpo;
  serializeJson(corpoDoc, corpo);

  JsonDocument resposta;
  int status = api("POST", "/device/card-reads/" + leituraId, corpo, &resposta);

  if (status != 200) {
    String motivo = mensagemDeErro(status, resposta);
    Serial.println("Nao foi possivel enviar o cartao: " + motivo);
    bipErro();
    encerrar("Erro ao enviar", motivo);
    return;
  }

  Serial.println("Cartao " + cartao + " enviado ao painel.");
  bipEntrada();

  if (!resposta["assignedTo"].isNull()) {
    String dono = resposta["assignedTo"]["fullName"] | "";
    Serial.println("Atencao: esse cartao ja pertence a " + dono + ".");
    encerrar("Ja pertence a:", dono);
    return;
  }

  encerrar("Cartao enviado", cartao);
}

// =====================================================
// API DO MODULO
// =====================================================

void pontoIniciar() {
  voltarAoInicio();
}

bool pontoOcioso() {
  return estado == AGUARDANDO_CARTAO;
}

bool aguardandoSenha() {
  return estado == DIGITANDO_SENHA;
}

void pontoDigitarSenha(const String& novaSenha) {
  if (estado != DIGITANDO_SENHA) {
    return;
  }
  senha = novaSenha;
  enviarBatida();
}

void pontoAtualizar() {
  String cartao;

  switch (estado) {
    case AGUARDANDO_CARTAO:
      if (lerCartao(cartao)) {
        iniciarBatida(cartao);
        return;
      }
      consultarPedidoDeLeitura();
      break;

    case DIGITANDO_SENHA:
      processarTeclado();

      if (estado == DIGITANDO_SENHA && passou(prazo)) {
        Serial.println();
        Serial.println("Tempo esgotado.");
        bipErro();
        encerrar("Tempo esgotado", "");
      }
      break;

    case LENDO_CADASTRO:
      if (lerCartao(cartao)) {
        enviarCartaoCadastro(cartao);
        return;
      }

      consultarPedidoDeLeitura();

      if (estado == LENDO_CADASTRO && passou(prazo)) {
        Serial.println("[PAINEL] Nenhum cartao lido a tempo.");
        encerrar("Cadastro", "tempo esgotado");
      }
      break;
  }
}
