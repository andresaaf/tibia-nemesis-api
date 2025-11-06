package http

import (
	"net/http"

	"tibia-nemesis-api/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewRouter(svc *service.Service) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	h := NewHandlers(svc)
	r.Get("/api/v1/status", h.Status)
	r.Get("/api/v1/worlds", h.Worlds)
	r.Get("/api/v1/bosses", h.Bosses)
	r.Get("/api/v1/boss/{name}/history", h.BossHistory)
	r.Post("/api/v1/refresh", h.Refresh)

	return r
}
