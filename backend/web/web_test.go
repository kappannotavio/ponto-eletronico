package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandlerAPIRouteIsNotThePanel(t *testing.T) {
	rec := httptest.NewRecorder()
	Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/nao-existe", nil))

	if rec.Code != http.StatusNotFound || !strings.Contains(rec.Header().Get("Content-Type"), "json") {
		t.Fatalf("rota de API inexistente deveria dar 404 JSON, veio %d %q", rec.Code, rec.Header().Get("Content-Type"))
	}
}

func TestHandlerRejectsNonGet(t *testing.T) {
	rec := httptest.NewRecorder()
	Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/employees", nil))

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST no painel deveria dar 405, veio %d", rec.Code)
	}
}

func TestHandlerServesPage(t *testing.T) {
	for _, caminho := range []string{"/", "/employees", "/time-records"} {
		rec := httptest.NewRecorder()
		Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, caminho, nil))

		if rec.Code != http.StatusOK || !strings.Contains(rec.Header().Get("Content-Type"), "text/html") {
			t.Fatalf("%s: esperado 200 HTML, veio %d %q", caminho, rec.Code, rec.Header().Get("Content-Type"))
		}
	}
}
