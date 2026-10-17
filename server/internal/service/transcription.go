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

	if err := s.dbRepository.UpdateRecordingStatus(ctx, recordingID, types.RecordingEmbedding); err != nil {
		return err
	}
	// request embedding
	embeddings, err := s.embeddingClient.EmbedSpeakers(ctx, file.FileData, transcription.Speakers)
	if err != nil {
		return err
	}

	if err := s.dbRepository.UpdateRecordingStatus(ctx, recordingID, types.RecordingAssigningSpeakers); err != nil {
		return err
	}

	mappedSpeakers := make(map[string]uuid.UUID)
	for _, embedding := range embeddings.Data {
		var speakerID uuid.UUID

		speaker, err := s.dbRepository.GetBestMatchingSpeaker(ctx, embedding.Embedding)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				newID, insertErr := s.dbRepository.InsertAnonymousSpeaker(ctx, embedding.Embedding)
				if insertErr != nil {
					log.Printf("Failed to insert speaker with label %s: %v", embedding.SpeakerLabel, insertErr)
					return insertErr
				}
				speakerID = newID
			} else {
				log.Printf("error fetching similar speaker: %v", err)
				return err
			}
		} else {
			speakerID = speaker.ID
			if err := s.dbRepository.PurgeEmbeddings(ctx, speakerID); err != nil {
				return err
			}
		}

		mappedSpeakers[embedding.SpeakerLabel] = speakerID
	}

	if err := s.dbRepository.UpdateRecordingStatus(ctx, recordingID, types.RecordingCompleted); err != nil {
		return err
	}

	// Update transcription segments with speakerID
	for i := range transcription.Segments {
		transcription.Segments[i].SpeakerID = mappedSpeakers[transcription.Segments[i].Speaker]
	}

	if err := s.dbRepository.UpdateRecordingStatus(ctx, recordingID, types.RecordingSummaizing); err != nil {
		return err
	}

	// Request Summary
	summary, err := s.summaryClient.GetSummary(ctx, transcription)
	if err != nil {
		return err
	}

	// Insert Data
	// TODO: Insert in transaction
	if err = s.dbRepository.WithTx(ctx, func(txCtx context.Context) error {
		conversationID, err := s.dbRepository.InsertCoversation(ctx, file.Title, transcription, recordingID, summary)
		if err != nil {
			return err
		}

		if err = s.dbRepository.InsertSegments(ctx, transcription.Segments, conversationID); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}

	return nil
}
