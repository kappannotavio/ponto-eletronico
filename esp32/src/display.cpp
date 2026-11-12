#include "display.h"

#include <LiquidCrystal_I2C.h>
#include <Wire.h>

#include "config.h"
#include "fila_offline.h"
#include "rede.h"
#include "relogio.h"

static LiquidCrystal_I2C* lcd = nullptr;

static String linhasAtuais[LCD_LINHAS];
static unsigned long mensagemAte = 0;
static bool mensagemTemporaria = false;
static unsigned long proximaAtualizacao = 0;

static bool enderecoResponde(uint8_t endereco) {
  Wire.beginTransmission(endereco);
  return Wire.endTransmission() == 0;
}

void displayIniciar() {
  // Wire.begin() ja foi chamado em main.cpp.
  for (uint8_t endereco : LCD_ENDERECOS) {
    if (enderecoResponde(endereco)) {
      lcd = new LiquidCrystal_I2C(endereco, LCD_COLUNAS, LCD_LINHAS);
      lcd->init();
      lcd->backlight();
      lcd->clear();
      Serial.printf("LCD encontrado no endereco 0x%02X\n", endereco);
      return;
    }
  }

  Serial.println("LCD NAO encontrado (I2C 0x27/0x3F). Confira SDA=D21, SCL=D22 e VCC=VIN.");
}

bool displayPresente() {
  return lcd != nullptr;
}

// Converte UTF-8 (como vem do backend) para ASCII sem acento.
String textoParaLcd(const String& texto) {
  String resultado = "";
  resultado.reserve(texto.length());

  for (unsigned int i = 0; i < texto.length(); i++) {
    uint8_t c = (uint8_t)texto[i];

    if (c < 0x80) {
      resultado += (char)c;
      continue;
    }

    // Latin-1 em UTF-8: 0xC3 seguido de um byte.
    if (c == 0xC3 && i + 1 < texto.length()) {
      uint8_t d = (uint8_t)texto[++i];
      const char* mapa =
        "AAAAAAACEEEEIIII"  // 0x80-0x8F: À Á Â Ã Ä Å Æ Ç È É Ê Ë Ì Í Î Ï
        "DNOOOOOxOUUUUYTs"  // 0x90-0x9F: Ð Ñ Ò Ó Ô Õ Ö × Ø Ù Ú Û Ü Ý Þ ß
        "aaaaaaaceeeeiiii"  // 0xA0-0xAF
        "dnooooo/ouuuuyty"; // 0xB0-0xBF
      if (d >= 0x80 && d <= 0xBF) {
        resultado += mapa[d - 0x80];
      }
      continue;
    }

    // Outros caracteres multibyte: pula os bytes de continuacao.
    while (i + 1 < texto.length() && ((uint8_t)texto[i + 1] & 0xC0) == 0x80) {
      i++;
    }
    resultado += '?';
  }

  return resultado;
}

static String ajustarLinha(const String& texto) {
  String linha = textoParaLcd(texto);

  if (linha.length() > LCD_COLUNAS) {
    linha = linha.substring(0, LCD_COLUNAS);
  }
  while (linha.length() < LCD_COLUNAS) {
    linha += ' ';
  }

  return linha;
}

static void escreverLinhas(const String& linha1, const String& linha2) {
  if (lcd == nullptr) {
    return;
  }

  String novas[LCD_LINHAS] = {ajustarLinha(linha1), ajustarLinha(linha2)};

  // So reescreve o que mudou para a tela nao piscar.
  for (uint8_t i = 0; i < LCD_LINHAS; i++) {
    if (novas[i] != linhasAtuais[i]) {
      lcd->setCursor(0, i);
      lcd->print(novas[i]);
      linhasAtuais[i] = novas[i];
    }
  }
}

void displayMostrar(const String& linha1, const String& linha2, unsigned long duracaoMs) {
  escreverLinhas(linha1, linha2);

  mensagemTemporaria = duracaoMs > 0;
  mensagemAte = millis() + duracaoMs;
}

static void mostrarTelaInicial() {
  String linha1 = relogioSincronizado() ? "Aproxime cartao" : "Sem hora certa";

  // Linha 2: "14:05:32      ON" ou "14:05:32  OFF 12" (pendentes na fila).
  String hora = relogioSincronizado() ? horaAtual() : String("--:--:--");
  String situacao = servidorOnline() ? "ON" : "OFF";

  int pendentes = filaTamanho();
  if (pendentes > 0) {
    situacao += " " + String(pendentes);
  }

  String linha2 = hora;
  while (linha2.length() + situacao.length() < LCD_COLUNAS) {
    linha2 += ' ';
  }
  linha2 += situacao;

  escreverLinhas(linha1, linha2);
}

void displayAtualizar(bool ocioso) {
  if (lcd == nullptr || !ocioso) {
    return;
  }

  if (mensagemTemporaria) {
    if ((long)(millis() - mensagemAte) < 0) {
      return;
    }
    mensagemTemporaria = false;
  }

  if ((long)(millis() - proximaAtualizacao) < 0) {
    return;
  }
  proximaAtualizacao = millis() + 500;

  mostrarTelaInicial();
}
