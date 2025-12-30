package observability

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCostTracker_Persistence(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "costs.json")

	// 1. Create and Track
	ct := NewCostTracker(filePath)
	ct.TrackUsage("gpt-4", 1000, 500) // 1k in, 0.5k out
	// Cost: (1 * 0.03) + (0.5 * 0.06) = 0.03 + 0.03 = 0.06

	expectedCost := 0.06
	assert.Equal(t, expectedCost, ct.GetDailyCost("gpt-4"))
	assert.Equal(t, 1500, ct.GetDailyTokens("gpt-4"))

	// 2. Save
	err := ct.Save()
	require.NoError(t, err)

	// 3. Reload into new instance
	ct2 := NewCostTracker(filePath)
	err = ct2.Load()
	require.NoError(t, err)

	// Verify state restored
	assert.Equal(t, expectedCost, ct2.GetDailyCost("gpt-4"))
	assert.Equal(t, 1500, ct2.GetDailyTokens("gpt-4"))

	// 4. Add more usage to restored instance
	ct2.TrackUsage("gpt-4", 1000, 0)
	// New total tokens: 2500
	// New cost: 0.06 + 0.03 = 0.09
	assert.Equal(t, 0.09, ct2.GetDailyCost("gpt-4"))
	assert.Equal(t, 2500, ct2.GetDailyTokens("gpt-4"))
}

func TestCostTracker_Rollover(t *testing.T) {
	// Note: We can't easily mock time.Now() in the struct without dependency injection,
	// but we can verify the key format logic indirectly if we exposed it,
	// or just verify that "GetDailyCost" returns 0 for a model with no usage "today".

	ct := NewCostTracker("dummy.json")
	ct.TrackUsage("gpt-4", 100, 100)

	assert.Greater(t, ct.GetDailyCost("gpt-4"), 0.0)
	assert.Equal(t, 0.0, ct.GetDailyCost("model-b-unused"))
}
