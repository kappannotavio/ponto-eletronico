package middleware

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestDeviceAuth(t *testing.T) {
	handler := DeviceAuth("chave-certa", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	tests := []struct {
		name   string
		key    string
		status int
	}{
		{"sem header", "", http.StatusUnauthorized},
		{"chave errada", "chave-errada", http.StatusUnauthorized},
		{"chave certa", "chave-certa", http.StatusNoContent},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/device/punches", nil)
			if tt.key != "" {
				req.Header.Set("X-Device-Key", tt.key)
			}
			rec := httptest.NewRecorder()

			handler(rec, req)

			if rec.Code != tt.status {
				t.Fatalf("status = %d, want %d", rec.Code, tt.status)
			}
			if tt.status != http.StatusUnauthorized {
				return
			}

			var body map[string]string
			if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if body["code"] != InvalidDeviceKeyCode {
				t.Fatalf("code = %q, want %q", body["code"], InvalidDeviceKeyCode)
			}
			if body["error"] == "" {
				t.Fatal("error vazio")
			}
		})
	}
}

// O HTTPClient do ESP32 le uma resposta sem Content-Length (como o 204 de
// "nenhum pedido de leitura") ate a conexao fechar. Com keep-alive o relogio
// travava, entao as rotas do dispositivo precisam fechar a conexao.
func TestDeviceAuthClosesConnectionAfterResponse(t *testing.T) {
	server := httptest.NewServer(DeviceAuth("chave", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	conn, err := net.Dial("tcp", server.Listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	// Mesma requisicao que o ESP32 manda: HTTP/1.1 com keep-alive.
	fmt.Fprintf(conn, "GET /device/card-reads/pending HTTP/1.1\r\nHost: x\r\nX-Device-Key: chave\r\nConnection: keep-alive\r\n\r\n")

	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	resposta, err := io.ReadAll(conn)
	if err != nil {
		t.Fatalf("servidor nao fechou a conexao depois do 204 (o ESP32 ficaria travado): %v", err)
	}
	if !strings.HasPrefix(string(resposta), "HTTP/1.1 204") {
		t.Fatalf("resposta inesperada: %q", resposta)
	}
	if !strings.Contains(string(resposta), "Connection: close") {
		t.Fatalf("resposta sem Connection: close: %q", resposta)
	}
}
