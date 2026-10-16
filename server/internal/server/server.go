package server

import (
	"fmt"
	"net/http"

	"github.com/ChrQR/meeting-assistant/server/internal/api"
	"github.com/ChrQR/meeting-assistant/server/internal/handler"
	"github.com/ChrQR/meeting-assistant/server/internal/service"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
)

type server struct {
	handler *chi.Mux
}
type Server interface {
	Run() error
}

func NewServer(
	transcriptionService service.TranscriptionService,
) Server {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	handlerCollection := &handler.Handlers{
		TranscriptionService: transcriptionService,
	}
	handler := api.NewStrictHandler(handlerCollection, nil)
	api.HandlerFromMux(handler, r)

	return &server{
		r,
	}
}

func (s *server) Run() error {
	fmt.Println("Server listening on :8080...")
	if err := http.ListenAndServe(":8080", s.handler); err != nil {
		return err
	}
	return nil
}
