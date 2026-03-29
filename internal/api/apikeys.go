package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/getflaggy/flaggy/internal/models"
)

type createAPIKeyRequest struct {
	Name        string            `json:"name"`
	Environment models.Environment `json:"environment"`
}

func (s *Server) CreateAPIKey(w http.ResponseWriter, r *http.Request) {
	var req createAPIKeyRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	if req.Name == "" {
		respondError(w, http.StatusBadRequest, "name is required")
		return
	}
	if err := models.ValidateEnvironment(req.Environment); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	keyWithRaw, hashedKey := models.GenerateAPIKey(req.Name, req.Environment)

	if err := s.store.CreateAPIKey(&keyWithRaw.APIKey, hashedKey); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Return the raw key — shown only this once
	respondJSON(w, http.StatusCreated, keyWithRaw)
}

func (s *Server) ListAPIKeys(w http.ResponseWriter, r *http.Request) {
	keys, err := s.store.ListAPIKeys()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if keys == nil {
		keys = []models.APIKey{}
	}
	respondJSON(w, http.StatusOK, keys)
}

func (s *Server) RevokeAPIKey(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.store.RevokeAPIKey(id); err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
