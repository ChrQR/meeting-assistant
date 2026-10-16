package service

import (
	"context"

	"github.com/ChrQR/meeting-assistant/server/internal/client"
	"github.com/ChrQR/meeting-assistant/server/internal/repository"
	"github.com/ChrQR/meeting-assistant/server/internal/types"
)

type transcriptionService struct {
	transcriptionClient client.TranscriptionClient
	storageClient       client.StorageClient
	dbRepository        repository.DBRepository
}

type TranscriptionService interface {
	ProcessRecording(ctx context.Context, file types.Recording) error
}

func NewTranscriptionService(transcriptionClient client.TranscriptionClient, storageClient client.StorageClient, dbRepository repository.DBRepository) TranscriptionService {
	return &transcriptionService{
		transcriptionClient,
		storageClient,
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
	// match with existing speaker or create new unknown
	return nil
}
