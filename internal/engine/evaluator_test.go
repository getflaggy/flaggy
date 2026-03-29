package engine

import (
	"encoding/json"
	"testing"

	"github.com/getflaggy/flaggy/internal/models"
	"github.com/stretchr/testify/assert"
)

func makeFlag(enabled bool, flagType models.FlagType, defaultVal interface{}, rules ...models.Rule) *models.Flag {
	return &models.Flag{
		Key:          "test_flag",
		Type:         flagType,
		Enabled:      enabled,
		DefaultValue: MustJSON(defaultVal),
		Rules:        rules,
	}
}

func makeRule(priority int, value interface{}, conditions ...models.Condition) models.Rule {
	return models.Rule{
		Priority:   priority,
		Value:      MustJSON(value),
		Conditions: conditions,
	}
}

func makeCond(attr string, op models.Operator, val interface{}) models.Condition {
	return models.Condition{
		Attribute: attr,
		Operator:  op,
		Value:     MustJSON(val),
	}
}

func TestEvaluate_FlagDisabled(t *testing.T) {
	flag := makeFlag(false, models.FlagTypeBoolean, false,
		makeRule(1, true, makeCond("plan", models.OpEquals, "pro")),
	)
	ctx := EvalContext{"plan": "pro"}

	resp := Evaluate(flag, ctx)
	assert.Equal(t, "test_flag", resp.FlagKey)
	assert.Equal(t, MustJSON(false), resp.Value)
	assert.False(t, resp.Match)
	assert.Equal(t, ReasonDisabled, resp.Reason)
}

func TestEvaluate_NoRules(t *testing.T) {
	flag := makeFlag(true, models.FlagTypeString, "default_val")
	ctx := EvalContext{}

	resp := Evaluate(flag, ctx)
	assert.Equal(t, MustJSON("default_val"), resp.Value)
	assert.False(t, resp.Match)
	assert.Equal(t, ReasonDefault, resp.Reason)
}

func TestEvaluate_SingleConditionMatch(t *testing.T) {
	flag := makeFlag(true, models.FlagTypeBoolean, false,
		makeRule(1, true, makeCond("plan", models.OpEquals, "pro")),
	)
	ctx := EvalContext{"plan": "pro"}

	resp := Evaluate(flag, ctx)
	assert.Equal(t, MustJSON(true), resp.Value)
	assert.True(t, resp.Match)
	assert.Equal(t, ReasonRuleMatch, resp.Reason)
}

func TestEvaluate_SingleConditionNoMatch(t *testing.T) {
	flag := makeFlag(true, models.FlagTypeBoolean, false,
		makeRule(1, true, makeCond("plan", models.OpEquals, "pro")),
	)
	ctx := EvalContext{"plan": "free"}

	resp := Evaluate(flag, ctx)
	assert.Equal(t, MustJSON(false), resp.Value)
	assert.False(t, resp.Match)
	assert.Equal(t, ReasonDefault, resp.Reason)
}

func TestEvaluate_MultiConditionsAllMatch(t *testing.T) {
	flag := makeFlag(true, models.FlagTypeString, "basic",
		makeRule(1, "premium",
			makeCond("plan", models.OpEquals, "pro"),
			makeCond("age", models.OpGTE, 18),
		),
	)
	ctx := EvalContext{"plan": "pro", "age": float64(25)}

	resp := Evaluate(flag, ctx)
	assert.Equal(t, MustJSON("premium"), resp.Value)
	assert.True(t, resp.Match)
}

func TestEvaluate_MultiConditionsPartialMatch(t *testing.T) {
	flag := makeFlag(true, models.FlagTypeString, "basic",
		makeRule(1, "premium",
			makeCond("plan", models.OpEquals, "pro"),
			makeCond("age", models.OpGTE, 18),
		),
	)
	ctx := EvalContext{"plan": "pro", "age": float64(15)}

	resp := Evaluate(flag, ctx)
	assert.Equal(t, MustJSON("basic"), resp.Value)
	assert.False(t, resp.Match)
	assert.Equal(t, ReasonDefault, resp.Reason)
}

func TestEvaluate_PriorityOrdering(t *testing.T) {
	flag := makeFlag(true, models.FlagTypeString, "none",
		makeRule(10, "low_priority", makeCond("active", models.OpEquals, true)),
		makeRule(1, "high_priority", makeCond("active", models.OpEquals, true)),
		makeRule(5, "mid_priority", makeCond("active", models.OpEquals, true)),
	)
	ctx := EvalContext{"active": true}

	resp := Evaluate(flag, ctx)
	assert.Equal(t, MustJSON("high_priority"), resp.Value)
	assert.True(t, resp.Match)
}

func TestEvaluate_SecondRuleMatches(t *testing.T) {
	flag := makeFlag(true, models.FlagTypeNumber, 0,
		makeRule(1, 100, makeCond("plan", models.OpEquals, "enterprise")),
		makeRule(2, 50, makeCond("plan", models.OpEquals, "pro")),
	)
	ctx := EvalContext{"plan": "pro"}

	resp := Evaluate(flag, ctx)
	assert.Equal(t, MustJSON(50), resp.Value)
	assert.True(t, resp.Match)
}

