package engine

import (
	"fmt"
	"testing"

	"github.com/getflaggy/flaggy/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestMurmur3_32_Deterministic(t *testing.T) {
	h1 := murmur3_32([]byte("test_flag:user_123"), 0)
	h2 := murmur3_32([]byte("test_flag:user_123"), 0)
	assert.Equal(t, h1, h2, "same input must produce same hash")
}

func TestMurmur3_32_DifferentInputs(t *testing.T) {
	h1 := murmur3_32([]byte("flag_a:user_1"), 0)
	h2 := murmur3_32([]byte("flag_b:user_1"), 0)
	assert.NotEqual(t, h1, h2, "different inputs should produce different hashes")
}

func TestRolloutBucket_Range(t *testing.T) {
	for i := 0; i < 1000; i++ {
		bucket := RolloutBucket("test_flag", fmt.Sprintf("user_%d", i))
		assert.GreaterOrEqual(t, bucket, 0)
		assert.Less(t, bucket, 100)
	}
}

func TestRolloutBucket_Deterministic(t *testing.T) {
	b1 := RolloutBucket("my_flag", "user_42")
	b2 := RolloutBucket("my_flag", "user_42")
	assert.Equal(t, b1, b2)
}

func TestInRollout_EdgeCases(t *testing.T) {
	assert.False(t, InRollout("flag", "user", 0), "0% should never be in rollout")
	assert.True(t, InRollout("flag", "user", 100), "100% should always be in rollout")
}

func TestInRollout_Consistency(t *testing.T) {
	// Same user should always get the same result
	result1 := InRollout("feature_x", "user_99", 50)
	result2 := InRollout("feature_x", "user_99", 50)
	assert.Equal(t, result1, result2)
}

// TestRolloutDistribution verifies uniform distribution over 10k users.
// With 50% rollout, we expect ~5000 users in, with tolerance for statistical variance.
func TestRolloutDistribution(t *testing.T) {
	tests := []struct {
		percentage int
		tolerance  float64 // allowed deviation from expected ratio
	}{
		{10, 0.03},
		{25, 0.03},
		{50, 0.03},
		{75, 0.03},
		{90, 0.03},
	}

	const numUsers = 10000

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%d_percent", tt.percentage), func(t *testing.T) {
			inCount := 0
			for i := 0; i < numUsers; i++ {
				if InRollout("distribution_test", fmt.Sprintf("user_%d", i), tt.percentage) {
					inCount++
				}
			}

			actualRatio := float64(inCount) / float64(numUsers)
			expectedRatio := float64(tt.percentage) / 100.0

			assert.InDelta(t, expectedRatio, actualRatio, tt.tolerance,
				"expected ~%.0f%% but got %.1f%% (%d/%d)",
				expectedRatio*100, actualRatio*100, inCount, numUsers)
		})
	}
}

// TestRolloutDistribution_DifferentFlags verifies that different flags
// produce different bucketing (so a user in flag A's rollout isn't
// necessarily in flag B's rollout).
func TestRolloutDistribution_DifferentFlags(t *testing.T) {
	const numUsers = 10000
	flagA := 0
	flagB := 0
	both := 0

	for i := 0; i < numUsers; i++ {
		uid := fmt.Sprintf("user_%d", i)
		inA := InRollout("flag_alpha", uid, 50)
		inB := InRollout("flag_beta", uid, 50)
		if inA {
			flagA++
		}
		if inB {
			flagB++
		}
		if inA && inB {
			both++
		}
	}

	// If flags were independent, overlap should be ~25% of total
	overlapRatio := float64(both) / float64(numUsers)
	assert.InDelta(t, 0.25, overlapRatio, 0.04,
		"overlap between two 50%% rollouts should be ~25%%, got %.1f%%", overlapRatio*100)

	// Each flag should be ~50%
	assert.InDelta(t, 0.5, float64(flagA)/float64(numUsers), 0.03)
	assert.InDelta(t, 0.5, float64(flagB)/float64(numUsers), 0.03)
}

