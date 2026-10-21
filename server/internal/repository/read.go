package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ChrQR/meeting-assistant/server/internal/database"
	"github.com/ChrQR/meeting-assistant/server/internal/types"
)

func (r *dbRepository) GetRecordingFile(ctx context.Context, userID uuid.UUID, id uuid.UUID) (types.RecordingFile, error) {
	db := r.getQueries(ctx)
	rec, err := db.GetRecording(ctx, database.GetRecordingParams{ID: id, UserID: &userID})
	if err != nil {
		return types.RecordingFile{}, err
	}
	return types.RecordingFile{
		ID:        rec.ID,
		Path:      rec.Path,
		Filename:  rec.Filename,
		Extension: rec.Extension,
	}, nil
}

func (r *dbRepository) GetConversations(ctx context.Context, userID uuid.UUID) ([]types.ConversationSummary, error) {
	db := r.getQueries(ctx)
	rows, err := db.GetConversations(ctx, &userID)
	if err != nil {
		return nil, err
	}
	return mapConversationSummaries(rows), nil
}

func mapConversationSummaries(rows []database.GetConversationsRow) []types.ConversationSummary {
	data := make([]types.ConversationSummary, 0, len(rows))
	for _, row := range rows {
		data = append(data, mapConversationSummary(row))
	}
	return data
}

func mapConversationSummary(row database.GetConversationsRow) types.ConversationSummary {
	var summary *string
	if row.Summary.Valid {
		summary = &row.Summary.String
	}
	return types.ConversationSummary{
		ID:                  row.ID,
		Title:               row.Title,
		Summary:             summary,
		Length:              row.Length,
		Cost:                row.Cost,
		Language:            row.Language,
		CreatedAt:           row.CreatedAt,
		MeetingAt:           row.MeetingAt,
		RecordingID:         row.RecordingID,
		RecordingStatus:     row.RecordingStatus,
		SpeakerCount:        int(row.SpeakerCount),
		UnknownSpeakerCount: int(row.UnknownSpeakerCount),
	}
}

func (r *dbRepository) GetConversation(ctx context.Context, userID uuid.UUID, id uuid.UUID) (types.Conversation, error) {
	db := r.getQueries(ctx)
	row, err := db.GetConversation(ctx, database.GetConversationParams{ID: id, UserID: &userID})
	if err != nil {
		return types.Conversation{}, err
	}

	segments, err := r.GetSegmentsByConversation(ctx, userID, id)
	if err != nil {
		return types.Conversation{}, err
	}

	return types.Conversation{
		ConversationSummary: mapConversationSummary(database.GetConversationsRow(row)),
		Segments:            segments,
	}, nil
}

func (r *dbRepository) GetSegmentsByConversation(ctx context.Context, userID uuid.UUID, conversationID uuid.UUID) ([]types.ConversationSegment, error) {
	db := r.getQueries(ctx)
	rows, err := db.GetSegmentsByConversation(ctx, database.GetSegmentsByConversationParams{
		ConversationID: conversationID,
		UserID:         &userID,
	})
	if err != nil {
		return nil, err
	}
	segments := make([]types.ConversationSegment, 0, len(rows))
	for _, row := range rows {
		var text string
		if row.Text.Valid {
			text = row.Text.String
		}
		var name *string
		if row.SpeakerName.Valid {
			name = &row.SpeakerName.String
		}
		segments = append(segments, types.ConversationSegment{
			ID:          row.ID,
			Text:        text,
			StartTime:   row.StartTime.Float64,
			EndTime:     row.EndTime.Float64,
			Duration:    row.Duration.Float64,
			SpeakerID:   row.SpeakerID,
			SpeakerName: name,
			Ignored:     row.Ignored,
		})
	}
	return segments, nil
}

func mapSpeaker(row database.GetSpeakerRow) types.Speaker {
	speaker := types.Speaker{
		ID:                row.ID,
		Email:             textPtr(row.Email),
		CompanyID:         row.CompanyID,
		CompanyName:       textPtr(row.CompanyName),
		UserID:            row.LinkedUserID,
		UserEmail:         textPtr(row.UserEmail),
		EmbeddingCount:    int(row.EmbeddingCount),
		SegmentCount:      int(row.SegmentCount),
		TotalSpeakingTime: row.TotalSpeakingTime,
		ConversationCount: int(row.ConversationCount),
		CreatedAt:         row.CreatedAt,
	}
	name := textPtr(row.Name)
	speaker.Name = name
	return speaker
}

