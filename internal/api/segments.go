package api

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/getflaggy/flaggy/internal/models"
	"github.com/getflaggy/flaggy/internal/sse"
	"github.com/getflaggy/flaggy/internal/store"
)

func (s *Server) CreateSegment(w http.ResponseWriter, r *http.Request) {
	var req models.CreateSegmentRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	segment := &models.Segment{
		Key:         req.Key,
		Description: req.Description,
		Conditions:  req.Conditions,
	}

	if err := models.ValidateSegment(segment); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := s.store.CreateSegment(segment); err != nil {
		respondError(w, http.StatusConflict, "segment already exists or DB error: "+err.Error())
		return
	}

	s.broadcaster.Publish(sse.Event{
		ID: fmt.Sprintf("%d", time.Now().UnixMilli()), Type: "segment_created", Data: segment,
	})
	respondJSON(w, http.StatusCreated, segment)
}

func (s *Server) ListSegments(w http.ResponseWriter, r *http.Request) {
	segments, err := s.store.ListSegments()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if segments == nil {
		segments = []models.Segment{}
	}
	respondJSON(w, http.StatusOK, segments)
}

func (s *Server) GetSegment(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")
	segment, err := s.store.GetSegment(key)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if segment == nil {
		respondError(w, http.StatusNotFound, "segment not found")
		return
	}
	respondJSON(w, http.StatusOK, segment)
}

func (s *Server) UpdateSegment(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")

	var req models.UpdateSegmentRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	// Validate new conditions if provided
	if req.Conditions != nil {
		if len(req.Conditions) == 0 {
			respondError(w, http.StatusBadRequest, "segment must have at least one condition")
			return
		}
		for i, c := range req.Conditions {
			if err := models.ValidateCondition(&c); err != nil {
				respondError(w, http.StatusBadRequest, fmt.Sprintf("condition[%d]: %s", i, err.Error()))
				return
			}
		}
	}

	segment, err := s.store.UpdateSegment(key, &req)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	if segment == nil {
		respondError(w, http.StatusNotFound, "segment not found")
		return
	}
	s.broadcaster.Publish(sse.Event{
		ID: fmt.Sprintf("%d", time.Now().UnixMilli()), Type: "segment_updated", Data: segment,
	})
	respondJSON(w, http.StatusOK, segment)
}

func (s *Server) DeleteSegment(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")
	if err := s.store.DeleteSegment(key); err != nil {
		if errors.Is(err, store.ErrSegmentInUse) {
			respondError(w, http.StatusConflict, err.Error())
			return
		}
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	s.broadcaster.Publish(sse.Event{
		ID:   fmt.Sprintf("%d", time.Now().UnixMilli()),
		Type: "segment_deleted",
		Data: map[string]string{"key": key},
	})
	w.WriteHeader(http.StatusNoContent)
}
