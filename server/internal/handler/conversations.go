package handler

import (
	"net/http"

	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/ChrQR/meeting-assistant/server/internal/api"
)

func (s *Handlers) GetConversations(w http.ResponseWriter, r *http.Request) {
	userID, ok := s.requireUserID(w, r)
	if !ok {
		return
	}

	conversations, err := s.StoreService.GetConversations(r.Context(), userID)
	if err != nil {
		writeError(w, "GetConversations", err)
		return
	}

	result := make([]api.ConversationSummary, 0, len(conversations))
	for _, conversation := range conversations {
		result = append(result, toConversationSummaryResponse(conversation))
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Handlers) GetConversation(w http.ResponseWriter, r *http.Request, id openapi_types.UUID) {
	userID, ok := s.requireUserID(w, r)
	if !ok {
		return
	}

	conversation, err := s.StoreService.GetConversation(r.Context(), userID, id)
	if err != nil {
		writeError(w, "GetConversation", err)
		return
	}
	writeJSON(w, http.StatusOK, toConversationResponse(conversation))
}

func (s *Handlers) SetSegmentIgnored(w http.ResponseWriter, r *http.Request, id openapi_types.UUID, segmentID openapi_types.UUID) {
	userID, ok := s.requireUserID(w, r)
	if !ok {
		return
	}

	var body api.SetSegmentIgnoredBody
	if err := decodeJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	segment, err := s.StoreService.SetSegmentIgnored(r.Context(), userID, id, segmentID, body.Ignored)
	if err != nil {
		writeError(w, "SetSegmentIgnored", err)
		return
	}

	writeJSON(w, http.StatusOK, toConversationSegmentResponse(segment))
}
