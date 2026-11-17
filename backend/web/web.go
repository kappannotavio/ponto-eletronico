// Package web serve o painel Angular embutido no executavel.
//
// O painel compilado (npm run build no frontend) vai para web/dist/painel e
// entra no binario pelo go:embed. Assim o PC que roda o sistema so precisa do
// ponto.exe: nada de Node, npm ou servidor web separado.
package web

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

//go:embed all:dist
var dist embed.FS

// Painel devolve o handler do painel e se ele foi embutido no build.
func Painel() (http.Handler, bool) {
	files, err := fs.Sub(dist, "dist/painel")
	if err != nil {
		return notBuilt(), false
	}
	if _, err := fs.Stat(files, "index.html"); err != nil {
		return notBuilt(), false
	}
	return spa(files), true
}

// spa serve os arquivos do painel. Caminhos que nao sao arquivo (ex.:
// /employees ao recarregar a pagina) recebem o index.html e o roteador do
// Angular decide a tela.
func spa(files fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(files))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error":"rota nao encontrada"}`))
			return
		}

		name := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if name != "" && name != "index.html" {
			if info, err := fs.Stat(files, name); err == nil && !info.IsDir() {
				// Arquivos com hash no nome (main-ABC123.js) nunca mudam.
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
				fileServer.ServeHTTP(w, r)
				return
			}
		}

		index, err := fs.ReadFile(files, "index.html")
		if err != nil {
			http.Error(w, "painel indisponivel", http.StatusInternalServerError)
			return
		}
		// O index.html aponta para os arquivos da versao atual: sempre buscar de novo.
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(index)
	})
}

func notBuilt() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error":"rota nao encontrada"}`))
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`<!doctype html><meta charset="utf-8"><title>Ponto Eletronico</title>
<body style="font-family:sans-serif;max-width:40rem;margin:3rem auto;padding:0 1rem">
<h1>Painel nao incluido neste executavel</h1>
<p>Este backend foi compilado sem o painel. Gere o executavel completo com
<code>./dev.sh build</code> na raiz do projeto.</p>
<p>Em desenvolvimento, o painel fica em <a href="http://localhost:4200">http://localhost:4200</a>.</p>
</body>`))
	})
}
