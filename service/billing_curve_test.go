package service

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/billing_curve_setting"
	"github.com/QuantumNous/new-api/types"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEffectiveBillingCurveMultiplierIntegratesTransition(t *testing.T) {
	curve := billing_curve_setting.Config{
		K1:           5,
		K2:           15,
		ThresholdUSD: 75,
		WindowUSD:    150,
	}

	warmup := EffectiveBillingCurveMultiplier(curve, 0, 75_000_000)
	transition := EffectiveBillingCurveMultiplier(curve, 75_000_000, 225_000_000)
	postTransition := EffectiveBillingCurveMultiplier(curve, 225_000_000, 300_000_000)

	assert.InDelta(t, 5.0, warmup, 0.000001)
	assert.InDelta(t, 10.0, transition, 0.000001)
	assert.InDelta(t, 15.0, postTransition, 0.000001)
}

func TestPerCallBaseQuotaBeforeGroupExcludesGroupRatio(t *testing.T) {
	originalQuotaPerUnit := common.QuotaPerUnit
	common.QuotaPerUnit = 500_000
	t.Cleanup(func() {
		common.QuotaPerUnit = originalQuotaPerUnit
	})

	priceData := types.PriceData{
		UsePrice:   true,
		ModelPrice: 2,
		GroupRatioInfo: types.GroupRatioInfo{
			GroupRatio: 3,
		},
	}
	priceData.AddOtherRatio("image_count", 2)

	baseQuota, err := perCallBaseQuotaBeforeGroup(priceData)

	require.NoError(t, err)
	assert.True(t, decimal.NewFromInt(2_000_000).Equal(baseQuota))
}

func TestBillingCurveConfigValidationRejectsInvalidOrder(t *testing.T) {
	err := billing_curve_setting.ValidateConfig(billing_curve_setting.Config{
		K1:             15,
		K2:             5,
		ThresholdUSD:   75,
		WindowUSD:      150,
		TargetAverageK: 10,
	})

	require.Error(t, err)
}

func TestShouldDeferBillingCurveForTokenTask(t *testing.T) {
	curve := &types.BillingCurveConfig{Enabled: true}

	assert.True(t, ShouldDeferBillingCurveForTokenTask(curve, false))
	assert.False(t, ShouldDeferBillingCurveForTokenTask(curve, true))
	assert.False(t, ShouldDeferBillingCurveForTokenTask(&types.BillingCurveConfig{}, false))
	assert.False(t, ShouldDeferBillingCurveForTokenTask(nil, false))
}
