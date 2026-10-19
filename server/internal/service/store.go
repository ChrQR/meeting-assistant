package service

import (
	"context"
	"errors"

	"github.com/ChrQR/meeting-assistant/server/internal/repository"
	"github.com/ChrQR/meeting-assistant/server/internal/types"
	"github.com/jackc/pgx/v5"
)

type storeService struct {
	dbRepository repository.DBRepository
}

type StoreService interface {
	GetRecordings(ctx context.Context) ([]types.SelectRecording, error)
	GetConversationList(ctx context.Context) ([]types.SelectConversation, error)
}

func NewStoreService(dbRepository repository.DBRepository) StoreService {
	return &storeService{
		dbRepository,
	}
}

func (s *storeService) GetRecordings(ctx context.Context) ([]types.SelectRecording, error) {
	recordings, err := s.dbRepository.GetRecordings(ctx)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return []types.SelectRecording{}, nil
		} else {
			return []types.SelectRecording{}, err
		}
	}
	return recordings, nil
}

func (s *storeService) GetConversationList(ctx context.Context) ([]types.SelectConversation, error) {
	return s.dbRepository.GetAllConversations(ctx)
}
