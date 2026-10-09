package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/kappannotavio/ponto-eletronico/service"
)

type TimeRecordHandler struct {
	service *service.TimeRecordService
}

func NewTimeRecordHandler(service *service.TimeRecordService) *TimeRecordHandler {
	return &TimeRecordHandler{
		service: service,
	}
}

type createTimeRecordRequest struct {
	TagID     string    `json:"tag_id"`
	Timestamp time.Time `json:"timestamp"`
}

func (h *TimeRecordHandler) Create(w http.ResponseWriter, r *http.Request) {
	var request createTimeRecordRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	record, err := h.service.CreateTimeRecord(
		request.TagID,
		request.Timestamp,
	)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	json.NewEncoder(w).Encode(record)
}
