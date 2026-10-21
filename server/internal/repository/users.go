package repository

import (
	"context"

	"github.com/google/uuid"

	"github.com/ChrQR/meeting-assistant/server/internal/database"
	"github.com/ChrQR/meeting-assistant/server/internal/types"
)

func mapUser(row database.AppUser) types.AppUser {
	return types.AppUser{
		ID:        row.ID,
		Email:     row.Email,
		Name:      textPtr(row.Name),
		AvatarURL: textPtr(row.AvatarUrl),
		CreatedAt: row.CreatedAt,
	}
}

func (r *dbRepository) GetUser(ctx context.Context, id uuid.UUID) (types.AppUser, error) {
	db := r.getQueries(ctx)
	row, err := db.GetUser(ctx, id)
	if err != nil {
		return types.AppUser{}, err
	}
	return mapUser(row), nil
}

func (r *dbRepository) GetUserByEmail(ctx context.Context, email string) (types.AppUser, error) {
	db := r.getQueries(ctx)
	row, err := db.GetUserByEmail(ctx, email)
	if err != nil {
		return types.AppUser{}, err
	}
	return mapUser(row), nil
}

func (r *dbRepository) InsertUser(ctx context.Context, user types.NewUser) (types.AppUser, error) {
	db := r.getQueries(ctx)
	row, err := db.InsertUser(ctx, database.InsertUserParams{
		Email:     user.Email,
		Name:      optionalText(user.Name),
		AvatarUrl: optionalText(user.AvatarURL),
	})
	if err != nil {
		return types.AppUser{}, err
	}
	return mapUser(row), nil
}

func (r *dbRepository) UpdateUserProfile(ctx context.Context, id uuid.UUID, user types.NewUser) error {
	db := r.getQueries(ctx)
	return db.UpdateUserProfile(ctx, database.UpdateUserProfileParams{
		ID:        id,
		Email:     user.Email,
		Name:      optionalText(user.Name),
		AvatarUrl: optionalText(user.AvatarURL),
	})
}

func mapAccount(row database.Account) types.Account {
	return types.Account{
		ID:                row.ID,
		UserID:            row.UserID,
		Provider:          row.Provider,
		ProviderAccountID: row.ProviderAccountID,
	}
}

func (r *dbRepository) GetAccount(ctx context.Context, provider string, providerAccountID string) (types.Account, error) {
	db := r.getQueries(ctx)
	row, err := db.GetAccount(ctx, database.GetAccountParams{
		Provider:          provider,
		ProviderAccountID: providerAccountID,
	})
	if err != nil {
		return types.Account{}, err
	}
	return mapAccount(row), nil
}

func (r *dbRepository) InsertAccount(ctx context.Context, userID uuid.UUID, provider string, providerAccountID string) error {
	db := r.getQueries(ctx)
	_, err := db.InsertAccount(ctx, database.InsertAccountParams{
		UserID:            userID,
		Provider:          provider,
		ProviderAccountID: providerAccountID,
	})
	return err
}

func (r *dbRepository) GetAccountsByUser(ctx context.Context, userID uuid.UUID) ([]types.Account, error) {
	db := r.getQueries(ctx)
	rows, err := db.GetAccountsByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	accounts := make([]types.Account, 0, len(rows))
	for _, row := range rows {
		accounts = append(accounts, mapAccount(row))
	}
	return accounts, nil
}
