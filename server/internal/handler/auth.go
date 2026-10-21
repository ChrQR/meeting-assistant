package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/ChrQR/meeting-assistant/server/internal/api"
	"github.com/ChrQR/meeting-assistant/server/internal/types"
)

// OAuthStateCookie carries the CSRF state and PKCE verifier between the login
// redirect and the provider callback. It is short lived and scoped to /auth.
const (
	OAuthStateCookie = "meeting_oauth_state"
	OAuthFlowTTL     = 10 * time.Minute
)

// userIDKey is the request context key holding the authenticated user id.
type contextKey string

const userIDKey contextKey = "userID"

// WithUserID stores the authenticated user id on the context.
func WithUserID(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, userIDKey, id)
}

// UserIDFromContext returns the authenticated user id, if any.
func UserIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(userIDKey).(uuid.UUID)
	return id, ok
}

// currentUser resolves the authenticated user for a request.
func (s *Handlers) currentUser(r *http.Request) (types.AppUser, bool) {
	id, ok := UserIDFromContext(r.Context())
	if !ok {
		return types.AppUser{}, false
	}
	user, err := s.AuthService.GetUser(r.Context(), id)
	if err != nil {
		return types.AppUser{}, false
	}
	return user, true
}

// requireUserID returns the authenticated user id, writing 401 when absent.
// Handlers that are only reachable behind the auth middleware can use MustUserID.
func (s *Handlers) requireUserID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, ok := UserIDFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "not authenticated"})
		return uuid.Nil, false
	}
	return id, true
}

// GetMe returns the signed-in user, or 401 when there is no valid session.
func (s *Handlers) GetMe(w http.ResponseWriter, r *http.Request) {
	user, ok := s.currentUser(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "not authenticated"})
		return
	}
	writeJSON(w, http.StatusOK, toUserResponse(user))
}

// Logout clears the session cookie.
func (s *Handlers) Logout(w http.ResponseWriter, r *http.Request) {
	s.clearSession(w)
	w.WriteHeader(http.StatusNoContent)
}

// GetAuthProviders lists the configured SSO providers so the frontend can render
// one sign-in option per provider.
func (s *Handlers) GetAuthProviders(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, api.AuthProviders{Providers: s.AuthService.Providers()})
}

func toUserResponse(user types.AppUser) api.User {
	return api.User{
		Id:        user.ID,
		Email:     user.Email,
		Name:      nullableFromPtr(user.Name),
		AvatarUrl: nullableFromPtr(user.AvatarURL),
		CreatedAt: user.CreatedAt,
	}
}
