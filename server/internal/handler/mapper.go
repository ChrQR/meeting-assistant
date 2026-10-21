package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"io"

	"github.com/oapi-codegen/nullable"

	"github.com/ChrQR/meeting-assistant/server/internal/api"
	"github.com/ChrQR/meeting-assistant/server/internal/service"
	"github.com/ChrQR/meeting-assistant/server/internal/types"
)

// writeJSON writes a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if body == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(body); err != nil {
		log.Printf("failed to encode response: %v", err)
	}
}

// writeError maps a domain/service error to an HTTP status and writes it.
func writeError(w http.ResponseWriter, operation string, err error) {
	status := httpStatus(err)
	if status == http.StatusInternalServerError {
		log.Printf("%s failed: %v", operation, err)
	} else {
		log.Printf("%s rejected: %v", operation, err)
	}
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

// decodeJSON reads and decodes a JSON request body into destination.
func decodeJSON(r *http.Request, destination any) error {
	defer r.Body.Close()
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		return err
	}
	if len(body) == 0 {
		return errors.New("request body is required")
	}
	return json.Unmarshal(body, destination)
}

// nullableFromPtr converts an optional value into the nullable wrapper used by
// the generated API models. A nil pointer is represented as an explicit JSON null.
func nullableFromPtr[T any](value *T) nullable.Nullable[T] {
	if value == nil {
		return nullable.NewNullNullable[T]()
	}
	return nullable.NewNullableWithValue(*value)
}

// valueOrNil unwraps an optional nullable value. Unspecified or explicit null
// values are represented as a nil pointer.
func valueOrNil[T any](value nullable.Nullable[T]) *T {
	if !value.IsSpecified() || value.IsNull() {
		return nil
	}
	resolved, err := value.Get()
	if err != nil {
		return nil
	}
	return &resolved
}

func toSpeakerResponse(speaker types.Speaker) api.Speaker {
	return api.Speaker{
		Id:                speaker.ID,
		Name:              nullableFromPtr(speaker.Name),
		Email:             nullableFromPtr(speaker.Email),
		CompanyId:         nullableFromPtr(speaker.CompanyID),
		CompanyName:       nullableFromPtr(speaker.CompanyName),
		UserId:            nullableFromPtr(speaker.UserID),
		UserEmail:         nullableFromPtr(speaker.UserEmail),
		EmbeddingCount:    speaker.EmbeddingCount,
		SegmentCount:      speaker.SegmentCount,
		TotalSpeakingTime: speaker.TotalSpeakingTime,
		ConversationCount: speaker.ConversationCount,
		CreatedAt:         speaker.CreatedAt,
	}
}

func toSpeakerDetailResponse(speaker types.SpeakerDetail) api.SpeakerDetail {
	base := toSpeakerResponse(speaker.Speaker)
	response := api.SpeakerDetail{
		CompanyId:         base.CompanyId,
		CompanyName:       base.CompanyName,
		UserId:            base.UserId,
		UserEmail:         base.UserEmail,
		ConversationCount: base.ConversationCount,
		CreatedAt:         base.CreatedAt,
		Email:             base.Email,
		EmbeddingCount:    base.EmbeddingCount,
		Id:                base.Id,
		Name:              base.Name,
		SegmentCount:      base.SegmentCount,
		TotalSpeakingTime: base.TotalSpeakingTime,
	}

	embeddings := make([]api.SpeakerEmbedding, 0, len(speaker.Embeddings))
	for _, embedding := range speaker.Embeddings {
		embeddings = append(embeddings, api.SpeakerEmbedding{
			Id:        embedding.ID,
			CreatedAt: embedding.CreatedAt,
		})
	}
	response.Embeddings = embeddings

	segments := make([]api.SpeakerSegment, 0, len(speaker.Segments))
	for _, segment := range speaker.Segments {
		segments = append(segments, api.SpeakerSegment{
			Id:                segment.ID,
			Text:              segment.Text,
			StartTime:         segment.StartTime,
			EndTime:           segment.EndTime,
			ConversationId:    segment.ConversationID,
			ConversationTitle: segment.ConversationTitle,
		})
	}
	response.Segments = segments

	return response
}

func toConversationSummaryResponse(conversation types.ConversationSummary) api.ConversationSummary {
	return api.ConversationSummary{
		Id:                  conversation.ID,
		Title:               conversation.Title,
		Summary:             nullableFromPtr(conversation.Summary),
		Length:              conversation.Length,
		Cost:                conversation.Cost,
		Language:            conversation.Language,
		CreatedAt:           conversation.CreatedAt,
		MeetingAt:           nullableFromPtr(conversation.MeetingAt),
		RecordingId:         conversation.RecordingID,
		RecordingStatus:     conversation.RecordingStatus,
		SpeakerCount:        conversation.SpeakerCount,
		UnknownSpeakerCount: conversation.UnknownSpeakerCount,
	}
}

func toConversationResponse(conversation types.Conversation) api.Conversation {
	base := toConversationSummaryResponse(conversation.ConversationSummary)
	response := api.Conversation{
		Cost:                base.Cost,
		CreatedAt:           base.CreatedAt,
		MeetingAt:           base.MeetingAt,
		Id:                  base.Id,
		Language:            base.Language,
		Length:              base.Length,
		RecordingId:         base.RecordingId,
		RecordingStatus:     base.RecordingStatus,
		SpeakerCount:        base.SpeakerCount,
		Summary:             base.Summary,
		Title:               base.Title,
		UnknownSpeakerCount: base.UnknownSpeakerCount,
	}

	segments := make([]api.ConversationSegment, 0, len(conversation.Segments))
	for _, segment := range conversation.Segments {
		segments = append(segments, toConversationSegmentResponse(segment))
	}
	response.Segments = segments

	return response
}

func toConversationSegmentResponse(segment types.ConversationSegment) api.ConversationSegment {
	return api.ConversationSegment{
		Id:          segment.ID,
		Text:        segment.Text,
		StartTime:   segment.StartTime,
		EndTime:     segment.EndTime,
		Duration:    segment.Duration,
		SpeakerId:   segment.SpeakerID,
		SpeakerName: nullableFromPtr(segment.SpeakerName),
		Ignored:     segment.Ignored,
	}
}

func toCompanyResponse(company types.Company) api.Company {
	return api.Company{
		Id:           company.ID,
		Name:         company.Name,
		SpeakerCount: company.SpeakerCount,
		CreatedAt:    company.CreatedAt,
	}
}

// isNotFound reports whether the given service error represents a missing entity.
func isNotFound(err error) bool {
	return errors.Is(err, service.ErrNotFound)
}

// httpStatus maps a domain error to an HTTP status code.
func httpStatus(err error) int {
	switch {
	case errors.Is(err, service.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, service.ErrConflict):
		return http.StatusConflict
	case errors.Is(err, service.ErrInvalidInput):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}
