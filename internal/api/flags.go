package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/getflaggy/flaggy/internal/models"
	"github.com/getflaggy/flaggy/internal/sse"
)

func (s *Server) CreateFlag(w http.ResponseWriter, r *http.Request) {
	var req models.CreateFlagRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	flag := &models.Flag{
		Key:          req.Key,
		Type:         req.Type,
		Description:  req.Description,
		Enabled:      req.Enabled,
		DefaultValue: req.DefaultValue,
	}

	if err := models.ValidateFlag(flag); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := s.store.CreateFlag(flag); err != nil {
		respondError(w, http.StatusConflict, "flag already exists or DB error: "+err.Error())
		return
	}

	s.broadcaster.Publish(sse.Event{
		ID: fmt.Sprintf("%d", time.Now().UnixMilli()), Type: "flag_created", Data: flag,
	})
	respondJSON(w, http.StatusCreated, flag)
}

func (s *Server) ListFlags(w http.ResponseWriter, r *http.Request) {
	flags, err := s.store.ListFlags()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if flags == nil {
		flags = []models.Flag{}
	}
	respondJSON(w, http.StatusOK, flags)
}

func (s *Server) GetFlag(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")
	flag, err := s.store.GetFlag(key)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if flag == nil {
		respondError(w, http.StatusNotFound, "flag not found")
		return
	}
	respondJSON(w, http.StatusOK, flag)
}

func (s *Server) UpdateFlag(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")

	var req models.UpdateFlagRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	flag, err := s.store.UpdateFlag(key, &req)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	if flag == nil {
		respondError(w, http.StatusNotFound, "flag not found")
		return
	}
	s.broadcaster.Publish(sse.Event{
		ID: fmt.Sprintf("%d", time.Now().UnixMilli()), Type: "flag_updated", Data: flag,
	})
	respondJSON(w, http.StatusOK, flag)
}

func (s *Server) DeleteFlag(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")
	if err := s.store.DeleteFlag(key); err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	s.broadcaster.Publish(sse.Event{
		ID: fmt.Sprintf("%d", time.Now().UnixMilli()), Type: "flag_deleted", Data: map[string]string{"key": key},
	})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) ToggleFlag(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")
	flag, err := s.store.ToggleFlag(key)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if flag == nil {
		respondError(w, http.StatusNotFound, "flag not found")
		return
	}
	s.broadcaster.Publish(sse.Event{
		ID: fmt.Sprintf("%d", time.Now().UnixMilli()), Type: "flag_toggled", Data: flag,
	})
	respondJSON(w, http.StatusOK, flag)
}
