package handler

import (
	"context"

	"github.com/ChrQR/meeting-assistant/server/internal/api"
)

func (h *Handlers) GetConversationList(ctx context.Context, req api.GetConversationWithSegmentsRequestObject) (api.GetConversationsListResponseObject, error) {
	data, err := h.StoreService.GetConversationList(ctx context.Context)
	if err != nil {
		return api.GetConversationsListResponseObject
	}
}
