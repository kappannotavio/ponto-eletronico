package handler

import (
	"net/http"

	"github.com/kappannotavio/ponto-eletronico/service"
)

type CardReadHandler struct {
	service *service.CardReadService
}

func NewCardReadHandler(service *service.CardReadService) *CardReadHandler {
	return &CardReadHandler{
		service: service,
	}
}

func (h *CardReadHandler) Create(w http.ResponseWriter, r *http.Request) {
	read, err := h.service.Create()
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, read)
}

func (h *CardReadHandler) Get(w http.ResponseWriter, r *http.Request) {
	read, err := h.service.Get(r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, read)
}

func (h *CardReadHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	read, err := h.service.Cancel(r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, read)
}
