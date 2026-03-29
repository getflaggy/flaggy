package api

import (
	"net/http"

	"github.com/getflaggy/flaggy/internal/engine"
	"github.com/getflaggy/flaggy/internal/models"
)

func (s *Server) EvaluateBatch(w http.ResponseWriter, r *http.Request) {
	var req models.BatchEvaluateRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	if len(req.Flags) == 0 {
		respondError(w, http.StatusBadRequest, "flags list is required")
		return
	}

	ctx := engine.EvalContext(req.Context)
	results := make([]models.EvaluateResponse, 0, len(req.Flags))

	for _, flagKey := range req.Flags {
		flag, err := s.store.GetFlagForEvaluation(flagKey)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if flag == nil {
			results = append(results, models.EvaluateResponse{
				FlagKey: flagKey,
				Reason:  "not_found",
			})
			continue
		}
		results = append(results, engine.Evaluate(flag, ctx))
	}

	respondJSON(w, http.StatusOK, models.BatchEvaluateResponse{Results: results})
}
