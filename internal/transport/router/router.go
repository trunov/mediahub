package router

import (
	"github.com/go-chi/chi/v5"
	"github.com/trunov/mediahub/internal/transport/handler"
)

func NewRouter(h *handler.Handler) chi.Router {
	r := chi.NewRouter()

	r.Route("/api", func(r chi.Router) {
		r.Post("/images", h.UploadImage)
	})

	return r
}
