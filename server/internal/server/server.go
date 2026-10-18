package server

import (
	"fmt"
	"net/http"

	"github.com/ChrQR/meeting-assistant/server/internal/api"
	"github.com/ChrQR/meeting-assistant/server/internal/handler"
	"github.com/ChrQR/meeting-assistant/server/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

type server struct {
	handler *chi.Mux
}
type Server interface {
	Run() error
}

func NewServer(
	transcriptionService service.TranscriptionService,
	storeService service.StoreService,
) Server {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		// AllowedOrigins:   []string{"https://foo.com"}, // Use this to allow specific origin hosts
		AllowedOrigins: []string{"https://*", "http://*"},
		// AllowOriginFunc:  func(r *http.Request, origin string) bool { return true },
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	}))

	handlerCollection := &handler.Handlers{
		TranscriptionService: transcriptionService,
		StoreService:         storeService,
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