func (r *dbRepository) GetSpeakers(ctx context.Context, userID uuid.UUID) ([]types.Speaker, error) {
	db := r.getQueries(ctx)
	rows, err := db.GetSpeakers(ctx, &userID)
	if err != nil {
		return nil, err
	}
	speakers := make([]types.Speaker, 0, len(rows))
	for _, row := range rows {
		speakers = append(speakers, mapSpeaker(database.GetSpeakerRow(row)))
	}
	return speakers, nil
}

func (r *dbRepository) GetSpeakerDetail(ctx context.Context, userID uuid.UUID, id uuid.UUID) (types.SpeakerDetail, error) {
	db := r.getQueries(ctx)
	row, err := db.GetSpeaker(ctx, database.GetSpeakerParams{ID: id, OwnerID: &userID})
	if err != nil {
		return types.SpeakerDetail{}, err
	}

	embeddingRows, err := db.GetSpeakerEmbeddings(ctx, id)
	if err != nil {
		return types.SpeakerDetail{}, err
	}
	embeddings := make([]types.SpeakerEmbedding, 0, len(embeddingRows))
	for _, embedding := range embeddingRows {
		embeddings = append(embeddings, types.SpeakerEmbedding{
			ID:        embedding.ID,
			CreatedAt: embedding.CreatedAt,
		})
	}

	segmentRows, err := db.GetSegmentsBySpeaker(ctx, database.GetSegmentsBySpeakerParams{
		SpeakerID: id,
		OwnerID:   &userID,
	})
	if err != nil {
		return types.SpeakerDetail{}, err
	}
	segments := make([]types.SpeakerSegment, 0, len(segmentRows))
	for _, segment := range segmentRows {
		var text string
		if segment.Text.Valid {
			text = segment.Text.String
		}
		segments = append(segments, types.SpeakerSegment{
			ID:                segment.ID,
			Text:              text,
			StartTime:         segment.StartTime.Float64,
			EndTime:           segment.EndTime.Float64,
			ConversationID:    segment.ConversationID,
			ConversationTitle: segment.ConversationTitle,
		})
	}

	return types.SpeakerDetail{
		Speaker:    mapSpeaker(row),
		Embeddings: embeddings,
		Segments:   segments,
	}, nil
}

func (r *dbRepository) CreateSpeaker(ctx context.Context, userID uuid.UUID, speaker types.CreateSpeaker) (uuid.UUID, error) {
	db := r.getQueries(ctx)
	return db.InsertSpeaker(ctx, database.InsertSpeakerParams{
		Name:      pgtype.Text{String: speaker.Name, Valid: true},
		Email:     optionalText(speaker.Email),
		CompanyID: speaker.CompanyID,
		OwnerID:   &userID,
	})
}

func (r *dbRepository) UpdateSpeaker(ctx context.Context, userID uuid.UUID, id uuid.UUID, update types.UpdateSpeaker) error {
	db := r.getQueries(ctx)
	current, err := db.GetSpeaker(ctx, database.GetSpeakerParams{ID: id, OwnerID: &userID})
	if err != nil {
		return err
	}
	return db.UpdateSpeaker(ctx, database.UpdateSpeakerParams{
		ID:           current.ID,
		Name:         optionalText(update.Name),
		Email:        optionalText(update.Email),
		ClearCompany: pgtype.Bool{Bool: update.ClearCompany, Valid: update.ClearCompany},
		CompanyID:    update.CompanyID,
		LinkUserID:   update.UserID,
		ClearUser:    pgtype.Bool{Bool: update.ClearUser, Valid: update.ClearUser},
	})
}

func (r *dbRepository) DeleteSpeaker(ctx context.Context, userID uuid.UUID, id uuid.UUID) error {
	db := r.getQueries(ctx)
	return db.DeleteSpeaker(ctx, database.DeleteSpeakerParams{ID: id, OwnerID: &userID})
}

func (r *dbRepository) CountSegmentsBySpeaker(ctx context.Context, userID uuid.UUID, id uuid.UUID) (int, error) {
	db := r.getQueries(ctx)
	count, err := db.CountSegmentsBySpeaker(ctx, database.CountSegmentsBySpeakerParams{SpeakerID: id, OwnerID: &userID})
	return int(count), err
}

