package handler

import (
	"net/http"

	"github.com/ChrQR/meeting-assistant/server/internal/api"
	"github.com/ChrQR/meeting-assistant/server/internal/service"
)

// SessionCookie writes and clears the httpOnly session cookie. The server
// package owns the cookie attributes; handlers only need these two operations.
type SessionCookie interface {
	Set(w http.ResponseWriter, token string)
	Clear(w http.ResponseWriter)
}

type Handlers struct {
	TranscriptionService service.TranscriptionService
	StoreService         service.StoreService
	AuthService          service.AuthService
	Session              SessionCookie
}

var _ api.ServerInterface = (*Handlers)(nil)

func (s *Handlers) clearSession(w http.ResponseWriter) {
	if s.Session != nil {
		s.Session.Clear(w)
	}
}
