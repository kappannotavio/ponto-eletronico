package middleware

import (
	"crypto/subtle"
	"encoding/json"
	"log"
	"net/http"
	"slices"
	"time"
)

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(recorder, r)

		log.Printf("%s %s %d %s", r.Method, r.URL.RequestURI(), recorder.status, time.Since(start).Round(time.Millisecond))
	})
}

// CORS libera o painel rodando em outra origem. "*" libera qualquer origem.
func CORS(allowedOrigins []string) func(http.Handler) http.Handler {
	allowAll := slices.Contains(allowedOrigins, "*")

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			if origin != "" && (allowAll || slices.Contains(allowedOrigins, origin)) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Device-Key")
				w.Header().Set("Access-Control-Expose-Headers", "Content-Disposition")
			}

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// DeviceAuth exige o header X-Device-Key nas rotas do relogio de ponto.
func DeviceAuth(apiKey string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := r.Header.Get("X-Device-Key")

		if subtle.ConstantTimeCompare([]byte(key), []byte(apiKey)) != 1 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			// "code" deixa o ESP32 diferenciar chave errada de senha errada
			// (as duas sao 401) e nao descartar a fila offline.
			json.NewEncoder(w).Encode(map[string]string{
				"error": "X-Device-Key invalido",
				"code":  "invalid_device_key",
			})
			return
		}

		next(w, r)
	}
}
