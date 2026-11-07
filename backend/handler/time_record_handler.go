package handler

import (
	"net/http"
	"time"

	"github.com/kappannotavio/ponto-eletronico/repository"
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
	EmployeeID int64     `json:"employeeId"`
	RecordedAt time.Time `json:"recordedAt"`
}

type replaceDayRequest struct {
	Times []string `json:"times"`
}

func (h *TimeRecordHandler) List(w http.ResponseWriter, r *http.Request) {
	employeeID, ok := queryEmployeeID(w, r)
	if !ok {
		return
	}
	from, to, ok := dateRange(w, r)
	if !ok {
		return
	}

	records, err := h.service.List(repository.TimeRecordFilter{
		EmployeeID: employeeID,
		From:       from,
		To:         to,
	})
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, records)
}

func (h *TimeRecordHandler) Create(w http.ResponseWriter, r *http.Request) {
	var request createTimeRecordRequest
	if !decodeJSON(w, r, &request) {
		return
	}

	record, err := h.service.CreateManual(request.EmployeeID, request.RecordedAt)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, record)
}

func (h *TimeRecordHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}

	if err := h.service.Delete(id); err != nil {
		writeError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *TimeRecordHandler) Attendance(w http.ResponseWriter, r *http.Request) {
	filter, ok := attendanceFilter(w, r)
	if !ok {
		return
	}

	attendance, err := h.service.Attendance(filter)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, attendance)
}

func (h *TimeRecordHandler) ReplaceDay(w http.ResponseWriter, r *http.Request) {
	employeeID, ok := pathID(w, r, "employeeId")
	if !ok {
		return
	}

	var request replaceDayRequest
	if !decodeJSON(w, r, &request) {
		return
	}

	attendance, err := h.service.ReplaceDay(employeeID, r.PathValue("date"), request.Times)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, attendance)
}

func (h *TimeRecordHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	dashboard, err := h.service.Dashboard()
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dashboard)
}

func attendanceFilter(w http.ResponseWriter, r *http.Request) (service.AttendanceFilter, bool) {
	employeeID, ok := queryEmployeeID(w, r)
	if !ok {
		return service.AttendanceFilter{}, false
	}
	from, to, ok := dateRange(w, r)
	if !ok {
		return service.AttendanceFilter{}, false
	}

	return service.AttendanceFilter{EmployeeID: employeeID, From: from, To: to}, true
}
