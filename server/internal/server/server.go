package server

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/ChrQR/meeting-assistant/server/internal/api"
	"github.com/ChrQR/meeting-assistant/server/internal/client"
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

type Options struct {
	TranscriptionService service.TranscriptionService
	StoreService         service.StoreService
	AuthService          service.AuthService
	Signer               client.SessionSigner
	// AppURL is the SPA origin users are sent back to after signing in.
	AppURL string
	// SecureCookies enables the Secure cookie flag (required over HTTPS).
	SecureCookies bool
}

func NewServer(options Options) Server {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"https://*", "http://*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders: []string{"Link"},
		// Required for the httpOnly session cookie to be sent cross-origin.
		AllowCredentials: true,
		MaxAge:           300,
	}))
	r.Use(withOptionalUser(options.Signer))

	session := cookieSession{secure: options.SecureCookies}

	handlerCollection := &handler.Handlers{
		TranscriptionService: options.TranscriptionService,
		StoreService:         options.StoreService,
		AuthService:          options.AuthService,
		Session:              session,
	}

	authHandlers := &authRoutes{
		auth:    options.AuthService,
		signer:  options.Signer,
		session: session,
		appURL:  options.AppURL,
	}

	// The login/callback endpoints speak in redirects rather than JSON, so they
	// are registered directly instead of through the generated handlers.
	r.Get("/auth/login", authHandlers.login)
	r.Get("/auth/callback", authHandlers.callback)

	// Everything except the auth endpoints requires a valid session.
	r.Group(func(r chi.Router) {
		r.Use(requireUser)

		r.Get("/auth/me", handlerCollection.GetMe)
		r.Post("/auth/logout", handlerCollection.Logout)
		r.Get("/auth/providers", handlerCollection.GetAuthProviders)

		api.HandlerFromMux(handlerCollection, r)
	})

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

// authRoutes implements the OAuth redirect endpoints, which are deliberately
// outside the generated API surface because they speak in redirects, not JSON.
type authRoutes struct {
	auth    service.AuthService
	signer  client.SessionSigner
	session cookieSession
	appURL  string
}

// login starts the OAuth flow. The provider, state and PKCE verifier are kept in
// a short-lived httpOnly cookie so the callback can complete the exchange.
func (a *authRoutes) login(w http.ResponseWriter, r *http.Request) {
	provider := r.URL.Query().Get("provider")

	redirectURL, verifier, state, err := a.auth.AuthorizeURL(provider)
	if err != nil {
		writeAuthError(w, err)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     handler.OAuthStateCookie,
		Value:    state + "|" + verifier,
		Path:     "/auth",
		HttpOnly: true,
		Secure:   a.session.secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(handler.OAuthFlowTTL.Seconds()),
	})

	http.Redirect(w, r, redirectURL, http.StatusFound)
}

// callback completes the flow, creates or resolves the user, sets the session
// cookie and sends the browser back to the SPA.
func (a *authRoutes) callback(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	if providerError := query.Get("error"); providerError != "" {
		a.redirectWithError(w, r, providerError+": "+query.Get("error_description"))
		return
	}

	cookie, err := r.Cookie(handler.OAuthStateCookie)
	if err != nil {
		a.redirectWithError(w, r, "missing oauth state")
		return
	}
	http.SetCookie(w, &http.Cookie{Name: handler.OAuthStateCookie, Value: "", Path: "/auth", MaxAge: -1})

	state, verifier, ok := strings.Cut(cookie.Value, "|")
	if !ok || state == "" || state != query.Get("state") {
		a.redirectWithError(w, r, "invalid oauth state")
		return
	}

	code := query.Get("code")
	if code == "" {
		a.redirectWithError(w, r, "missing authorization code")
		return
	}

	user, err := a.auth.CompleteLogin(r.Context(), query.Get("provider"), code, verifier)
	if err != nil {
		a.redirectWithError(w, r, err.Error())
		return
	}

	token, err := a.signer.Sign(client.SessionClaims{UserID: user.ID, Email: user.Email})
	if err != nil {
		a.redirectWithError(w, r, "could not create session")
		return
	}
	a.session.Set(w, token)

	http.Redirect(w, r, a.appURL+"/", http.StatusFound)
}

func (a *authRoutes) redirectWithError(w http.ResponseWriter, r *http.Request, message string) {
	target := a.appURL + "/?auth_error=" + url.QueryEscape(message)
	http.Redirect(w, r, target, http.StatusFound)
}

func writeAuthError(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	_, _ = fmt.Fprintf(w, `{"error":%q}`, err.Error())
}

// requireUser rejects requests without an authenticated session.
func requireUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := handler.UserIDFromContext(r.Context()); !ok {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"not authenticated"}`))
			return
		}
		next.ServeHTTP(w, r)
	})
}
