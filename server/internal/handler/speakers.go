package handler

import (
	"net/http"

	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/ChrQR/meeting-assistant/server/internal/api"
	"github.com/ChrQR/meeting-assistant/server/internal/types"
)

func (s *Handlers) GetSpeakers(w http.ResponseWriter, r *http.Request) {
	userID, ok := s.requireUserID(w, r)
	if !ok {
		return
	}

	speakers, err := s.StoreService.GetSpeakers(r.Context(), userID)
	if err != nil {
		writeError(w, "GetSpeakers", err)
		return
	}

	result := make([]api.Speaker, 0, len(speakers))
	for _, speaker := range speakers {
		result = append(result, toSpeakerResponse(speaker))
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Handlers) GetSpeaker(w http.ResponseWriter, r *http.Request, id openapi_types.UUID) {
	userID, ok := s.requireUserID(w, r)
	if !ok {
		return
	}

	speaker, err := s.StoreService.GetSpeaker(r.Context(), userID, id)
	if err != nil {
		writeError(w, "GetSpeaker", err)
		return
	}
	writeJSON(w, http.StatusOK, toSpeakerDetailResponse(speaker))
}

func (s *Handlers) CreateSpeaker(w http.ResponseWriter, r *http.Request) {
	userID, ok := s.requireUserID(w, r)
	if !ok {
		return
	}

	var body api.CreateSpeakerBody
	if err := decodeJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	speaker, err := s.StoreService.CreateSpeaker(r.Context(), userID, types.CreateSpeaker{
		Name:      body.Name,
		Email:     valueOrNil(body.Email),
		CompanyID: valueOrNil(body.CompanyId),
	})
	if err != nil {
		writeError(w, "CreateSpeaker", err)
		return
	}
	writeJSON(w, http.StatusCreated, toSpeakerResponse(speaker))
}

func (s *Handlers) UpdateSpeaker(w http.ResponseWriter, r *http.Request, id openapi_types.UUID) {
	userID, ok := s.requireUserID(w, r)
	if !ok {
		return
	}

	var body api.UpdateSpeakerBody
	if err := decodeJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	speaker, err := s.StoreService.UpdateSpeaker(r.Context(), userID, id, types.UpdateSpeaker{
		Name:         valueOrNil(body.Name),
		Email:        valueOrNil(body.Email),
		CompanyID:    valueOrNil(body.CompanyId),
		ClearCompany: body.ClearCompany,
		UserID:       valueOrNil(body.UserId),
		ClearUser:    body.ClearUser,
	})
	if err != nil {
		writeError(w, "UpdateSpeaker", err)
		return
	}
	writeJSON(w, http.StatusOK, toSpeakerResponse(speaker))
}

func (s *Handlers) DeleteSpeaker(w http.ResponseWriter, r *http.Request, id openapi_types.UUID) {
	userID, ok := s.requireUserID(w, r)
	if !ok {
		return
	}

	if err := s.StoreService.DeleteSpeaker(r.Context(), userID, id); err != nil {
		writeError(w, "DeleteSpeaker", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Handlers) AssignSpeaker(w http.ResponseWriter, r *http.Request, id openapi_types.UUID) {
	userID, ok := s.requireUserID(w, r)
	if !ok {
		return
	}

	var body api.AssignSpeakerBody
	if err := decodeJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	speaker, err := s.StoreService.AssignSpeaker(r.Context(), userID, id, types.AssignSpeaker{
		TargetSpeakerID: valueOrNil(body.TargetSpeakerId),
		Name:            valueOrNil(body.Name),
		Email:           valueOrNil(body.Email),
		CompanyID:       valueOrNil(body.CompanyId),
	})
	if err != nil {
		writeError(w, "AssignSpeaker", err)
		return
	}
	writeJSON(w, http.StatusOK, toSpeakerResponse(speaker))
}
