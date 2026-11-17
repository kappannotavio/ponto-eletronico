package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func TestSPA(t *testing.T) {
	files := fstest.MapFS{
		"index.html":  {Data: []byte("<html>painel</html>")},
		"main-ABC.js": {Data: []byte("console.log(1)")},
		"favicon.ico": {Data: []byte("ico")},
	}
	handler := spa(files)

	tests := []struct {
		path, wantBody, wantCache string
		wantStatus                int
	}{
		{"/", "painel", "no-cache", 200},
		{"/employees/new", "painel", "no-cache", 200},
		{"/main-ABC.js", "console.log", "immutable", 200},
		{"/api/v1/nao-existe", "rota nao encontrada", "", 404},
	}

	for _, tt := range tests {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.path, nil))

		if rec.Code != tt.wantStatus {
			t.Errorf("%s: status %d, want %d", tt.path, rec.Code, tt.wantStatus)
		}
		if !strings.Contains(rec.Body.String(), tt.wantBody) {
			t.Errorf("%s: body %q, want %q", tt.path, rec.Body.String(), tt.wantBody)
		}
		if !strings.Contains(rec.Header().Get("Cache-Control"), tt.wantCache) {
			t.Errorf("%s: Cache-Control %q, want %q", tt.path, rec.Header().Get("Cache-Control"), tt.wantCache)
		}
	}
}
