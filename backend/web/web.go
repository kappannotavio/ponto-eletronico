// Package web serve o painel (Angular) de dentro do executavel do backend.
//
// O ./dev.sh build compila o painel e copia o resultado para web/dist antes
// do go build; o //go:embed coloca esses arquivos dentro do binario. Assim o
// sistema inteiro vira um arquivo so (ponto / ponto.exe).
//
// Em desenvolvimento (go run, ./dev.sh) a pasta so tem o .gitkeep e o painel
// continua sendo servido pelo ng serve na porta 4200.
package web

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

//go:embed all:dist
var arquivos embed.FS

var dist, _ = fs.Sub(arquivos, "dist")

// Disponivel diz se o painel foi embutido neste executavel.
func Disponivel() bool {
	_, err := fs.Stat(dist, "index.html")
	return err == nil
}

const semPainel = `<!doctype html>
<html lang="pt-BR"><meta charset="utf-8"><title>Ponto eletronico</title>
<body style="font-family:sans-serif;max-width:40rem;margin:3rem auto;padding:0 1rem">
<h1>Painel nao incluido</h1>
<p>Este backend foi iniciado sem o painel embutido (modo de desenvolvimento).</p>
<p>Use <code>./dev.sh</code> e abra <a href="http://localhost:4200">http://localhost:4200</a>,
ou gere o executavel completo com <code>./dev.sh build</code>.</p>
</body></html>`

// Handler serve os arquivos do painel. Caminhos que nao sao arquivos
// (/employees, /time-records...) recebem o index.html, porque quem resolve
// essas rotas e o roteador do Angular no navegador.
func Handler() http.Handler {
	arquivosEstaticos := http.FileServer(http.FS(dist))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Rota de API inexistente: 404 em JSON, nunca a pagina do painel.
		if strings.HasPrefix(r.URL.Path, "/api/") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error":"rota nao encontrada"}`))
			return
		}

		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "metodo nao permitido", http.StatusMethodNotAllowed)
			return
		}

		if !Disponivel() {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write([]byte(semPainel))
			return
		}

		arquivo := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if arquivo != "" && arquivo != "index.html" {
			if info, err := fs.Stat(dist, arquivo); err == nil && !info.IsDir() {
				// Arquivos com hash no nome (main-ABC123.js) nunca mudam.
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
				arquivosEstaticos.ServeHTTP(w, r)
				return
			}
		}

		servirIndex(w)
	})
}

func servirIndex(w http.ResponseWriter) {
	conteudo, err := fs.ReadFile(dist, "index.html")
	if err != nil {
		http.Error(w, "painel indisponivel", http.StatusInternalServerError)
		return
	}

	// Sempre buscar o index.html de novo: ele aponta para os arquivos da versao atual.
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(conteudo)
}
