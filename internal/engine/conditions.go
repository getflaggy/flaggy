package engine

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/getflaggy/flaggy/internal/models"
)

// EvalContext is the user-provided context for evaluation.
type EvalContext map[string]interface{}

// ConditionFunc evaluates a condition against a context attribute value.
// attrVal may be nil if the attribute doesn't exist.
type ConditionFunc func(attrVal interface{}, condVal json.RawMessage) (bool, error)

var conditionFuncs = map[models.Operator]ConditionFunc{
	models.OpEquals:     opEquals,
	models.OpNotEquals:  opNotEquals,
	models.OpIn:         opIn,
	models.OpNotIn:      opNotIn,
	models.OpContains:   opContains,
	models.OpStartsWith: opStartsWith,
	models.OpGT:         opGT,
	models.OpGTE:        opGTE,
	models.OpLT:         opLT,
	models.OpLTE:        opLTE,
	models.OpExists:     opExists,
	models.OpRegex:      opRegex,
}

// resolveAttribute walks a nested map using dot-separated keys.
// e.g. "user.plan" on {"user": {"plan": "pro"}} returns "pro".
func resolveAttribute(ctx EvalContext, attr string) (interface{}, bool) {
	parts := strings.Split(attr, ".")
	var current interface{} = map[string]interface{}(ctx)

	for _, part := range parts {
		m, ok := current.(map[string]interface{})
		if !ok {
			return nil, false
		}
		current, ok = m[part]
		if !ok {
			return nil, false
		}
	}
	return current, true
}

// EvalCondition evaluates a single condition against the context.
func EvalCondition(cond *models.Condition, ctx EvalContext) (bool, error) {
	fn, ok := conditionFuncs[cond.Operator]
	if !ok {
		return false, fmt.Errorf("unknown operator: %q", cond.Operator)
	}

	attrVal, exists := resolveAttribute(ctx, cond.Attribute)
	if !exists && cond.Operator != models.OpExists {
		return false, nil
	}
	if !exists {
		attrVal = nil
	}

	return fn(attrVal, cond.Value)
}

func unmarshalCondVal(raw json.RawMessage) (interface{}, error) {
	var v interface{}
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, err
	}
	return v, nil
}

func toString(v interface{}) (string, bool) {
	switch val := v.(type) {
	case string:
		return val, true
	case float64:
		return fmt.Sprintf("%g", val), true
	case float32:
		return fmt.Sprintf("%g", val), true
	case int:
		return fmt.Sprintf("%d", val), true
	case int64:
		return fmt.Sprintf("%d", val), true
	case bool:
		return fmt.Sprintf("%t", val), true
	default:
		return "", false
	}
}

func toFloat64(v interface{}) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	case json.Number:
		f, err := val.Float64()
		return f, err == nil
	default:
		return 0, false
	}
}

// --- Operator implementations ---

func opEquals(attrVal interface{}, condVal json.RawMessage) (bool, error) {
	cv, err := unmarshalCondVal(condVal)
	if err != nil {
		return false, err
	}
	return compareValues(attrVal, cv), nil
}

func opNotEquals(attrVal interface{}, condVal json.RawMessage) (bool, error) {
	result, err := opEquals(attrVal, condVal)
	return !result, err
}

func opIn(attrVal interface{}, condVal json.RawMessage) (bool, error) {
	var list []interface{}
	if err := json.Unmarshal(condVal, &list); err != nil {
		return false, fmt.Errorf("in operator requires array value: %w", err)
	}
	for _, item := range list {
		if compareValues(attrVal, item) {
			return true, nil
		}
	}
	return false, nil
}

func opNotIn(attrVal interface{}, condVal json.RawMessage) (bool, error) {
	result, err := opIn(attrVal, condVal)
	return !result, err
}

func opContains(attrVal interface{}, condVal json.RawMessage) (bool, error) {
	attrStr, ok := toString(attrVal)
	if !ok {
		return false, nil
	}
	cv, err := unmarshalCondVal(condVal)
	if err != nil {
		return false, err
	}
	cvStr, ok := toString(cv)
	if !ok {
		return false, nil
	}
	return strings.Contains(attrStr, cvStr), nil
}

func opStartsWith(attrVal interface{}, condVal json.RawMessage) (bool, error) {
	attrStr, ok := toString(attrVal)
	if !ok {
		return false, nil
	}
	cv, err := unmarshalCondVal(condVal)
	if err != nil {
		return false, err
	}
	cvStr, ok := toString(cv)
	if !ok {
		return false, nil
	}
	return strings.HasPrefix(attrStr, cvStr), nil
}

func numericCompare(attrVal interface{}, condVal json.RawMessage) (float64, float64, bool, error) {
	av, ok := toFloat64(attrVal)
	if !ok {
		return 0, 0, false, nil
	}
	cv, err := unmarshalCondVal(condVal)
	if err != nil {
		return 0, 0, false, err
	}
	cvf, ok := toFloat64(cv)
	if !ok {
		return 0, 0, false, nil
	}
	return av, cvf, true, nil
}

func opGT(attrVal interface{}, condVal json.RawMessage) (bool, error) {
	av, cv, ok, err := numericCompare(attrVal, condVal)
	if err != nil || !ok {
		return false, err
	}
	return av > cv, nil
}

func opGTE(attrVal interface{}, condVal json.RawMessage) (bool, error) {
	av, cv, ok, err := numericCompare(attrVal, condVal)
	if err != nil || !ok {
		return false, err
	}
	return av >= cv, nil
}

func opLT(attrVal interface{}, condVal json.RawMessage) (bool, error) {
	av, cv, ok, err := numericCompare(attrVal, condVal)
	if err != nil || !ok {
		return false, err
	}
	return av < cv, nil
}

func opLTE(attrVal interface{}, condVal json.RawMessage) (bool, error) {
	av, cv, ok, err := numericCompare(attrVal, condVal)
	if err != nil || !ok {
		return false, err
	}
	return av <= cv, nil
}

func opExists(attrVal interface{}, condVal json.RawMessage) (bool, error) {
	cv, err := unmarshalCondVal(condVal)
	if err != nil {
		return false, err
	}
	expected, ok := cv.(bool)
	if !ok {
		return false, fmt.Errorf("exists operator requires boolean value")
	}
	exists := attrVal != nil
	return exists == expected, nil
}

func opRegex(attrVal interface{}, condVal json.RawMessage) (bool, error) {
	attrStr, ok := toString(attrVal)
	if !ok {
		return false, nil
	}
	cv, err := unmarshalCondVal(condVal)
	if err != nil {
		return false, err
	}
	pattern, ok := cv.(string)
	if !ok {
		return false, fmt.Errorf("regex operator requires string pattern")
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return false, fmt.Errorf("invalid regex: %w", err)
	}
	return re.MatchString(attrStr), nil
}

// compareValues does a type-aware equality check.
func compareValues(a, b interface{}) bool {
	// Try numeric comparison first
	af, aOk := toFloat64(a)
	bf, bOk := toFloat64(b)
	if aOk && bOk {
		return af == bf
	}

	// String comparison
	as, aOk := toString(a)
	bs, bOk := toString(b)
	if aOk && bOk {
		return as == bs
	}

	// Bool comparison
	ab, aOk := a.(bool)
	bb, bOk := b.(bool)
	if aOk && bOk {
		return ab == bb
	}

	return false
}
