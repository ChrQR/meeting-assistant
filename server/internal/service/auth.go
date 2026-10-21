package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/ChrQR/meeting-assistant/server/internal/client"
	"github.com/ChrQR/meeting-assistant/server/internal/repository"
	"github.com/ChrQR/meeting-assistant/server/internal/types"
)

// ErrProviderUnknown is returned when a login is requested for an unconfigured provider.
var ErrProviderUnknown = errors.New("unknown identity provider")

type authService struct {
	providers client.AuthProviderRegistry
	db        repository.DBRepository
}

type AuthService interface {
	// AuthorizeURL builds the provider redirect URL plus the PKCE verifier and
	// state that must be stored in the client session during the flow.
	AuthorizeURL(provider string) (redirectURL string, codeVerifier string, state string, err error)
	// CompleteLogin exchanges the authorization code and returns the local user.
	CompleteLogin(ctx context.Context, provider string, code string, codeVerifier string) (types.AppUser, error)
	// GetUser loads a user by id.
	GetUser(ctx context.Context, id uuid.UUID) (types.AppUser, error)
	// Providers lists the configured provider keys.
	Providers() []string
}

func NewAuthService(db repository.DBRepository, providers client.AuthProviderRegistry) AuthService {
	return &authService{providers: providers, db: db}
}

func (s *authService) Providers() []string {
	return s.providers.Names()
}

func (s *authService) AuthorizeURL(name string) (string, string, string, error) {
	provider, err := s.provider(name)
	if err != nil {
		return "", "", "", err
	}

	verifier, challenge, err := client.NewPKCEPair()
	if err != nil {
		return "", "", "", fmt.Errorf("generate pkce: %w", err)
	}
	state, err := client.NewRandomState()
	if err != nil {
		return "", "", "", fmt.Errorf("generate state: %w", err)
	}

	return provider.AuthCodeURL(state, challenge), verifier, state, nil
}

func (s *authService) CompleteLogin(ctx context.Context, name string, code string, codeVerifier string) (types.AppUser, error) {
	provider, err := s.provider(name)
	if err != nil {
		return types.AppUser{}, err
	}

	identity, err := provider.Exchange(ctx, code, codeVerifier)
	if err != nil {
		return types.AppUser{}, err
	}

	return s.ensureUser(ctx, identity)
}

func (s *authService) GetUser(ctx context.Context, id uuid.UUID) (types.AppUser, error) {
	user, err := s.db.GetUser(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return types.AppUser{}, ErrNotFound
		}
		return types.AppUser{}, err
	}
	return user, nil
}

// ensureUser resolves the provider identity to a local user, creating one on
// first sign-in. Keeping identity resolution in the account table means new
// providers only add rows, never columns.
func (s *authService) ensureUser(ctx context.Context, identity client.AuthIdentity) (types.AppUser, error) {
	account, err := s.db.GetAccount(ctx, identity.Provider, identity.ProviderAccountID)
	switch {
	case err == nil:
		// Existing link: refresh the cached profile so names and emails stay current.
		user, err := s.db.GetUser(ctx, account.UserID)
		if err != nil {
			return types.AppUser{}, err
		}
		if err := s.db.UpdateUserProfile(ctx, user.ID, profileFrom(identity)); err != nil {
			return types.AppUser{}, err
		}
		user, err = s.db.GetUser(ctx, user.ID)
		return user, err
	case !errors.Is(err, pgx.ErrNoRows):
		return types.AppUser{}, err
	}

	if identity.Email == "" {
		return types.AppUser{}, fmt.Errorf("%w: provider did not return an email address", ErrInvalidInput)
	}

	var user types.AppUser
	err = s.db.WithTx(ctx, func(txCtx context.Context) error {
		existing, err := s.db.GetUserByEmail(txCtx, identity.Email)
		switch {
		case err == nil:
			user = existing
			if err := s.db.UpdateUserProfile(txCtx, existing.ID, profileFrom(identity)); err != nil {
				return err
			}
			user, err = s.db.GetUser(txCtx, existing.ID)
			return err
		case !errors.Is(err, pgx.ErrNoRows):
			return err
		}

		created, err := s.db.InsertUser(txCtx, profileFrom(identity))
		if err != nil {
			return err
		}
		user = created

		return s.db.InsertAccount(txCtx, created.ID, identity.Provider, identity.ProviderAccountID)
	})
	if err != nil {
		return types.AppUser{}, err
	}

	return user, nil
}

func profileFrom(identity client.AuthIdentity) types.NewUser {
	profile := types.NewUser{Email: strings.ToLower(identity.Email)}
	if name := strings.TrimSpace(identity.Name); name != "" {
		profile.Name = &name
	}
	if avatar := strings.TrimSpace(identity.AvatarURL); avatar != "" {
		profile.AvatarURL = &avatar
	}
	return profile
}

func (s *authService) provider(name string) (client.OAuthProvider, error) {
	if strings.TrimSpace(name) == "" {
		if provider := s.providers.Default(); provider != nil {
			return provider, nil
		}
		return nil, ErrProviderUnknown
	}

	provider, ok := s.providers.Get(name)
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrProviderUnknown, name)
	}
	return provider, nil
}
