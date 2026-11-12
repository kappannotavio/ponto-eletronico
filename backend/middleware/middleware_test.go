package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
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
