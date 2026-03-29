package api

import (
	"net/http"

	"github.com/getflaggy/flaggy/internal/engine"
	"github.com/getflaggy/flaggy/internal/models"
)

func (s *Server) Evaluate(w http.ResponseWriter, r *http.Request) {
	var req models.EvaluateRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	if req.FlagKey == "" {
		respondError(w, http.StatusBadRequest, "flag_key is required")
		return
	}

	flag, err := s.store.GetFlagForEvaluation(req.FlagKey)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if flag == nil {
		respondError(w, http.StatusNotFound, "flag not found")
		return
	}

	ctx := engine.EvalContext(req.Context)
	resp := engine.Evaluate(flag, ctx)

	respondJSON(w, http.StatusOK, resp)
}
