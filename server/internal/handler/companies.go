package handler

import (
	"net/http"

	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/ChrQR/meeting-assistant/server/internal/api"
)

func (s *Handlers) GetCompanies(w http.ResponseWriter, r *http.Request) {
	userID, ok := s.requireUserID(w, r)
	if !ok {
		return
	}

	companies, err := s.StoreService.GetCompanies(r.Context(), userID)
	if err != nil {
		writeError(w, "GetCompanies", err)
		return
	}

	result := make([]api.Company, 0, len(companies))
	for _, company := range companies {
		result = append(result, toCompanyResponse(company))
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Handlers) CreateCompany(w http.ResponseWriter, r *http.Request) {
	userID, ok := s.requireUserID(w, r)
	if !ok {
		return
	}

	var body api.CompanyBody
	if err := decodeJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	company, err := s.StoreService.CreateCompany(r.Context(), userID, body.Name)
	if err != nil {
		writeError(w, "CreateCompany", err)
		return
	}
	writeJSON(w, http.StatusCreated, toCompanyResponse(company))
}

func (s *Handlers) UpdateCompany(w http.ResponseWriter, r *http.Request, id openapi_types.UUID) {
	userID, ok := s.requireUserID(w, r)
	if !ok {
		return
	}

	var body api.CompanyBody
	if err := decodeJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	company, err := s.StoreService.UpdateCompany(r.Context(), userID, id, body.Name)
	if err != nil {
		writeError(w, "UpdateCompany", err)
		return
	}
	writeJSON(w, http.StatusOK, toCompanyResponse(company))
}

func (s *Handlers) DeleteCompany(w http.ResponseWriter, r *http.Request, id openapi_types.UUID) {
	userID, ok := s.requireUserID(w, r)
	if !ok {
		return
	}

	if err := s.StoreService.DeleteCompany(r.Context(), userID, id); err != nil {
		writeError(w, "DeleteCompany", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
