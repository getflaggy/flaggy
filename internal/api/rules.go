package api

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/getflaggy/flaggy/internal/models"
	"github.com/getflaggy/flaggy/internal/sse"
)

func (s *Server) CreateRule(w http.ResponseWriter, r *http.Request) {
	flagKey := chi.URLParam(r, "key")

	// Verify flag exists
	flag, err := s.store.GetFlag(flagKey)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if flag == nil {
		respondError(w, http.StatusNotFound, "flag not found")
		return
	}

	var req models.CreateRuleRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	rule := &models.Rule{
		Description:       req.Description,
		Value:             req.Value,
		Priority:          req.Priority,
		RolloutPercentage: req.RolloutPercentage,
		Conditions:        req.Conditions,
		SegmentKeys:       req.SegmentKeys,
	}

	if err := models.ValidateRule(rule); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := models.ValidateValueForType(flag.Type, req.Value); err != nil {
		respondError(w, http.StatusBadRequest, "value: "+err.Error())
		return
	}

	if err := s.store.CreateRule(flagKey, rule); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.broadcaster.Publish(sse.Event{
		ID: fmt.Sprintf("%d", time.Now().UnixMilli()), Type: "rule_created", Data: rule,
	})
	respondJSON(w, http.StatusCreated, rule)
}

func (s *Server) UpdateRule(w http.ResponseWriter, r *http.Request) {
	flagKey := chi.URLParam(r, "key")
	ruleIDStr := chi.URLParam(r, "ruleID")
	ruleID, err := strconv.ParseInt(ruleIDStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid rule ID")
		return
	}

	var req models.CreateRuleRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	rule := &models.Rule{
		Description:       req.Description,
		Value:             req.Value,
		Priority:          req.Priority,
		RolloutPercentage: req.RolloutPercentage,
		Conditions:        req.Conditions,
		SegmentKeys:       req.SegmentKeys,
	}
	if err := models.ValidateRule(rule); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	updated, err := s.store.UpdateRule(flagKey, ruleID, &req)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	s.broadcaster.Publish(sse.Event{
		ID: fmt.Sprintf("%d", time.Now().UnixMilli()), Type: "rule_updated", Data: updated,
	})
	respondJSON(w, http.StatusOK, updated)
}

func (s *Server) DeleteRule(w http.ResponseWriter, r *http.Request) {
	flagKey := chi.URLParam(r, "key")
	ruleIDStr := chi.URLParam(r, "ruleID")
	ruleID, err := strconv.ParseInt(ruleIDStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid rule ID")
		return
	}

	if err := s.store.DeleteRule(flagKey, ruleID); err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	s.broadcaster.Publish(sse.Event{
		ID:   fmt.Sprintf("%d", time.Now().UnixMilli()),
		Type: "rule_deleted",
		Data: map[string]interface{}{"flag_key": flagKey, "rule_id": ruleID},
	})
	w.WriteHeader(http.StatusNoContent)
}
