package client

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// AuthIdentity is the provider-independent result of an OAuth sign-in.
type AuthIdentity struct {
	// Provider is the stable identifier of the identity provider, e.g. "microsoft".
	Provider string
	// ProviderAccountID is the provider's stable user identifier (the `oid` claim
	// for Microsoft). It is never the email, which can change.
	ProviderAccountID string
	Email             string
	Name              string
	AvatarURL         string
}

// OAuthProvider describes one external identity provider.
type OAuthProvider interface {
	// Name returns the provider key stored on the account row.
	Name() string
	// AuthCodeURL builds the authorization URL and returns the PKCE code verifier
	// that must be replayed on exchange.
	AuthCodeURL(state string, codeChallenge string) string
	// Exchange swaps an authorization code for verified user identity.
	Exchange(ctx context.Context, code string, codeVerifier string) (AuthIdentity, error)
}

// AuthProviderRegistry resolves providers by name so additional SSO providers can
// be configured without touching the handlers.
type AuthProviderRegistry interface {
	Get(name string) (OAuthProvider, bool)
	Default() OAuthProvider
	// Names lists the configured provider keys.
	Names() []string
}

type providerRegistry struct {
	providers map[string]OAuthProvider
	order     []string
}

func NewProviderRegistry(providers ...OAuthProvider) AuthProviderRegistry {
	registry := &providerRegistry{providers: make(map[string]OAuthProvider, len(providers))}
	for _, provider := range providers {
		registry.providers[provider.Name()] = provider
		registry.order = append(registry.order, provider.Name())
	}
	return registry
}

func (r *providerRegistry) Get(name string) (OAuthProvider, bool) {
	provider, ok := r.providers[name]
	return provider, ok
}

func (r *providerRegistry) Default() OAuthProvider {
	if len(r.order) == 0 {
		return nil
	}
	return r.providers[r.order[0]]
}

func (r *providerRegistry) Names() []string {
	return append([]string(nil), r.order...)
}

// microsoftProvider implements the Microsoft identity platform (Entra ID) v2
// authorization code flow with PKCE.
type microsoftProvider struct {
	tenantID     string
	clientID     string
	clientSecret string
	redirectURI  string
	client       http.Client
}

const microsoftScopes = "openid email profile"

func NewMicrosoftProvider(tenantID, clientID, clientSecret, redirectURI string) OAuthProvider {
	return &microsoftProvider{
		tenantID:     tenantID,
		clientID:     clientID,
		clientSecret: clientSecret,
		redirectURI:  redirectURI,
		client:       http.Client{Timeout: 30 * time.Second},
	}
}

func (p *microsoftProvider) Name() string { return "microsoft" }

func (p *microsoftProvider) authorizeEndpoint() string {
	return fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/authorize", url.PathEscape(p.tenantID))
}

func (p *microsoftProvider) tokenEndpoint() string {
	return fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", url.PathEscape(p.tenantID))
}

func (p *microsoftProvider) AuthCodeURL(state string, codeChallenge string) string {
	query := url.Values{
		"client_id":             {p.clientID},
		"response_type":         {"code"},
		"redirect_uri":          {p.redirectURI},
		"response_mode":         {"query"},
		"scope":                 {microsoftScopes},
		"state":                 {state},
		"code_challenge":        {codeChallenge},
		"code_challenge_method": {"S256"},
		"prompt":                {"select_account"},
	}
	return p.authorizeEndpoint() + "?" + query.Encode()
}

type microsoftTokenResponse struct {
	IDToken string `json:"id_token"`
	Error   string `json:"error"`
	Desc    string `json:"error_description"`
}

type microsoftClaims struct {
	ObjectID          string `json:"oid"`
	Email             string `json:"email"`
	PreferredUsername string `json:"preferred_username"`
	Name              string `json:"name"`
}

func (p *microsoftProvider) Exchange(ctx context.Context, code string, codeVerifier string) (AuthIdentity, error) {
	form := url.Values{
		"client_id":     {p.clientID},
		"client_secret": {p.clientSecret},
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {p.redirectURI},
		"code_verifier": {codeVerifier},
		"scope":         {microsoftScopes},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.tokenEndpoint(), strings.NewReader(form.Encode()))
	if err != nil {
		return AuthIdentity{}, fmt.Errorf("create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.client.Do(req)
	if err != nil {
		return AuthIdentity{}, fmt.Errorf("send token request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return AuthIdentity{}, fmt.Errorf("read token response: %w", err)
	}

	var token microsoftTokenResponse
	if err := json.Unmarshal(body, &token); err != nil {
		return AuthIdentity{}, fmt.Errorf("decode token response: %w", err)
	}
	if resp.StatusCode != http.StatusOK || token.IDToken == "" {
		return AuthIdentity{}, fmt.Errorf("token exchange failed (%s): %s %s", resp.Status, token.Error, token.Desc)
	}

	claims, err := decodeJWTClaims(token.IDToken)
	if err != nil {
		return AuthIdentity{}, fmt.Errorf("decode id token: %w", err)
	}
	if claims.ObjectID == "" {
		return AuthIdentity{}, errors.New("id token is missing the oid claim")
	}

	email := claims.Email
	if email == "" {
		email = claims.PreferredUsername
	}

	return AuthIdentity{
		Provider:          p.Name(),
		ProviderAccountID: claims.ObjectID,
		Email:             email,
		Name:              claims.Name,
	}, nil
}

// decodeJWTClaims reads the claims payload of a JWT without verifying the
// signature. The token was received directly from the provider's token endpoint
// over TLS as part of the code exchange, which is the trusted path.
func decodeJWTClaims(token string) (microsoftClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return microsoftClaims{}, errors.New("malformed id token")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return microsoftClaims{}, err
	}
	var claims microsoftClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return microsoftClaims{}, err
	}
	return claims, nil
}

// NewPKCEPair returns a code verifier and its S256 challenge.
func NewPKCEPair() (verifier string, challenge string, err error) {
	random := make([]byte, 32)
	if _, err := rand.Read(random); err != nil {
		return "", "", err
	}
	verifier = base64.RawURLEncoding.EncodeToString(random)
	sum := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(sum[:])
	return verifier, challenge, nil
}

// NewRandomState returns an unguessable state value for CSRF protection.
func NewRandomState() (string, error) {
	random := make([]byte, 24)
	if _, err := rand.Read(random); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(random), nil
}
