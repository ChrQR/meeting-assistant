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

// ErrNotFound is returned when a requested entity does not exist.
var ErrNotFound = errors.New("not found")

// ErrConflict is returned when an operation is not valid for the entity state.
var ErrConflict = errors.New("conflict")

// ErrInvalidInput is returned for semantically invalid requests.
var ErrInvalidInput = errors.New("invalid input")

type storeService struct {
	dbRepository  repository.DBRepository
	storageClient client.StorageClient
}

type StoreService interface {
	GetRecordings(ctx context.Context, userID uuid.UUID) ([]types.SelectRecording, error)
	GetRecordingAudio(ctx context.Context, userID uuid.UUID, recordingID uuid.UUID) ([]byte, types.RecordingFile, error)
	GetConversations(ctx context.Context, userID uuid.UUID) ([]types.ConversationSummary, error)
	GetConversation(ctx context.Context, userID uuid.UUID, id uuid.UUID) (types.Conversation, error)
	SetSegmentIgnored(ctx context.Context, userID uuid.UUID, conversationID uuid.UUID, segmentID uuid.UUID, ignored bool) (types.ConversationSegment, error)
	GetSpeakers(ctx context.Context, userID uuid.UUID) ([]types.Speaker, error)
	GetSpeaker(ctx context.Context, userID uuid.UUID, id uuid.UUID) (types.SpeakerDetail, error)
	CreateSpeaker(ctx context.Context, userID uuid.UUID, speaker types.CreateSpeaker) (types.Speaker, error)
	UpdateSpeaker(ctx context.Context, userID uuid.UUID, id uuid.UUID, update types.UpdateSpeaker) (types.Speaker, error)
	DeleteSpeaker(ctx context.Context, userID uuid.UUID, id uuid.UUID) error
	AssignSpeaker(ctx context.Context, userID uuid.UUID, id uuid.UUID, assignment types.AssignSpeaker) (types.Speaker, error)
	GetCompanies(ctx context.Context, userID uuid.UUID) ([]types.Company, error)
	CreateCompany(ctx context.Context, userID uuid.UUID, name string) (types.Company, error)
	UpdateCompany(ctx context.Context, userID uuid.UUID, id uuid.UUID, name string) (types.Company, error)
	DeleteCompany(ctx context.Context, userID uuid.UUID, id uuid.UUID) error
}

func NewStoreService(dbRepository repository.DBRepository, storageClient client.StorageClient) StoreService {
	return &storeService{
		dbRepository:  dbRepository,
		storageClient: storageClient,
	}
}

func (s *storeService) GetRecordings(ctx context.Context, userID uuid.UUID) ([]types.SelectRecording, error) {
	recordings, err := s.dbRepository.GetRecordings(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return []types.SelectRecording{}, nil
		} else {
			return []types.SelectRecording{}, err
		}
	}
	return recordings, nil
}

func (s *storeService) GetRecordingAudio(ctx context.Context, userID uuid.UUID, recordingID uuid.UUID) ([]byte, types.RecordingFile, error) {
	file, err := s.dbRepository.GetRecordingFile(ctx, userID, recordingID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, types.RecordingFile{}, ErrNotFound
		}
		return nil, types.RecordingFile{}, err
	}

	audio, err := s.storageClient.GetRecording(ctx, file.Path, file.Filename, file.Extension)
	if err != nil {
		return nil, types.RecordingFile{}, fmt.Errorf("fetch audio from storage: %w", err)
	}
	return audio, file, nil
}

func (s *storeService) GetConversations(ctx context.Context, userID uuid.UUID) ([]types.ConversationSummary, error) {
	conversations, err := s.dbRepository.GetConversations(ctx, userID)
	if err != nil {
		return nil, err
	}
	return conversations, nil
}

func (s *storeService) GetConversation(ctx context.Context, userID uuid.UUID, id uuid.UUID) (types.Conversation, error) {
	conversation, err := s.dbRepository.GetConversation(ctx, userID, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return types.Conversation{}, ErrNotFound
		}
		return types.Conversation{}, err
	}
	return conversation, nil
}

// SetSegmentIgnored dismisses (or restores) a transcript segment. Only segments
// whose speaker is unidentified may be ignored: those are the ones that cannot
// be attributed, usually because the utterance is too short to identify a voice.
func (s *storeService) SetSegmentIgnored(ctx context.Context, userID uuid.UUID, conversationID uuid.UUID, segmentID uuid.UUID, ignored bool) (types.ConversationSegment, error) {
	segment, err := s.dbRepository.GetConversationSegment(ctx, userID, conversationID, segmentID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return types.ConversationSegment{}, ErrNotFound
		}
		return types.ConversationSegment{}, err
	}

	if segment.SpeakerName != nil {
		return types.ConversationSegment{}, fmt.Errorf("%w: segment belongs to a known speaker", ErrConflict)
	}

	if err := s.dbRepository.SetSegmentIgnored(ctx, conversationID, segmentID, ignored); err != nil {
		return types.ConversationSegment{}, err
	}

	segment.Ignored = ignored
	return segment, nil
}

func (s *storeService) GetSpeakers(ctx context.Context, userID uuid.UUID) ([]types.Speaker, error) {
	return s.dbRepository.GetSpeakers(ctx, userID)
}

func (s *storeService) GetSpeaker(ctx context.Context, userID uuid.UUID, id uuid.UUID) (types.SpeakerDetail, error) {
	speaker, err := s.dbRepository.GetSpeakerDetail(ctx, userID, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return types.SpeakerDetail{}, ErrNotFound
		}
		return types.SpeakerDetail{}, err
	}
	return speaker, nil
}

