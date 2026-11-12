package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kappannotavio/ponto-eletronico/middleware"
	"github.com/kappannotavio/ponto-eletronico/service"
)

// A senha errada tambem e 401, mas nao pode vir com o code da chave do
// dispositivo: o ESP32 descarta essa batida da fila, e guarda a fila
// so quando o code e de chave invalida.
func TestWriteErrorWrongPinIsNotDeviceKeyError(t *testing.T) {
	rec := httptest.NewRecorder()

	writeError(rec, &service.Error{Kind: service.KindUnauthorized, Message: "senha incorreta"})

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}

	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["code"] == middleware.InvalidDeviceKeyCode {
		t.Fatalf("senha errada respondeu com code %q", middleware.InvalidDeviceKeyCode)
	}
}
