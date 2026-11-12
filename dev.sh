#!/usr/bin/env bash
# Script de desenvolvimento do Ponto Eletronico.
# Uso: ./dev.sh [comando]   (sem comando = sobe backend + frontend)
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BACKEND="$ROOT/backend"
FRONTEND="$ROOT/frontend"
ESP32="$ROOT/esp32"

# Dados de exemplo ligados por padrao. Desligue com: SEED_DEMO_DATA=false ./dev.sh
export SEED_DEMO_DATA="${SEED_DEMO_DATA:-true}"

blue() { printf '\033[34m%s\033[0m\n' "$*"; }
red() { printf '\033[31m%s\033[0m\n' "$*" >&2; }

require() {
  if ! command -v "$1" >/dev/null 2>&1; then
    red "Comando '$1' nao encontrado. $2"
    exit 1
  fi
}

install_frontend() {
  require npm "Instale o Node.js 20+: https://nodejs.org"
  if [ ! -d "$FRONTEND/node_modules" ]; then
    blue "Instalando dependencias do frontend (so na primeira vez)..."
    (cd "$FRONTEND" && npm install)
  fi
}

# Prefixa cada linha de saida para saber de onde veio.
prefix() {
  local label="$1" color="$2"
  while IFS= read -r line; do
    printf '\033[%sm[%s]\033[0m %s\n' "$color" "$label" "$line"
  done
}

run_backend() {
  require go "Instale o Go 1.22+: https://go.dev/dl"
  blue "Backend em http://localhost:${PORT:-8080} (SEED_DEMO_DATA=$SEED_DEMO_DATA)"
  cd "$BACKEND" && exec go run ./cmd/api
}

run_frontend() {
  install_frontend
  blue "Frontend em http://localhost:4200"
  cd "$FRONTEND" && exec npx ng serve
}

run_all() {
  require go "Instale o Go 1.22+: https://go.dev/dl"
  install_frontend

  # Ctrl+C encerra os dois processos.
  trap 'echo; blue "Encerrando..."; kill 0' INT TERM

  (cd "$BACKEND" && go run ./cmd/api 2>&1 | prefix backend 33) &
  (cd "$FRONTEND" && npx ng serve 2>&1 | prefix frontend 36) &

  blue "Subindo tudo... painel: http://localhost:4200 | API: http://localhost:8080/api/v1"
  blue "Ctrl+C para parar."
  wait
}

run_tests() {
  require go "Instale o Go 1.22+: https://go.dev/dl"
  install_frontend
  blue "Testes do backend"
  (cd "$BACKEND" && go test ./...)
  blue "Testes do frontend"
  (cd "$FRONTEND" && npx ng test --watch=false)
}

run_relogio() {
  require go "Instale o Go 1.22+: https://go.dev/dl"
  cd "$BACKEND" && exec go run ./cmd/relogio-virtual
}

# Usa o pio do PATH ou o instalado em ~/.platformio (mesmo da extensao do VS Code).
find_pio() {
  if command -v pio >/dev/null 2>&1; then
    command -v pio
  elif [ -x "$HOME/.platformio/penv/bin/pio" ]; then
    echo "$HOME/.platformio/penv/bin/pio"
  else
    red "PlatformIO nao encontrado. Instale a extensao 'PlatformIO IDE' no VS Code ou: pip install platformio"
    exit 1
  fi
}

run_esp32() {
  local pio action="${1:-build}"
  pio="$(find_pio)"
  cd "$ESP32"
  case "$action" in
    build)
      "$pio" run
      blue "Firmware pronto. Para simular: abra esp32/diagram.json no VS Code e rode 'Wokwi: Start Simulator'."
      ;;
    upload) "$pio" run -t upload ;;
    monitor) "$pio" device monitor ;;
    *) red "Use: ./dev.sh esp32 [build|upload|monitor]"; exit 1 ;;
  esac
}

# Bate um ponto de teste como se fosse o ESP32 (precisa do backend rodando).
fake_punch() {
  local tag="${1:-A1:B2:C3:D4}" pin="${2:-1234}"
  curl -s -X POST "http://localhost:${PORT:-8080}/api/v1/device/punches" \
    -H "X-Device-Key: ${DEVICE_API_KEY:-dev-device-key}" \
    -H 'Content-Type: application/json' \
    -d "{\"cardTag\":\"$tag\",\"pin\":\"$pin\"}"
  echo
}

usage() {
  cat <<USAGE
Uso: ./dev.sh [comando]

  (nada)            sobe backend + frontend juntos
  backend           sobe so o backend (porta 8080)
  frontend          sobe so o frontend (porta 4200)
  relogio           relogio de ponto virtual no terminal (simula o ESP32)
  test              roda os testes do backend e do frontend
  esp32 [build|upload|monitor]
                    compila, grava ou abre o serial do ESP32
  punch [TAG] [PIN] bate um ponto de teste sem ESP32
                    (padrao: A1:B2:C3:D4 1234)
  help              mostra esta ajuda

Variaveis: SEED_DEMO_DATA=false para nao criar dados de exemplo, DB_PATH (arquivo do banco, padrao backend/ponto.db), PORT, DEVICE_API_KEY.
USAGE
}

case "${1:-all}" in
  all) run_all ;;
  backend) run_backend ;;
  frontend) run_frontend ;;
  relogio) run_relogio ;;
  test) run_tests ;;
  esp32) run_esp32 "${2:-build}" ;;
  punch) fake_punch "${2:-}" "${3:-}" ;;
  help | -h | --help) usage ;;
  *) red "Comando desconhecido: $1"; usage; exit 1 ;;
esac
