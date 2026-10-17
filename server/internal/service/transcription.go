package service

import (
	"context"
	"errors"
	"log"

	"github.com/ChrQR/meeting-assistant/server/internal/client"
	"github.com/ChrQR/meeting-assistant/server/internal/repository"
	"github.com/ChrQR/meeting-assistant/server/internal/types"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type transcriptionService struct {
	transcriptionClient client.TranscriptionClient
	storageClient       client.StorageClient
	embeddingClient     client.EmbeddingClient
	summaryClient       client.SummaryClient
	dbRepository        repository.DBRepository
}

type TranscriptionService interface {
	ProcessRecording(ctx context.Context, file types.Recording) error
}

func NewTranscriptionService(
	transcriptionClient client.TranscriptionClient,
	storageClient client.StorageClient,
	embeddingClient client.EmbeddingClient,
	summaryClient client.SummaryClient,
	dbRepository repository.DBRepository,
) TranscriptionService {
	return &transcriptionService{
		transcriptionClient,
		storageClient,
		embeddingClient,
		summaryClient,
		dbRepository,
	}
}

func (s *transcriptionService) ProcessRecording(ctx context.Context, file types.Recording) error {
	// upload to s3
	if err := s.storageClient.InsertRecording(ctx, file.FileData, file.UserPath, file.FileName, file.Extension); err != nil {
		return err
	}

	recordingID, err := s.dbRepository.InsertRecording(ctx, file)
	if err != nil {
		return err
	}
	// send to elevenlabs for transcription
	transcription, err := s.transcriptionClient.Transcribe(file.FileData, file.Extension)
	if err != nil {
		return err
	}
	// inserts transcription and update recording status
	conversationID, err := s.dbRepository.InsertCoversation(ctx, file.Title, transcription, recordingID)
	if err != nil {
		return err
	}

	if err = s.dbRepository.InsertSegments(ctx, transcription.Segments, conversationID); err != nil {
		return nil
	}

	if err = s.dbRepository.UpdateRecordingStatus(ctx, recordingID, types.RecordingEmbedding); err != nil {
		return err
	}

	// request embedding
	embeddings, err := s.embeddingClient.EmbedSpeakers(ctx, file.FileData, transcription.Speakers)
	if err != nil {
		return err
	}

	mappedSpeakers := make(map[string]uuid.UUID)

	for _, embedding := range embeddings.Data {
		speaker, err := s.dbRepository.GetBestMatchikngSpeaker(ctx, embedding.Embedding)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				// If no speakers returned, insert an unknow speaker.
				ID, err := s.dbRepository.InsertAnonymousSpeaker(ctx, embedding.Embedding)
				if err != nil {
					log.Printf("Failed to insert speaker with label %s: %v", embedding.SpeakerLabel, err)
				}
				mappedSpeakers[embedding.SpeakerLabel] = ID
			} else {
				log.Printf("error fetching similar speaker: %v", err)
				return err
			}
		}
		if err := s.dbRepository.PurgeEmbeddings(ctx, speaker.ID); err != nil {
			return err
		}
		mappedSpeakers[embedding.SpeakerLabel] = speaker.ID
	}

	// Update segments with speakerID
	for speakerLabel, speakerID := range mappedSpeakers {
		if err := s.dbRepository.UpdateSegmentSpeaker(ctx, conversationID, speakerLabel, speakerID); err != nil {
			return err
		}
	}

	// Request Summary
	summary, err := s.summaryClient.GetSummary(ctx, transcription)
	if err != nil {
		return err
	}

	// Insert Summary
	if err := s.dbRepository.UpdateConversationSummary(ctx, summary, conversationID); err != nil {
		return err
	}

	return nil
}
