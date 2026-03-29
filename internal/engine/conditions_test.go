package engine

import (
	"encoding/json"
	"testing"

	"github.com/getflaggy/flaggy/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func j(v interface{}) json.RawMessage {
	return MustJSON(v)
}

func TestResolveAttribute(t *testing.T) {
	ctx := EvalContext{
		"plan": "pro",
		"user": map[string]interface{}{
			"name": "alice",
			"meta": map[string]interface{}{
				"age": 30,
			},
		},
	}

	tests := []struct {
		attr   string
		want   interface{}
		exists bool
	}{
		{"plan", "pro", true},
		{"user.name", "alice", true},
		{"user.meta.age", 30, true},
		{"missing", nil, false},
		{"user.missing", nil, false},
		{"user.name.deep", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.attr, func(t *testing.T) {
			got, exists := resolveAttribute(ctx, tt.attr)
			assert.Equal(t, tt.exists, exists)
			if tt.exists {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestOpEquals(t *testing.T) {
	tests := []struct {
		name    string
		attr    interface{}
		cond    json.RawMessage
		want    bool
		wantErr bool
	}{
		{"string match", "pro", j("pro"), true, false},
		{"string mismatch", "free", j("pro"), false, false},
		{"number match", float64(42), j(42), true, false},
		{"number mismatch", float64(10), j(42), false, false},
		{"int vs float", 42, j(42.0), true, false},
		{"bool match", true, j(true), true, false},
		{"bool mismatch", true, j(false), false, false},
		{"nil attr", nil, j("pro"), false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := opEquals(tt.attr, tt.cond)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestOpNotEquals(t *testing.T) {
	got, err := opNotEquals("free", j("pro"))
	require.NoError(t, err)
	assert.True(t, got)

	got, err = opNotEquals("pro", j("pro"))
	require.NoError(t, err)
	assert.False(t, got)
}

func TestOpIn(t *testing.T) {
	tests := []struct {
		name string
		attr interface{}
		cond json.RawMessage
		want bool
	}{
		{"found", "pro", j([]string{"free", "pro", "enterprise"}), true},
		{"not found", "basic", j([]string{"free", "pro"}), false},
		{"number in list", float64(1), j([]int{1, 2, 3}), true},
		{"empty list", "pro", j([]string{}), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := opIn(tt.attr, tt.cond)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestOpNotIn(t *testing.T) {
	got, err := opNotIn("basic", j([]string{"free", "pro"}))
	require.NoError(t, err)
	assert.True(t, got)

	got, err = opNotIn("pro", j([]string{"free", "pro"}))
	require.NoError(t, err)
	assert.False(t, got)
}

func TestOpContains(t *testing.T) {
	tests := []struct {
		name string
		attr interface{}
		cond json.RawMessage
		want bool
	}{
		{"contains substring", "hello world", j("world"), true},
		{"no match", "hello", j("world"), false},
		{"non-string attr", 123, j("test"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := opContains(tt.attr, tt.cond)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestOpStartsWith(t *testing.T) {
	tests := []struct {
		name string
		attr interface{}
		cond json.RawMessage
		want bool
	}{
		{"match", "hello world", j("hello"), true},
		{"no match", "hello", j("world"), false},
		{"non-string attr", []int{1}, j("4"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := opStartsWith(tt.attr, tt.cond)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNumericOperators(t *testing.T) {
	tests := []struct {
		name string
		op   ConditionFunc
		attr interface{}
		cond json.RawMessage
		want bool
	}{
		{"gt true", opGT, float64(10), j(5), true},
		{"gt false", opGT, float64(5), j(10), false},
		{"gt equal", opGT, float64(5), j(5), false},
		{"gte true", opGTE, float64(5), j(5), true},
		{"gte false", opGTE, float64(4), j(5), false},
		{"lt true", opLT, float64(3), j(5), true},
		{"lt false", opLT, float64(10), j(5), false},
		{"lte true", opLTE, float64(5), j(5), true},
		{"lte false", opLTE, float64(6), j(5), false},
		{"int attr", opGT, 10, j(5), true},
		{"int64 attr", opGT, int64(10), j(5), true},
		{"non-numeric attr", opGT, "abc", j(5), false},
		{"non-numeric cond", opGT, float64(10), j("abc"), false},
	}
	for _, tt := range tests {
		op := tt.op
		if op == nil {
			op = opGT
		}
		t.Run(tt.name, func(t *testing.T) {
			got, err := op(tt.attr, tt.cond)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestOpExists(t *testing.T) {
	tests := []struct {
		name string
		attr interface{}
		cond json.RawMessage
		want bool
	}{
		{"exists and expected true", "value", j(true), true},
		{"exists but expected false", "value", j(false), false},
		{"nil and expected false", nil, j(false), true},
		{"nil and expected true", nil, j(true), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := opExists(tt.attr, tt.cond)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}

	// Error case: non-bool value
	_, err := opExists("val", j("not-bool"))
	assert.Error(t, err)
}

func TestOpRegex(t *testing.T) {
	tests := []struct {
		name    string
		attr    interface{}
		cond    json.RawMessage
		want    bool
		wantErr bool
	}{
		{"match", "user@example.com", j(`^[^@]+@[^@]+\.[^@]+$`), true, false},
		{"no match", "invalid", j(`^[^@]+@[^@]+\.[^@]+$`), false, false},
		{"invalid regex", "test", j(`[invalid`), false, true},
		{"non-string attr", []int{1}, j(".*"), false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := opRegex(tt.attr, tt.cond)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestEvalCondition_MissingAttribute(t *testing.T) {
	ctx := EvalContext{"plan": "pro"}
	cond := &models.Condition{
		Attribute: "missing",
		Operator:  models.OpEquals,
		Value:     j("pro"),
	}
	got, err := EvalCondition(cond, ctx)
	require.NoError(t, err)
	assert.False(t, got)
}

func TestEvalCondition_UnknownOperator(t *testing.T) {
	ctx := EvalContext{"plan": "pro"}
	cond := &models.Condition{
		Attribute: "plan",
		Operator:  "unknown_op",
		Value:     j("pro"),
	}
	_, err := EvalCondition(cond, ctx)
	assert.Error(t, err)
}

func TestToFloat64(t *testing.T) {
	tests := []struct {
		name string
		val  interface{}
		want float64
		ok   bool
	}{
		{"float64", float64(1.5), 1.5, true},
		{"float32", float32(1.5), float64(float32(1.5)), true},
		{"int", 42, 42.0, true},
		{"int64", int64(42), 42.0, true},
		{"string", "nope", 0, false},
		{"bool", true, 0, false},
		{"nil", nil, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := toFloat64(tt.val)
			assert.Equal(t, tt.ok, ok)
			if ok {
				assert.InDelta(t, tt.want, got, 0.001)
			}
		})
	}
}

func TestToString(t *testing.T) {
	tests := []struct {
		name string
		val  interface{}
		want string
		ok   bool
	}{
		{"string", "hello", "hello", true},
		{"float64", float64(3.14), "3.14", true},
		{"int", 42, "42", true},
		{"int64", int64(100), "100", true},
		{"bool", true, "true", true},
		{"nil", nil, "", false},
		{"float32", float32(2.5), "2.5", true},
		{"slice", []int{1}, "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := toString(tt.val)
			assert.Equal(t, tt.ok, ok)
			if ok {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestOpEquals_InvalidJSON(t *testing.T) {
	_, err := opEquals("val", json.RawMessage(`{invalid`))
	assert.Error(t, err)
}

func TestOpIn_InvalidCondValue(t *testing.T) {
	_, err := opIn("val", json.RawMessage(`"not-array"`))
	assert.Error(t, err)
}

func TestOpContains_InvalidCondJSON(t *testing.T) {
	_, err := opContains("hello", json.RawMessage(`{bad`))
	assert.Error(t, err)
}

func TestOpContains_NonStringCondValue(t *testing.T) {
	got, err := opContains("hello", j([]int{1}))
	assert.NoError(t, err)
	assert.False(t, got)
}

func TestOpStartsWith_InvalidCondJSON(t *testing.T) {
	_, err := opStartsWith("hello", json.RawMessage(`{bad`))
	assert.Error(t, err)
}

func TestOpStartsWith_NonStringCondValue(t *testing.T) {
	got, err := opStartsWith("hello", j([]int{1}))
	assert.NoError(t, err)
	assert.False(t, got)
}

func TestNumericCompare_InvalidJSON(t *testing.T) {
	_, _, _, err := numericCompare(float64(1), json.RawMessage(`{bad`))
	assert.Error(t, err)
}

func TestOpExists_InvalidCondJSON(t *testing.T) {
	_, err := opExists("val", json.RawMessage(`{bad`))
	assert.Error(t, err)
}

func TestOpRegex_InvalidCondJSON(t *testing.T) {
	_, err := opRegex("val", json.RawMessage(`{bad`))
	assert.Error(t, err)
}

func TestOpRegex_NonStringPattern(t *testing.T) {
	_, err := opRegex("val", j(42))
	assert.Error(t, err)
}

func TestEvalCondition_ExistsWithMissingAttr(t *testing.T) {
	ctx := EvalContext{"other": "val"}
	cond := &models.Condition{
		Attribute: "missing",
		Operator:  models.OpExists,
		Value:     j(false),
	}
	got, err := EvalCondition(cond, ctx)
	require.NoError(t, err)
	assert.True(t, got) // missing + expected false = true
}
