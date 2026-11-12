package handler

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/kappannotavio/ponto-eletronico/service"
)

// DeviceHandler concentra as rotas usadas pelo relogio de ponto (ESP32).
type DeviceHandler struct {
	employees   *service.EmployeeService
	timeRecords *service.TimeRecordService
	cardReads   *service.CardReadService
}

func NewDeviceHandler(
	employees *service.EmployeeService,
	timeRecords *service.TimeRecordService,
	cardReads *service.CardReadService,
) *DeviceHandler {
	return &DeviceHandler{
		employees:   employees,
		timeRecords: timeRecords,
		cardReads:   cardReads,
	}
}

type pingResponse struct {
	Status     string    `json:"status"`
	ServerTime time.Time `json:"serverTime"`
	UnixTime   int64     `json:"unixTime"`
}

type deviceEmployeeResponse struct {
	ID       int64  `json:"id"`
	FullName string `json:"fullName"`
	IsActive bool   `json:"isActive"`
}

// deviceEmployeeListItem e o que o relogio guarda localmente para mostrar o
// nome e conferir a senha mesmo sem conexao com o servidor.
type deviceEmployeeListItem struct {
	CardTag  string `json:"cardTag"`
	FullName string `json:"fullName"`
	IsActive bool   `json:"isActive"`
	PinHash  string `json:"pinHash"`
}

type deviceEmployeeListResponse struct {
	Employees []deviceEmployeeListItem `json:"employees"`
}

type completeCardReadRequest struct {
	CardTag string `json:"cardTag"`
}

// Ping serve como healthcheck e para o ESP32 acertar o relogio sem NTP.
func (h *DeviceHandler) Ping(w http.ResponseWriter, r *http.Request) {
	now := time.Now()

	writeJSON(w, http.StatusOK, pingResponse{
		Status:     "ok",
		ServerTime: now,
		UnixTime:   now.Unix(),
	})
}

// ListEmployees devolve todos os funcionarios (ativos e desativados) para o
// relogio manter uma copia local. Responde 304 quando a lista nao mudou desde
// o ETag enviado em If-None-Match, para o ESP32 nao baixar tudo de novo.
func (h *DeviceHandler) ListEmployees(w http.ResponseWriter, r *http.Request) {
	employees, err := h.employees.List(service.EmployeeFilter{})
	if err != nil {
		writeError(w, err)
		return
	}

	response := deviceEmployeeListResponse{
		Employees: make([]deviceEmployeeListItem, 0, len(employees)),
	}
	for _, employee := range employees {
		response.Employees = append(response.Employees, deviceEmployeeListItem{
			CardTag:  employee.CardTag,
			FullName: employee.FullName,
			IsActive: employee.IsActive,
			PinHash:  employee.PinHash,
		})
	}

	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(response); err != nil {
		log.Printf("encode device employees: %v", err)
		writeJSON(w, http.StatusInternalServerError, errorResponse{"erro interno do servidor"})
		return
	}

	sum := sha256.Sum256(body.Bytes())
	etag := `"` + hex.EncodeToString(sum[:16]) + `"`

	w.Header().Set("ETag", etag)
	w.Header().Set("Cache-Control", "no-cache")

	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(body.Bytes())
}

func (h *DeviceHandler) FindEmployeeByTag(w http.ResponseWriter, r *http.Request) {
	employee, err := h.employees.FindByCardTag(r.PathValue("cardTag"))
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, deviceEmployeeResponse{
		ID:       employee.ID,
		FullName: employee.FullName,
		IsActive: employee.IsActive,
	})
}

func (h *DeviceHandler) Punch(w http.ResponseWriter, r *http.Request) {
	var input service.PunchInput
	if !decodeJSON(w, r, &input) {
		return
	}

	result, err := h.timeRecords.Punch(input)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, result)
}

func (h *DeviceHandler) PendingCardRead(w http.ResponseWriter, r *http.Request) {
	read, ok := h.cardReads.Pending()
	if !ok {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	writeJSON(w, http.StatusOK, read)
}

func (h *DeviceHandler) CompleteCardRead(w http.ResponseWriter, r *http.Request) {
	var request completeCardReadRequest
	if !decodeJSON(w, r, &request) {
		return
	}

	read, err := h.cardReads.Complete(r.PathValue("id"), request.CardTag)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, read)
}
