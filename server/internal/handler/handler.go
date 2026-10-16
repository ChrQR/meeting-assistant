package handler

import (
	"github.com/ChrQR/meeting-assistant/server/internal/api"
	"github.com/ChrQR/meeting-assistant/server/internal/service"
)

type Handlers struct {
	TranscriptionService service.TranscriptionService
}

var _ api.StrictServerInterface = (*Handlers)(nil)