func TestEvaluate_BooleanFlag(t *testing.T) {
	flag := makeFlag(true, models.FlagTypeBoolean, false,
		makeRule(1, true, makeCond("user.plan", models.OpEquals, "pro")),
	)

	resp := Evaluate(flag, EvalContext{
		"user": map[string]interface{}{"plan": "pro"},
	})
	assert.Equal(t, MustJSON(true), resp.Value)
	assert.True(t, resp.Match)
}

func TestEvaluate_NumberFlag(t *testing.T) {
	flag := makeFlag(true, models.FlagTypeNumber, 10,
		makeRule(1, 100, makeCond("tier", models.OpGT, 2)),
	)
	ctx := EvalContext{"tier": float64(3)}

	resp := Evaluate(flag, ctx)
	assert.Equal(t, MustJSON(100), resp.Value)
	assert.True(t, resp.Match)
}

func TestEvaluate_StringFlag(t *testing.T) {
	flag := makeFlag(true, models.FlagTypeString, "v1",
		makeRule(1, "v2", makeCond("beta", models.OpEquals, true)),
	)
	ctx := EvalContext{"beta": true}

	resp := Evaluate(flag, ctx)
	assert.Equal(t, MustJSON("v2"), resp.Value)
	assert.True(t, resp.Match)
}

func TestEvaluate_JSONFlag(t *testing.T) {
	defaultCfg := map[string]interface{}{"theme": "light", "limit": float64(10)}
	ruleCfg := map[string]interface{}{"theme": "dark", "limit": float64(100)}

	flag := makeFlag(true, models.FlagTypeJSON, defaultCfg,
		makeRule(1, ruleCfg, makeCond("plan", models.OpEquals, "pro")),
	)
	ctx := EvalContext{"plan": "pro"}

	resp := Evaluate(flag, ctx)
	assert.True(t, resp.Match)
	// Verify the JSON value can be parsed back
	var got map[string]interface{}
	err := json.Unmarshal(resp.Value, &got)
	assert.NoError(t, err)
	assert.Equal(t, "dark", got["theme"])
}

func TestEvaluate_NestedContext(t *testing.T) {
	flag := makeFlag(true, models.FlagTypeBoolean, false,
		makeRule(1, true,
			makeCond("user.meta.role", models.OpEquals, "admin"),
		),
	)
	ctx := EvalContext{
		"user": map[string]interface{}{
			"meta": map[string]interface{}{
				"role": "admin",
			},
		},
	}

	resp := Evaluate(flag, ctx)
	assert.True(t, resp.Match)
	assert.Equal(t, MustJSON(true), resp.Value)
}

func TestEvaluate_InOperator(t *testing.T) {
	flag := makeFlag(true, models.FlagTypeBoolean, false,
		makeRule(1, true,
			makeCond("country", models.OpIn, []string{"FR", "DE", "IT"}),
		),
	)

	resp := Evaluate(flag, EvalContext{"country": "FR"})
	assert.True(t, resp.Match)

	resp = Evaluate(flag, EvalContext{"country": "US"})
	assert.False(t, resp.Match)
}

func TestEvaluate_RegexOperator(t *testing.T) {
	flag := makeFlag(true, models.FlagTypeBoolean, false,
		makeRule(1, true,
			makeCond("email", models.OpRegex, `^.*@company\.com$`),
		),
	)

	resp := Evaluate(flag, EvalContext{"email": "alice@company.com"})
	assert.True(t, resp.Match)

	resp = Evaluate(flag, EvalContext{"email": "alice@other.com"})
	assert.False(t, resp.Match)
}

// --- Segment tests ---

func makeSegment(key string, conditions ...models.Condition) *models.Segment {
	return &models.Segment{
		Key:        key,
		Conditions: conditions,
	}
}

func makeRuleWithSegments(priority int, value interface{}, segmentKeys []string, conditions ...models.Condition) models.Rule {
	return models.Rule{
		Priority:    priority,
		Value:       MustJSON(value),
		SegmentKeys: segmentKeys,
		Conditions:  conditions,
	}
}

func TestEvaluate_SegmentOnly(t *testing.T) {
	seg := makeSegment("pro_users", makeCond("user.plan", models.OpEquals, "pro"))
	flag := makeFlag(true, models.FlagTypeBoolean, false,
		makeRuleWithSegments(1, true, []string{"pro_users"}),
	)
	flag.Segments = map[string]*models.Segment{"pro_users": seg}

	// Match
	resp := Evaluate(flag, EvalContext{
		"user": map[string]interface{}{"plan": "pro"},
	})
	assert.True(t, resp.Match)
	assert.Equal(t, MustJSON(true), resp.Value)
	assert.Equal(t, ReasonRuleMatch, resp.Reason)

	// No match
	resp = Evaluate(flag, EvalContext{
		"user": map[string]interface{}{"plan": "free"},
	})
	assert.False(t, resp.Match)
	assert.Equal(t, ReasonDefault, resp.Reason)
}

