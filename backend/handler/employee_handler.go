package handler

import (
	"net/http"
	"strconv"

	"github.com/kappannotavio/ponto-eletronico/service"
)

type EmployeeHandler struct {
	service *service.EmployeeService
}

func NewEmployeeHandler(service *service.EmployeeService) *EmployeeHandler {
	return &EmployeeHandler{
		service: service,
	}
}

type updateStatusRequest struct {
	IsActive *bool `json:"isActive"`
}

func (h *EmployeeHandler) List(w http.ResponseWriter, r *http.Request) {
	filter := service.EmployeeFilter{
		Search: r.URL.Query().Get("search"),
	}

	if value := r.URL.Query().Get("active"); value != "" {
		active, err := strconv.ParseBool(value)
		if err != nil {
			badRequest(w, "active deve ser true ou false")
			return
		}
		filter.Active = &active
	}

	employees, err := h.service.List(filter)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, employees)
}

func (h *EmployeeHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}

	employee, err := h.service.Get(id)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, employee)
}

func (h *EmployeeHandler) Create(w http.ResponseWriter, r *http.Request) {
	var input service.EmployeeInput
	if !decodeJSON(w, r, &input) {
		return
	}

	employee, err := h.service.Create(input)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, employee)
}

func (h *EmployeeHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}

	var input service.EmployeeInput
	if !decodeJSON(w, r, &input) {
		return
	}

	employee, err := h.service.Update(id, input)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, employee)
}

func (h *EmployeeHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}

	var request updateStatusRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	if request.IsActive == nil {
		badRequest(w, "isActive e obrigatorio")
		return
	}

	employee, err := h.service.SetActive(id, *request.IsActive)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, employee)
}