func (r *dbRepository) MoveSegmentsToSpeaker(ctx context.Context, fromID uuid.UUID, toID uuid.UUID) error {
	db := r.getQueries(ctx)
	return db.MoveSegmentsToSpeaker(ctx, database.MoveSegmentsToSpeakerParams{
		SpeakerID:   fromID,
		SpeakerID_2: toID,
	})
}

func (r *dbRepository) MoveEmbeddingsToSpeaker(ctx context.Context, fromID uuid.UUID, toID uuid.UUID) error {
	db := r.getQueries(ctx)
	return db.MoveEmbeddingsToSpeaker(ctx, database.MoveEmbeddingsToSpeakerParams{
		SpeakerID:   fromID,
		SpeakerID_2: toID,
	})
}

func mapCompany(row database.GetCompaniesRow) types.Company {
	company := types.Company{
		ID:           row.ID,
		SpeakerCount: int(row.SpeakerCount),
		CreatedAt:    row.CreatedAt,
	}
	if row.Name.Valid {
		company.Name = row.Name.String
	}
	return company
}

func (r *dbRepository) GetCompanies(ctx context.Context, userID uuid.UUID) ([]types.Company, error) {
	db := r.getQueries(ctx)
	rows, err := db.GetCompanies(ctx, &userID)
	if err != nil {
		return nil, err
	}
	companies := make([]types.Company, 0, len(rows))
	for _, row := range rows {
		companies = append(companies, mapCompany(row))
	}
	return companies, nil
}

func (r *dbRepository) GetCompany(ctx context.Context, userID uuid.UUID, id uuid.UUID) (types.Company, error) {
	db := r.getQueries(ctx)
	row, err := db.GetCompany(ctx, database.GetCompanyParams{ID: id, UserID: &userID})
	if err != nil {
		return types.Company{}, err
	}
	return mapCompany(database.GetCompaniesRow(row)), nil
}

func (r *dbRepository) CreateCompany(ctx context.Context, userID uuid.UUID, name string) (uuid.UUID, error) {
	db := r.getQueries(ctx)
	return db.InsertCompany(ctx, database.InsertCompanyParams{
		Name:   pgtype.Text{String: name, Valid: true},
		UserID: &userID,
	})
}

func (r *dbRepository) UpdateCompany(ctx context.Context, userID uuid.UUID, id uuid.UUID, name string) error {
	db := r.getQueries(ctx)
	return db.UpdateCompany(ctx, database.UpdateCompanyParams{
		ID:     id,
		Name:   pgtype.Text{String: name, Valid: true},
		UserID: &userID,
	})
}

func (r *dbRepository) DeleteCompany(ctx context.Context, userID uuid.UUID, id uuid.UUID) error {
	db := r.getQueries(ctx)
	return db.DeleteCompany(ctx, database.DeleteCompanyParams{ID: id, UserID: &userID})
}

func textPtr(value pgtype.Text) *string {
	if !value.Valid {
		return nil
	}
	return &value.String
}

func optionalText(value *string) pgtype.Text {
	if value == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *value, Valid: true}
}

func (r *dbRepository) GetConversationSegment(ctx context.Context, userID uuid.UUID, conversationID uuid.UUID, segmentID uuid.UUID) (types.ConversationSegment, error) {
	db := r.getQueries(ctx)
	row, err := db.GetConversationSegment(ctx, database.GetConversationSegmentParams{
		ConversationID: conversationID,
		ID:             segmentID,
		UserID:         &userID,
	})
	if err != nil {
		return types.ConversationSegment{}, err
	}

	var text string
	if row.Text.Valid {
		text = row.Text.String
	}
	return types.ConversationSegment{
		ID:          row.ID,
		Text:        text,
		StartTime:   row.StartTime.Float64,
		EndTime:     row.EndTime.Float64,
		Duration:    row.Duration.Float64,
		SpeakerID:   row.SpeakerID,
		SpeakerName: textPtr(row.SpeakerName),
		Ignored:     row.Ignored,
	}, nil
}

func (r *dbRepository) SetSegmentIgnored(ctx context.Context, conversationID uuid.UUID, segmentID uuid.UUID, ignored bool) error {
	db := r.getQueries(ctx)
	return db.SetSegmentIgnored(ctx, database.SetSegmentIgnoredParams{
		ID:             segmentID,
		Ignored:        ignored,
		ConversationID: conversationID,
	})
}