func (s *storeService) CreateSpeaker(ctx context.Context, userID uuid.UUID, input types.CreateSpeaker) (types.Speaker, error) {
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		return types.Speaker{}, fmt.Errorf("%w: speaker name is required", ErrInvalidInput)
	}

	id, err := s.dbRepository.CreateSpeaker(ctx, userID, input)
	if err != nil {
		return types.Speaker{}, err
	}
	return s.speaker(ctx, userID, id)
}

func (s *storeService) UpdateSpeaker(ctx context.Context, userID uuid.UUID, id uuid.UUID, update types.UpdateSpeaker) (types.Speaker, error) {
	if update.Name != nil {
		trimmed := strings.TrimSpace(*update.Name)
		if trimmed == "" {
			return types.Speaker{}, fmt.Errorf("%w: speaker name cannot be empty", ErrInvalidInput)
		}
		update.Name = &trimmed
	}

	if err := s.dbRepository.UpdateSpeaker(ctx, userID, id, update); err != nil {
		return types.Speaker{}, err
	}
	return s.speaker(ctx, userID, id)
}

func (s *storeService) DeleteSpeaker(ctx context.Context, userID uuid.UUID, id uuid.UUID) error {
	segmentCount, err := s.dbRepository.CountSegmentsBySpeaker(ctx, userID, id)
	if err != nil {
		return err
	}
	if segmentCount > 0 {
		return fmt.Errorf("%w: speaker still has %d segments", ErrConflict, segmentCount)
	}
	return s.dbRepository.DeleteSpeaker(ctx, userID, id)
}

// AssignSpeaker resolves an unknown (unnamed) speaker either by merging it into
// an existing speaker or by naming it in place.
func (s *storeService) AssignSpeaker(ctx context.Context, userID uuid.UUID, id uuid.UUID, assignment types.AssignSpeaker) (types.Speaker, error) {
	speaker, err := s.GetSpeaker(ctx, userID, id)
	if err != nil {
		return types.Speaker{}, err
	}
	if speaker.Name != nil {
		return types.Speaker{}, fmt.Errorf("%w: speaker is already known as %q", ErrConflict, *speaker.Name)
	}

	if assignment.TargetSpeakerID != nil {
		if *assignment.TargetSpeakerID == id {
			return types.Speaker{}, fmt.Errorf("%w: cannot merge a speaker into itself", ErrInvalidInput)
		}

		target, err := s.GetSpeaker(ctx, userID, *assignment.TargetSpeakerID)
		if err != nil {
			return types.Speaker{}, err
		}
		if target.Name == nil {
			return types.Speaker{}, fmt.Errorf("%w: target speaker is also unknown", ErrInvalidInput)
		}

		if err := s.dbRepository.WithTx(ctx, func(txCtx context.Context) error {
			if err := s.dbRepository.MoveEmbeddingsToSpeaker(txCtx, id, *assignment.TargetSpeakerID); err != nil {
				return err
			}
			if err := s.dbRepository.MoveSegmentsToSpeaker(txCtx, id, *assignment.TargetSpeakerID); err != nil {
				return err
			}
			return s.dbRepository.DeleteSpeaker(txCtx, userID, id)
		}); err != nil {
			return types.Speaker{}, err
		}

		return s.speaker(ctx, userID, *assignment.TargetSpeakerID)
	}

	if assignment.Name == nil || strings.TrimSpace(*assignment.Name) == "" {
		return types.Speaker{}, fmt.Errorf("%w: provide either target_speaker_id or name", ErrInvalidInput)
	}

	name := strings.TrimSpace(*assignment.Name)
	if err := s.dbRepository.UpdateSpeaker(ctx, userID, id, types.UpdateSpeaker{
		Name:      &name,
		Email:     assignment.Email,
		CompanyID: assignment.CompanyID,
	}); err != nil {
		return types.Speaker{}, err
	}

	return s.speaker(ctx, userID, id)
}

func (s *storeService) GetCompanies(ctx context.Context, userID uuid.UUID) ([]types.Company, error) {
	return s.dbRepository.GetCompanies(ctx, userID)
}

func (s *storeService) CreateCompany(ctx context.Context, userID uuid.UUID, name string) (types.Company, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return types.Company{}, fmt.Errorf("%w: company name is required", ErrInvalidInput)
	}
	id, err := s.dbRepository.CreateCompany(ctx, userID, name)
	if err != nil {
		return types.Company{}, err
	}
	return s.company(ctx, userID, id)
}

func (s *storeService) UpdateCompany(ctx context.Context, userID uuid.UUID, id uuid.UUID, name string) (types.Company, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return types.Company{}, fmt.Errorf("%w: company name is required", ErrInvalidInput)
	}
	if err := s.dbRepository.UpdateCompany(ctx, userID, id, name); err != nil {
		return types.Company{}, err
	}
	return s.company(ctx, userID, id)
}

func (s *storeService) DeleteCompany(ctx context.Context, userID uuid.UUID, id uuid.UUID) error {
	if err := s.dbRepository.DeleteCompany(ctx, userID, id); err != nil {
		return err
	}
	return nil
}

func (s *storeService) speaker(ctx context.Context, userID uuid.UUID, id uuid.UUID) (types.Speaker, error) {
	detail, err := s.GetSpeaker(ctx, userID, id)
	if err != nil {
		return types.Speaker{}, err
	}
	return detail.Speaker, nil
}

func (s *storeService) company(ctx context.Context, userID uuid.UUID, id uuid.UUID) (types.Company, error) {
	company, err := s.dbRepository.GetCompany(ctx, userID, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return types.Company{}, ErrNotFound
		}
		return types.Company{}, err
	}
	return company, nil
}
