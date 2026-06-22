package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/qm-secrets/qm-secrets/internal/auth"
	"github.com/qm-secrets/qm-secrets/internal/model"
	"github.com/qm-secrets/qm-secrets/internal/service"
	"github.com/qm-secrets/qm-secrets/internal/store"
)

type SecretHandler struct {
	svc *service.SecretService
}

func NewSecretHandler(svc *service.SecretService) *SecretHandler {
	return &SecretHandler{svc: svc}
}

func (h *SecretHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/", h.Create)
	r.Get("/", h.List)
	r.Post("/poll", h.Poll)
	r.Route("/{name}", func(r chi.Router) {
		r.Get("/", h.Get)
		r.Put("/", h.Update)
		r.Delete("/", h.Delete)
	})
	return r
}

func (h *SecretHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req model.CreateSecretRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	billets := auth.BilletsFromContext(r.Context())
	summary, err := h.svc.Create(r.Context(), billets, req)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, summary)
}

func (h *SecretHandler) List(w http.ResponseWriter, r *http.Request) {
	billets := auth.BilletsFromContext(r.Context())
	secrets, err := h.svc.List(r.Context(), billets)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, secrets)
}

func (h *SecretHandler) Get(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	billets := auth.BilletsFromContext(r.Context())
	secret, err := h.svc.Get(r.Context(), billets, name)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, secret)
}

func (h *SecretHandler) Update(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	var req model.UpdateSecretRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	billets := auth.BilletsFromContext(r.Context())
	summary, err := h.svc.Update(r.Context(), billets, name, req)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

func (h *SecretHandler) Delete(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	billets := auth.BilletsFromContext(r.Context())
	if err := h.svc.Delete(r.Context(), billets, name); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *SecretHandler) Poll(w http.ResponseWriter, r *http.Request) {
	var req model.PollRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	billets := auth.BilletsFromContext(r.Context())
	resp, err := h.svc.Poll(r.Context(), billets, req)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func decodeJSON(r *http.Request, v any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, store.ErrNotFound):
		writeError(w, http.StatusNotFound, "secret not found")
	case errors.Is(err, service.ErrForbidden):
		writeError(w, http.StatusForbidden, "forbidden")
	default:
		if strings.Contains(err.Error(), "is required") {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
	}
}