func TestEvaluate_RolloutPercentage(t *testing.T) {
	flag := &models.Flag{
		Key:          "rollout_flag",
		Type:         models.FlagTypeBoolean,
		Enabled:      true,
		DefaultValue: MustJSON(false),
		Rules: []models.Rule{
			{
				Priority:          1,
				Value:             MustJSON(true),
				RolloutPercentage: 50,
				Conditions: []models.Condition{
					{Attribute: "plan", Operator: models.OpEquals, Value: MustJSON("pro")},
				},
			},
		},
	}

	// Run with many users, roughly half should get true
	inCount := 0
	total := 1000
	for i := 0; i < total; i++ {
		ctx := EvalContext{
			"plan":      "pro",
			"entity_id": fmt.Sprintf("user_%d", i),
		}
		resp := Evaluate(flag, ctx)
		if resp.Match {
			inCount++
		}
	}

	ratio := float64(inCount) / float64(total)
	assert.InDelta(t, 0.5, ratio, 0.06)
}

func TestEvaluate_RolloutZeroPercent(t *testing.T) {
	flag := &models.Flag{
		Key:          "no_rollout",
		Type:         models.FlagTypeBoolean,
		Enabled:      true,
		DefaultValue: MustJSON(false),
		Rules: []models.Rule{
			{
				Priority:          1,
				Value:             MustJSON(true),
				RolloutPercentage: 0,
				Conditions: []models.Condition{
					{Attribute: "plan", Operator: models.OpEquals, Value: MustJSON("pro")},
				},
			},
		},
	}

	// RolloutPercentage=0 means no rollout check, rule matches normally
	ctx := EvalContext{"plan": "pro", "entity_id": "user_1"}
	resp := Evaluate(flag, ctx)
	assert.True(t, resp.Match)
}

func TestEvaluate_Rollout100Percent(t *testing.T) {
	flag := &models.Flag{
		Key:          "full_rollout",
		Type:         models.FlagTypeBoolean,
		Enabled:      true,
		DefaultValue: MustJSON(false),
		Rules: []models.Rule{
			{
				Priority:          1,
				Value:             MustJSON(true),
				RolloutPercentage: 100,
				Conditions: []models.Condition{
					{Attribute: "plan", Operator: models.OpEquals, Value: MustJSON("pro")},
				},
			},
		},
	}

	ctx := EvalContext{"plan": "pro", "entity_id": "user_1"}
	resp := Evaluate(flag, ctx)
	assert.True(t, resp.Match)
}

func TestEvaluate_RolloutNoEntityID(t *testing.T) {
	flag := &models.Flag{
		Key:          "rollout_no_entity",
		Type:         models.FlagTypeBoolean,
		Enabled:      true,
		DefaultValue: MustJSON(false),
		Rules: []models.Rule{
			{
				Priority:          1,
				Value:             MustJSON(true),
				RolloutPercentage: 50,
				Conditions: []models.Condition{
					{Attribute: "plan", Operator: models.OpEquals, Value: MustJSON("pro")},
				},
			},
		},
	}

	// No entity_id → skips the rule (can't bucket)
	ctx := EvalContext{"plan": "pro"}
	resp := Evaluate(flag, ctx)
	assert.False(t, resp.Match)
	assert.Equal(t, ReasonDefault, resp.Reason)
}

func TestEvaluate_RolloutFallsThrough(t *testing.T) {
	// Rule 1: 10% rollout, Rule 2: always matches
	flag := &models.Flag{
		Key:          "fallthrough_flag",
		Type:         models.FlagTypeString,
		Enabled:      true,
		DefaultValue: MustJSON("default"),
		Rules: []models.Rule{
			{
				Priority:          1,
				Value:             MustJSON("canary"),
				RolloutPercentage: 10,
				Conditions: []models.Condition{
					{Attribute: "active", Operator: models.OpEquals, Value: MustJSON(true)},
				},
			},
			{
				Priority:          2,
				Value:             MustJSON("stable"),
				RolloutPercentage: 0,
				Conditions: []models.Condition{
					{Attribute: "active", Operator: models.OpEquals, Value: MustJSON(true)},
				},
			},
		},
	}

	// Find a user not in the 10% rollout
	var found bool
	for i := 0; i < 100; i++ {
		uid := fmt.Sprintf("user_%d", i)
		if !InRollout("fallthrough_flag", uid, 10) {
			ctx := EvalContext{"active": true, "entity_id": uid}
			resp := Evaluate(flag, ctx)
			assert.Equal(t, MustJSON("stable"), resp.Value,
				"user not in rollout should fall through to rule 2")
			found = true
			break
		}
	}
	assert.True(t, found, "should find at least one user not in 10%% rollout")
}
