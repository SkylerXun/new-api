package billing_curve_setting

import (
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validMonthlyConfig() Config {
	return Config{
		K1:             1,
		K2:             1,
		WindowUSD:      1,
		MonthlyEnabled: true,
		MonthlyTiers: []types.BillingDiscountTier{
			{ThresholdUSD: 1000, DiscountPercent: 10},
		},
	}
}

func TestPrepareConfigJSONSetsAndPreservesFirstEnableCutoff(t *testing.T) {
	original := setting.Config
	t.Cleanup(func() { setting.Config = original })
	setting.Config = Config{K1: 1, K2: 1, WindowUSD: 1}
	now := time.Date(2026, time.August, 27, 12, 0, 0, 0, time.FixedZone("UTC+8", 8*60*60))

	prepared, err := PrepareConfigJSON(`{"k1":1,"k2":1,"window_usd":1,"monthly_enabled":true,"monthly_tiers":[{"threshold_usd":1000,"discount_percent":10}]}`, now)
	require.NoError(t, err)
	var enabled Config
	require.NoError(t, unmarshalConfig(prepared, &enabled))
	assert.Equal(t, now.Unix(), enabled.MonthlyBackfillCutoff)

	setting.Config = validMonthlyConfig()
	setting.Config.MonthlyBackfillCutoff = now.Unix()
	prepared, err = PrepareConfigJSON(`{"k1":1,"k2":1,"window_usd":1,"monthly_enabled":true,"monthly_tiers":[{"threshold_usd":1000,"discount_percent":20}],"monthly_backfill_cutoff":1}`, now.Add(time.Hour))
	require.NoError(t, err)
	var updated Config
	require.NoError(t, unmarshalConfig(prepared, &updated))
	assert.Equal(t, now.Unix(), updated.MonthlyBackfillCutoff)
}

func unmarshalConfig(raw string, target *Config) error {
	return common.UnmarshalJsonStr(raw, target)
}
