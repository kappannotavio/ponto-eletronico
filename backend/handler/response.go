package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/kappannotavio/ponto-eletronico/service"
)

const maxBodySize = 1 << 20

type errorResponse struct {
	Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(body); err != nil {
		log.Printf("encode response: %v", err)
	}
}

func writeError(w http.ResponseWriter, err error) {
	var serviceError *service.Error
	if !errors.As(err, &serviceError) {
		log.Printf("internal error: %v", err)
		writeJSON(w, http.StatusInternalServerError, errorResponse{"erro interno do servidor"})
		return
	}

	status := http.StatusBadRequest
	switch serviceError.Kind {
	case service.KindNotFound:
		status = http.StatusNotFound
	case service.KindConflict:
		status = http.StatusConflict
	case service.KindUnauthorized:
		status = http.StatusUnauthorized
	case service.KindForbidden:
		status = http.StatusForbidden
	}

	writeJSON(w, status, errorResponse{serviceError.Message})
}

func badRequest(w http.ResponseWriter, message string) {
	writeJSON(w, http.StatusBadRequest, errorResponse{message})
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)

	if err := json.NewDecoder(r.Body).Decode(target); err != nil {
		badRequest(w, "corpo da requisicao invalido: "+err.Error())
		return false
	}

	return true
}

func pathID(w http.ResponseWriter, r *http.Request, name string) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue(name), 10, 64)
	if err != nil || id <= 0 {
		badRequest(w, name+" invalido")
		return 0, false
	}

	return id, true
}

// dateRange le ?date=YYYY-MM-DD ou ?from=...&to=... (ambos inclusivos).
// Devolve From inclusivo e To exclusivo, no formato do repositorio.
func dateRange(w http.ResponseWriter, r *http.Request) (time.Time, time.Time, bool) {
	query := r.URL.Query()
	fromValue, toValue := query.Get("from"), query.Get("to")

	if date := query.Get("date"); date != "" {
		fromValue, toValue = date, date
	}

	var from, to time.Time
	if fromValue != "" {
		parsed, err := service.ParseDate(fromValue)
		if err != nil {
			writeError(w, err)
			return from, to, false
		}
		from = parsed
	}
	if toValue != "" {
		parsed, err := service.ParseDate(toValue)
		if err != nil {
			writeError(w, err)
			return from, to, false
		}
		to = parsed.AddDate(0, 0, 1)
	}

	return from, to, true
}

func queryEmployeeID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	value := r.URL.Query().Get("employeeId")
	if value == "" {
		return 0, true
	}

	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id <= 0 {
		badRequest(w, "employeeId invalido")
		return 0, false
	}

	return id, true
}
