package service

import (
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
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

func TestSnapshotBillingCurveConfigCopiesMonthlyTiers(t *testing.T) {
	source := &types.BillingCurveConfig{
		MonthlyEnabled: true,
		MonthlyTiers: []types.BillingDiscountTier{
			{ThresholdUSD: 1000, DiscountPercent: 10},
		},
	}
	relayInfo := &relaycommon.RelayInfo{BillingCurveConfig: source}

	snapshot := SnapshotBillingCurveConfig(relayInfo)
	source.MonthlyTiers[0].DiscountPercent = 25

	assert.Equal(t, 10.0, snapshot.MonthlyTiers[0].DiscountPercent)
}

func TestMonthlyDiscountPercentUsesHighestReachedTier(t *testing.T) {
	curve := billing_curve_setting.Config{MonthlyEnabled: true, MonthlyTiers: []types.BillingDiscountTier{
		{ThresholdUSD: 1000, DiscountPercent: 10},
		{ThresholdUSD: 2000, DiscountPercent: 20},
	}}

	assert.Equal(t, 0.0, MonthlyDiscountPercent(curve, 999_000_000))
	assert.Equal(t, 10.0, MonthlyDiscountPercent(curve, 1_500_000_000))
	assert.Equal(t, 20.0, MonthlyDiscountPercent(curve, 2_000_000_000))
}

func TestBillingCurveConfigValidationRejectsInvalidMonthlyTiers(t *testing.T) {
	base := billing_curve_setting.Config{K1: 1, K2: 1, WindowUSD: 1}
	base.MonthlyTiers = []types.BillingDiscountTier{{ThresholdUSD: 1000, DiscountPercent: 20}, {ThresholdUSD: 900, DiscountPercent: 30}}
	require.Error(t, billing_curve_setting.ValidateConfig(base))
	base.MonthlyTiers = []types.BillingDiscountTier{{ThresholdUSD: 1000, DiscountPercent: 20}, {ThresholdUSD: 2000, DiscountPercent: 10}}
	require.Error(t, billing_curve_setting.ValidateConfig(base))
}

func TestCalculateMonthlyDiscountSplitsCrossingRequest(t *testing.T) {
	tiers := []types.BillingDiscountTier{{ThresholdUSD: 1000, DiscountPercent: 10}}
	charged, after, err := calculateMonthlyDiscount(20, 990, tiers)
	require.NoError(t, err)
	assert.InDelta(t, 19, charged, 0.000001)
	assert.InDelta(t, 1009, after, 0.000001)
}

func TestCalculateMonthlyDiscountUsesHighestTierAfterMaximum(t *testing.T) {
	tiers := []types.BillingDiscountTier{{ThresholdUSD: 1000, DiscountPercent: 10}, {ThresholdUSD: 2000, DiscountPercent: 20}}
	charged, after, err := calculateMonthlyDiscount(100, 2500, tiers)
	require.NoError(t, err)
	assert.InDelta(t, 80, charged, 0.000001)
	assert.InDelta(t, 2580, after, 0.000001)
}

func TestBillingMonthStartUsesShanghaiCalendarMonth(t *testing.T) {
	at := time.Date(2026, time.August, 31, 16, 30, 0, 0, time.UTC)
	expected := time.Date(2026, time.September, 1, 0, 0, 0, 0, shanghaiLocation).Unix()
	assert.Equal(t, expected, billingMonthStartAt(at))
}
