package repository

import (
	"context"
	"fmt"
	"time"

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
	UpdateRecordingExtension(ctx context.Context, recordingID uuid.UUID, extension string) error
	InsertCoversation(ctx context.Context, title string, transcriptions types.Transcription, recordingID uuid.UUID, summary string, meetingAt *time.Time) (uuid.UUID, error)
	InsertSegments(ctx context.Context, segments []types.TranscriptionSegment, conversationID uuid.UUID) error
	GetBestMatchingSpeaker(ctx context.Context, userID uuid.UUID, embedding []float32) (types.IdentifiedSpeaker, error)
	InsertAnonymousSpeaker(ctx context.Context, userID uuid.UUID, embedding []float32) (uuid.UUID, error)
	PurgeEmbeddings(ctx context.Context, id uuid.UUID) error
	UpdateConversationSummary(ctx context.Context, summary string, conversationID uuid.UUID) error
	GetRecordings(ctx context.Context, userID uuid.UUID) ([]types.SelectRecording, error)
	GetRecordingFile(ctx context.Context, userID uuid.UUID, id uuid.UUID) (types.RecordingFile, error)
	GetConversations(ctx context.Context, userID uuid.UUID) ([]types.ConversationSummary, error)
	GetConversation(ctx context.Context, userID uuid.UUID, id uuid.UUID) (types.Conversation, error)
	GetSegmentsByConversation(ctx context.Context, userID uuid.UUID, conversationID uuid.UUID) ([]types.ConversationSegment, error)
	GetConversationSegment(ctx context.Context, userID uuid.UUID, conversationID uuid.UUID, segmentID uuid.UUID) (types.ConversationSegment, error)
	SetSegmentIgnored(ctx context.Context, conversationID uuid.UUID, segmentID uuid.UUID, ignored bool) error
	GetSpeakers(ctx context.Context, userID uuid.UUID) ([]types.Speaker, error)
	GetSpeakerDetail(ctx context.Context, userID uuid.UUID, id uuid.UUID) (types.SpeakerDetail, error)
	CreateSpeaker(ctx context.Context, userID uuid.UUID, speaker types.CreateSpeaker) (uuid.UUID, error)
	UpdateSpeaker(ctx context.Context, userID uuid.UUID, id uuid.UUID, update types.UpdateSpeaker) error
	DeleteSpeaker(ctx context.Context, userID uuid.UUID, id uuid.UUID) error
	CountSegmentsBySpeaker(ctx context.Context, userID uuid.UUID, id uuid.UUID) (int, error)
	MoveSegmentsToSpeaker(ctx context.Context, fromID uuid.UUID, toID uuid.UUID) error
	MoveEmbeddingsToSpeaker(ctx context.Context, fromID uuid.UUID, toID uuid.UUID) error
	GetCompanies(ctx context.Context, userID uuid.UUID) ([]types.Company, error)
	GetCompany(ctx context.Context, userID uuid.UUID, id uuid.UUID) (types.Company, error)
	CreateCompany(ctx context.Context, userID uuid.UUID, name string) (uuid.UUID, error)
	UpdateCompany(ctx context.Context, userID uuid.UUID, id uuid.UUID, name string) error
	DeleteCompany(ctx context.Context, userID uuid.UUID, id uuid.UUID) error

	// Users and linked SSO accounts.
	GetUser(ctx context.Context, id uuid.UUID) (types.AppUser, error)
	GetUserByEmail(ctx context.Context, email string) (types.AppUser, error)
	InsertUser(ctx context.Context, user types.NewUser) (types.AppUser, error)
	UpdateUserProfile(ctx context.Context, id uuid.UUID, user types.NewUser) error
	GetAccount(ctx context.Context, provider string, providerAccountID string) (types.Account, error)
	InsertAccount(ctx context.Context, userID uuid.UUID, provider string, providerAccountID string) error
	GetAccountsByUser(ctx context.Context, userID uuid.UUID) ([]types.Account, error)
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
		UserID:    &recording.UserID,
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

// UpdateRecordingExtension records the extension of the file actually stored in
// object storage, which differs from the uploaded one after normalisation.
func (r *dbRepository) UpdateRecordingExtension(ctx context.Context, recordingID uuid.UUID, extension string) error {
	db := r.getQueries(ctx)
	return db.UpdateRecordingExtension(ctx, database.UpdateRecordingExtensionParams{
		ID:        recordingID,
		Extension: extension,
	})
}

func (r *dbRepository) InsertCoversation(ctx context.Context, title string, transcription types.Transcription, recordingID uuid.UUID, summary string, meetingAt *time.Time) (uuid.UUID, error) {
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
		MeetingAt: meetingAt,
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

// speakerMatchThreshold is the minimum cosine similarity required to consider a
// voice embedding to belong to an already known speaker.
//
// Measured on real meeting audio, the same speaker scores 0.72-0.95 across
// samples while different people score 0.19-0.26, but noisy samples from
// different people can reach ~0.72. The ranges overlap, so the threshold is set
// conservatively: it recognises clear matches without ever silently merging two
// different people. Ambiguous cases are resolved by the user assigning or
// merging speakers in the UI.
const speakerMatchThreshold = 0.85

func (r *dbRepository) GetBestMatchingSpeaker(ctx context.Context, userID uuid.UUID, embedding []float32) (types.IdentifiedSpeaker, error) {
	db := r.getQueries(ctx)
	data, err := db.GetBestMatchingSpeaker(ctx, database.GetBestMatchingSpeakerParams{
		Column1:       pgvector.NewVector(embedding),
		OwnerID:       userID,
		MinSimilarity: speakerMatchThreshold,
	})
	return types.IdentifiedSpeaker{
		ID:              data.SpeakerID,
		SimilarityScore: data.SimilarityScore,
	}, err
}

func (r *dbRepository) InsertAnonymousSpeaker(ctx context.Context, userID uuid.UUID, embedding []float32) (uuid.UUID, error) {
	db := r.getQueries(ctx)
	id, err := db.InsertAnonymousSpeaker(ctx, &userID)
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

func (r *dbRepository) UpdateConversationSummary(ctx context.Context, summary string, conversationID uuid.UUID) error {
	db := r.getQueries(ctx)
	return db.UpdateConversationSummary(ctx, database.UpdateConversationSummaryParams{
		Summary: pgtype.Text{
			String: summary,
			Valid:  true,
		},
		ID: conversationID,
	})
}

func (r *dbRepository) GetRecordings(ctx context.Context, userID uuid.UUID) ([]types.SelectRecording, error) {
	db := r.getQueries(ctx)
	recs, err := db.GetRecordings(ctx, &userID)
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
