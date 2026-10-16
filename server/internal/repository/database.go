package repository

import (
	"context"

	"github.com/ChrQR/meeting-assistant/server/internal/database"
	"github.com/ChrQR/meeting-assistant/server/internal/types"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/pgvector/pgvector-go"
)

type dbRepository struct {
	db *database.DB
}

type DBRepository interface {
	InsertRecording(ctx context.Context, recording types.Recording) (uuid.UUID, error)
	UpdateRecordingStatus(ctx context.Context, recordingID uuid.UUID, status types.RecordingStatus) error
	InsertCoversation(ctx context.Context, title string, transcriptions types.Transcription, recordingID uuid.UUID) (uuid.UUID, error)
	InsertSegments(ctx context.Context, segments []types.TranscriptionSegment, conversationID uuid.UUID) error
	GetSimilarSpeaker(ctx context.Context, embedding []float32) (types.IdentifiedSpeaker, error)
	InsertAnonymousSpeaker(ctx context.Context, embedding []float32) (uuid.UUID, error)
}

func NewDBRepository(db *database.DB) DBRepository {
	return &dbRepository{
		db,
	}
}

func (r *dbRepository) InsertRecording(ctx context.Context, recording types.Recording) (uuid.UUID, error) {
	recordingID, err := r.db.InsertRecording(ctx, database.InsertRecordingParams{
		Filename:  recording.FileName,
		Extension: recording.Extension,
		Path:      recording.UserPath,
		Status:    string(types.RecordingTranscribing),
	})
	return recordingID, err
}

func (r *dbRepository) UpdateRecordingStatus(ctx context.Context, recordingID uuid.UUID, status types.RecordingStatus) error {
	return r.db.UpdateRecordingStatus(ctx, database.UpdateRecordingStatusParams{
		ID:     recordingID,
		Status: string(status),
	})
}

func (r *dbRepository) InsertCoversation(ctx context.Context, title string, transcription types.Transcription, recordingID uuid.UUID) (uuid.UUID, error) {
	conversationID, err := r.db.InsertConversation(ctx, database.InsertConversationParams{
		Title:              title,
		Length:             transcription.Duration,
		Cost:               transcription.Usage.Cost,
		Language:           transcription.Language,
		LanguageConfidence: transcription.LanguageConfidence,
		RecordingID:        recordingID,
	})
	return conversationID, err
}

func (r *dbRepository) InsertSegments(ctx context.Context, segments []types.TranscriptionSegment, conversationID uuid.UUID) error {
	var dbSegments []database.InsertSegmentsParams
	for _, segment := range segments {
		dbSegments = append(dbSegments, database.InsertSegmentsParams{
			Text: pgtype.Text{
				String: segment.Text,
				Valid:  true,
			},
			StartTime:      pgtype.Float8{Float64: segment.Start, Valid: true},
			EndTime:        pgtype.Float8{Float64: segment.End, Valid: true},
			Duration:       pgtype.Float8{Float64: segment.Duration, Valid: true},
			ConversationID: conversationID,
			SpeakerLabel:   segment.Speaker,
		})
	}
	_, err := r.db.InsertSegments(ctx, dbSegments)
	return err
}

func (r *dbRepository) GetSimilarSpeaker(ctx context.Context, embedding []float32) (types.IdentifiedSpeaker, error) {
	data, err := r.db.GetSimilarSpeaker(ctx, pgvector.NewVector(embedding))
	return types.IdentifiedSpeaker{
		ID:              data.ID,
		SimilarityScore: data.SimilarityScore,
	}, err
}

func (r *dbRepository) InsertAnonymousSpeaker(ctx context.Context, embedding []float32) (uuid.UUID, error) {
	return r.db.InsertAnonymousSpeaker(ctx, pgvector.NewVector(embedding))
}
