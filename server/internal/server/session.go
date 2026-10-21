package server

import (
	"net/http"

	"github.com/ChrQR/meeting-assistant/server/internal/client"
	"github.com/ChrQR/meeting-assistant/server/internal/handler"
)

const (
	sessionCookieName = "meeting_session"
	// sessionCookiePath is "/" so the cookie is sent to every API route.
	sessionCookiePath = "/"
)

// withOptionalUser attaches the session user when a valid cookie is present. It
// never rejects the request; authorization is enforced per route by
// requireUser.
func withOptionalUser(signer client.SessionSigner) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(sessionCookieName)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			claims, err := signer.Verify(cookie.Value)
			if err != nil {
				// Stale or tampered session: drop it and continue anonymously.
				clearSessionCookie(w)
				next.ServeHTTP(w, r)
				return
			}

			next.ServeHTTP(w, r.WithContext(handler.WithUserID(r.Context(), claims.UserID)))
		})
	}
}

func setSessionCookie(w http.ResponseWriter, token string, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     sessionCookiePath,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(client.SessionTTL.Seconds()),
	})
}

func clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     sessionCookiePath,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

// cookieSession adapts the cookie helpers to the interface handlers expect.
type cookieSession struct{ secure bool }

func (c cookieSession) Set(w http.ResponseWriter, token string) {
	setSessionCookie(w, token, c.secure)
}

func (c cookieSession) Clear(w http.ResponseWriter) {
	clearSessionCookie(w)
}
