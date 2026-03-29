package engine

import (
	"encoding/json"
	"sort"

	"github.com/getflaggy/flaggy/internal/models"
)

const (
	ReasonDisabled  = "disabled"
	ReasonDefault   = "default"
	ReasonRuleMatch = "rule_match"
	ReasonRollout   = "rollout"
	ReasonError     = "error"
)

// Evaluate evaluates a flag against the given context.
// Algorithm:
//  1. If flag is disabled → return default value
//  2. Sort rules by priority (ascending = highest priority first)
//  3. For each rule, check all conditions. First rule where ALL conditions match → return rule's value
//  4. No rule matched → return default value
func Evaluate(flag *models.Flag, ctx EvalContext) models.EvaluateResponse {
	resp := models.EvaluateResponse{
		FlagKey: flag.Key,
	}

	if !flag.Enabled {
		if flag.Type == "boolean" {
			resp.Value = json.RawMessage("false")
		} else {
			resp.Value = flag.DefaultValue
		}
		resp.Reason = ReasonDisabled
		return resp
	}

	if len(flag.Rules) == 0 {
		if flag.Type == "boolean" {
			resp.Value = json.RawMessage("true")
		} else {
			resp.Value = flag.DefaultValue
		}
		resp.Reason = ReasonDefault
		return resp
	}

	// Sort rules by priority (lower number = higher priority)
	rules := make([]models.Rule, len(flag.Rules))
	copy(rules, flag.Rules)
	sort.Slice(rules, func(i, j int) bool {
		return rules[i].Priority < rules[j].Priority
	})

	// Resolve entity ID for rollout bucketing
	entityID, _ := resolveEntityID(ctx)

	for _, rule := range rules {
		matched, err := evalRule(&rule, ctx, flag.Segments)
		if err != nil {
			resp.Value = flag.DefaultValue
			resp.Reason = ReasonError
			return resp
		}
		if matched {
			// Check rollout percentage if set
			if rule.RolloutPercentage > 0 && rule.RolloutPercentage < 100 {
				if entityID == "" || !InRollout(flag.Key, entityID, rule.RolloutPercentage) {
					continue // Not in rollout, try next rule
				}
			}
			resp.Value = rule.Value
			resp.Match = true
			resp.Reason = ReasonRuleMatch
			return resp
		}
	}

	resp.Value = flag.DefaultValue
	resp.Reason = ReasonDefault
	return resp
}

// evalRule returns true if ALL inline conditions AND all segment conditions match.
// Fail closed: if a referenced segment is not found in the map, the rule does not match.
func evalRule(rule *models.Rule, ctx EvalContext, segments map[string]*models.Segment) (bool, error) {
	// Evaluate inline conditions
	for i := range rule.Conditions {
		ok, err := EvalCondition(&rule.Conditions[i], ctx)
		if err != nil {
			return false, err
		}
		if !ok {
			return false, nil
		}
	}

	// Evaluate segment conditions (AND with inline conditions)
	for _, segKey := range rule.SegmentKeys {
		seg, ok := segments[segKey]
		if !ok || seg == nil {
			// Fail closed: missing segment → rule does not match
			return false, nil
		}
		for i := range seg.Conditions {
			ok, err := EvalCondition(&seg.Conditions[i], ctx)
			if err != nil {
				return false, err
			}
			if !ok {
				return false, nil
			}
		}
	}

	return true, nil
}

// resolveEntityID extracts the entity identifier from the context.
// Looks for "entity_id", "user_id", or "user.id" in that order.
func resolveEntityID(ctx EvalContext) (string, bool) {
	for _, key := range []string{"entity_id", "user_id"} {
		if v, ok := ctx[key]; ok {
			if s, ok := toString(v); ok {
				return s, true
			}
		}
	}
	// Try nested user.id
	if v, ok := resolveAttribute(ctx, "user.id"); ok {
		if s, ok := toString(v); ok {
			return s, true
		}
	}
	return "", false
}

// MustJSON marshals v to json.RawMessage, panicking on error. Test helper.
func MustJSON(v interface{}) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}
