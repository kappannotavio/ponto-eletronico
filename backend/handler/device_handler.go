package handler

import (
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
