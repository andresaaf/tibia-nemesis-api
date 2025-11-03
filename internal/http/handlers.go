package http

import (
	"encoding/json"
	"net/http"
	"strconv"

	"tibia-nemesis-api/internal/service"

	"github.com/go-chi/chi/v5"
)

type Handlers struct {
	svc *service.Service
}

func NewHandlers(svc *service.Service) *Handlers { return &Handlers{svc: svc} }

func (h *Handlers) Status(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":  "ok",
		"version": "v0.1.0",
	})
}

func (h *Handlers) Worlds(w http.ResponseWriter, r *http.Request) {
	worlds, err := h.svc.Worlds(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, worlds)
}

func (h *Handlers) Spawnables(w http.ResponseWriter, r *http.Request) {
	world := r.URL.Query().Get("world")
	if world == "" {
		writeError(w, http.StatusBadRequest, errMissing("world"))
		return
	}
	list, err := h.svc.Spawnables(r.Context(), world)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (h *Handlers) BossHistory(w http.ResponseWriter, r *http.Request) {
	world := r.URL.Query().Get("world")
	if world == "" {
		writeError(w, http.StatusBadRequest, errMissing("world"))
		return
	}
	name := chi.URLParam(r, "name")
	limit := 25
	if s := r.URL.Query().Get("limit"); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v > 0 {
			limit = v
		}
	}
	list, err := h.svc.BossHistory(r.Context(), world, name, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (h *Handlers) Refresh(w http.ResponseWriter, r *http.Request) {
	world := r.URL.Query().Get("world")
	if world == "" {
		writeError(w, http.StatusBadRequest, errMissing("world"))
		return
	}
	if err := h.svc.RefreshWorld(r.Context(), world); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "world": world})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

type badReq string

func (e badReq) Error() string { return string(e) }

func errMissing(p string) error { return badReq("missing parameter: " + p) }