func TestEvaluate_SegmentPlusInlineConditions(t *testing.T) {
	seg := makeSegment("pro_users", makeCond("user.plan", models.OpEquals, "pro"))
	flag := makeFlag(true, models.FlagTypeString, "basic",
		makeRuleWithSegments(1, "premium", []string{"pro_users"},
			makeCond("country", models.OpIn, []string{"FR", "DE"}),
		),
	)
	flag.Segments = map[string]*models.Segment{"pro_users": seg}

	// Both segment and inline match
	resp := Evaluate(flag, EvalContext{
		"user":    map[string]interface{}{"plan": "pro"},
		"country": "FR",
	})
	assert.True(t, resp.Match)
	assert.Equal(t, MustJSON("premium"), resp.Value)

	// Segment matches but inline doesn't
	resp = Evaluate(flag, EvalContext{
		"user":    map[string]interface{}{"plan": "pro"},
		"country": "US",
	})
	assert.False(t, resp.Match)

	// Inline matches but segment doesn't
	resp = Evaluate(flag, EvalContext{
		"user":    map[string]interface{}{"plan": "free"},
		"country": "FR",
	})
	assert.False(t, resp.Match)
}

func TestEvaluate_MissingSegment_FailClosed(t *testing.T) {
	// Rule references a segment that is not in the map
	flag := makeFlag(true, models.FlagTypeBoolean, false,
		makeRuleWithSegments(1, true, []string{"nonexistent_segment"}),
	)
	flag.Segments = map[string]*models.Segment{} // empty map

	resp := Evaluate(flag, EvalContext{"anything": "value"})
	assert.False(t, resp.Match)
	assert.Equal(t, ReasonDefault, resp.Reason)
}

func TestEvaluate_NilSegmentsMap_FailClosed(t *testing.T) {
	flag := makeFlag(true, models.FlagTypeBoolean, false,
		makeRuleWithSegments(1, true, []string{"some_segment"}),
	)
	// Segments is nil (not loaded)

	resp := Evaluate(flag, EvalContext{"anything": "value"})
	assert.False(t, resp.Match)
	assert.Equal(t, ReasonDefault, resp.Reason)
}

func TestEvaluate_MultiSegments(t *testing.T) {
	segPro := makeSegment("pro_users", makeCond("user.plan", models.OpEquals, "pro"))
	segBeta := makeSegment("beta_testers", makeCond("user.beta", models.OpEquals, true))

	flag := makeFlag(true, models.FlagTypeBoolean, false,
		makeRuleWithSegments(1, true, []string{"pro_users", "beta_testers"}),
	)
	flag.Segments = map[string]*models.Segment{
		"pro_users":    segPro,
		"beta_testers": segBeta,
	}

	// Both segments match
	resp := Evaluate(flag, EvalContext{
		"user": map[string]interface{}{"plan": "pro", "beta": true},
	})
	assert.True(t, resp.Match)

	// Only first segment matches
	resp = Evaluate(flag, EvalContext{
		"user": map[string]interface{}{"plan": "pro", "beta": false},
	})
	assert.False(t, resp.Match)
}

// Benchmark

func BenchmarkEvaluate_SimpleRule(b *testing.B) {
	flag := makeFlag(true, models.FlagTypeBoolean, false,
		makeRule(1, true, makeCond("plan", models.OpEquals, "pro")),
	)
	ctx := EvalContext{"plan": "pro"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Evaluate(flag, ctx)
	}
}

func BenchmarkEvaluate_MultiRules(b *testing.B) {
	flag := makeFlag(true, models.FlagTypeBoolean, false,
		makeRule(1, true,
			makeCond("plan", models.OpEquals, "enterprise"),
			makeCond("region", models.OpIn, []string{"US", "EU"}),
		),
		makeRule(2, true,
			makeCond("plan", models.OpEquals, "pro"),
			makeCond("user.age", models.OpGTE, 18),
		),
		makeRule(3, true,
			makeCond("plan", models.OpEquals, "free"),
			makeCond("beta", models.OpEquals, true),
		),
	)
	ctx := EvalContext{
		"plan":   "pro",
		"region": "EU",
		"user":   map[string]interface{}{"age": float64(25)},
		"beta":   false,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Evaluate(flag, ctx)
	}
}

func BenchmarkEvaluate_NestedContext(b *testing.B) {
	flag := makeFlag(true, models.FlagTypeBoolean, false,
		makeRule(1, true,
			makeCond("user.profile.settings.theme", models.OpEquals, "dark"),
		),
	)
	ctx := EvalContext{
		"user": map[string]interface{}{
			"profile": map[string]interface{}{
				"settings": map[string]interface{}{
					"theme": "dark",
				},
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Evaluate(flag, ctx)
	}
}
