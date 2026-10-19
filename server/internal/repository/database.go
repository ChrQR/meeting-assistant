package repository

import (
	"context"
	"fmt"

	"github.com/ChrQR/meeting-assistant/server/internal/database"
	"github.com/ChrQR/meeting-assistant/server/internal/types"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/pgvector/pgvector-go"
)

type dbRepository struct {
	db *database.DB
}

type DBRepository interface {
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
	InsertRecording(ctx context.Context, recording types.Recording) (uuid.UUID, error)
	UpdateRecordingStatus(ctx context.Context, recordingID uuid.UUID, status types.RecordingStatus) error
	InsertCoversation(ctx context.Context, title string, transcriptions types.Transcription, recordingID uuid.UUID, summary string) (uuid.UUID, error)
	InsertSegments(ctx context.Context, segments []types.TranscriptionSegment, conversationID uuid.UUID) error
	GetBestMatchingSpeaker(ctx context.Context, embedding []float32) (types.IdentifiedSpeaker, error)
	InsertAnonymousSpeaker(ctx context.Context, embedding []float32) (uuid.UUID, error)
	PurgeEmbeddings(ctx context.Context, id uuid.UUID) error
	GetRecordings(ctx context.Context) ([]types.SelectRecording, error)
	GetAllConversations(ctx context.Context) ([]types.SelectConversation, error)
}

func NewDBRepository(db *database.DB) DBRepository {
	return &dbRepository{
		db,
	}
}

type txKey struct{}

func (r *dbRepository) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := r.db.Conn.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	defer tx.Rollback(ctx)

	txCtx := context.WithValue(ctx, txKey{}, tx)
	if err := fn(txCtx); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *dbRepository) getQueries(ctx context.Context) *database.Queries {
	if tx, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return r.db.WithTx(tx)
	}
	return r.db.Queries
}

func (r *dbRepository) InsertRecording(ctx context.Context, recording types.Recording) (uuid.UUID, error) {
	db := r.getQueries(ctx)

	recordingID, err := db.InsertRecording(ctx, database.InsertRecordingParams{
		Filename:  recording.FileName,
		Extension: recording.Extension,
		Path:      recording.UserPath,
		Status:    string(types.RecordingTranscribing),
	})
	return recordingID, err
}

func (r *dbRepository) UpdateRecordingStatus(ctx context.Context, recordingID uuid.UUID, status types.RecordingStatus) error {
	db := r.getQueries(ctx)
	return db.UpdateRecordingStatus(ctx, database.UpdateRecordingStatusParams{
		ID:     recordingID,
		Status: string(status),
	})
}

func (r *dbRepository) InsertCoversation(ctx context.Context, title string, transcription types.Transcription, recordingID uuid.UUID, summary string) (uuid.UUID, error) {
	db := r.getQueries(ctx)
	conversationID, err := db.InsertConversation(ctx, database.InsertConversationParams{
		Title:              title,
		Length:             transcription.Duration,
		Cost:               transcription.Usage.Cost,
		Language:           transcription.Language,
		LanguageConfidence: transcription.LanguageConfidence,
		RecordingID:        recordingID,
		Summary: pgtype.Text{
			String: summary,
			Valid:  true,
		},
	})
	return conversationID, err
}

func (r *dbRepository) InsertSegments(ctx context.Context, segments []types.TranscriptionSegment, conversationID uuid.UUID) error {
	db := r.getQueries(ctx)
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
			SpeakerID:      segment.SpeakerID,
		})
	}
	_, err := db.InsertSegments(ctx, dbSegments)
	return err
}

func (r *dbRepository) GetBestMatchingSpeaker(ctx context.Context, embedding []float32) (types.IdentifiedSpeaker, error) {
	db := r.getQueries(ctx)
	data, err := db.GetBestMatchingSpeaker(ctx, pgvector.NewVector(embedding))
	return types.IdentifiedSpeaker{
		ID:              data.SpeakerID,
		SimilarityScore: data.SimilarityScore,
	}, err
}

func (r *dbRepository) InsertAnonymousSpeaker(ctx context.Context, embedding []float32) (uuid.UUID, error) {
	db := r.getQueries(ctx)
	id, err := db.InsertAnonymousSpeaker(ctx)
	if err != nil {
		return id, err
	}
	_, err = r.db.InsertSpeakerEmbedding(ctx, database.InsertSpeakerEmbeddingParams{
		SpeakerID: id,
		Column2:   pgvector.NewVector(embedding),
	})
	if err != nil {
		return id, nil
	}

	return id, nil
}

func (r *dbRepository) PurgeEmbeddings(ctx context.Context, id uuid.UUID) error {
	db := r.getQueries(ctx)
	count, err := db.CountSpeakerEmbeddings(ctx, id)
	if err != nil {
		return err
	}
	if count > 5 {
		if err := r.db.PruneMostRedundantSpeakerEmbedding(ctx, id); err != nil {
			return err
		}
	}
	return nil
}

func (r *dbRepository) GetRecordings(ctx context.Context) ([]types.SelectRecording, error) {
	db := r.getQueries(ctx)
	recs, err := db.GetRecordings(ctx)
	if err != nil {
		return []types.SelectRecording{}, err
	}
	var data []types.SelectRecording
	for _, rec := range recs {
		data = append(data, types.SelectRecording{
			ID:        rec.ID,
			Extension: rec.Extension,
			Filename:  rec.Filename,
			Status:    types.RecordingStatus(rec.Status),
		})
	}
	return data, nil
}

func (r *dbRepository) GetAllConversations(ctx context.Context) ([]types.SelectConversation, error) {
	data, err := r.db.GetAllConversations(ctx)
	if err != nil {
		return []types.SelectConversation{}, nil
	}
	var result []types.SelectConversation
	for _, conv := range data {
		result = append(result, types.SelectConversation{
			ID:                 conv.ID,
			Title:              conv.Title,
			Summary:            conv.Summary,
			Length:             conv.Length,
			Cost:               conv.Length,
			Language:           conv.Language,
			LanguageConfidence: conv.LanguageConfidence,
			RecordingID:        conv.RecordingID,
			CreatedAt:          conv.CreatedAt,
			UpdatedAt:          conv.UpdatedAt,
		})
	}
	return result, nil
}
