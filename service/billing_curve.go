package service

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/billing_curve_setting"
	hosttypes "github.com/QuantumNous/new-api/types"
	"github.com/shopspring/decimal"
)

const billingCurveMicroUSD = 1_000_000.0

var shanghaiLocation = func() *time.Location {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return time.FixedZone("Asia/Shanghai", 8*60*60)
	}
	return loc
}()

func CurrentBillingMonthStart() int64 {
	return billingMonthStartAt(time.Now())
}

func billingMonthStartAt(at time.Time) int64 {
	now := at.In(shanghaiLocation)
	return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, shanghaiLocation).Unix()
}

func CurrentBillingMonthEnd() int64 {
	now := time.Now().In(shanghaiLocation)
	return time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, shanghaiLocation).Unix()
}

func monthlyBackfillCutoff(curve billing_curve_setting.Config, monthStart int64) int64 {
	cutoff := curve.MonthlyBackfillCutoff
	if cutoff <= monthStart {
		return 0
	}
	if billingMonthStartAt(time.Unix(cutoff, 0)) != monthStart {
		return 0
	}
	return cutoff
}

func EnsureCurrentMonthlyBillingBackfill(userIDs []int, curve billing_curve_setting.Config, monthStart int64) error {
	cutoff := monthlyBackfillCutoff(curve, monthStart)
	if cutoff == 0 {
		return nil
	}
	return model.EnsureUserMonthlyBillingBackfill(userIDs, monthStart, cutoff)
}

// ShouldDeferBillingCurveForTokenTask keeps token-priced async tasks out of
// submit-time curve settlement. Their actual base usage is only known when
// polling returns total_tokens, while fixed per-call tasks settle immediately.
func ShouldDeferBillingCurveForTokenTask(curve *hosttypes.BillingCurveConfig, perCallBilling bool) bool {
	return curve != nil && (curve.Enabled || curve.MonthlyEnabled) && !perCallBilling
}

// SnapshotBillingCurveConfig freezes the current settings at pre-consume time.
// It is intentionally harmless for disabled curves so all billing paths can
// call it without branching.
func SnapshotBillingCurveConfig(relayInfo *relaycommon.RelayInfo) hosttypes.BillingCurveConfig {
	if relayInfo != nil {
		if relayInfo.BillingCurveConfig != nil {
			return billing_curve_setting.CloneConfig(*relayInfo.BillingCurveConfig)
		}
		curve := billing_curve_setting.GetConfig()
		snapshot := billing_curve_setting.CloneConfig(curve)
		relayInfo.BillingCurveConfig = &snapshot
		return curve
	}
	return billing_curve_setting.GetConfig()
}

func ReserveBillingCurveQuota(relayInfo *relaycommon.RelayInfo, normalQuota int) (int, error) {
	if normalQuota < 0 {
		return 0, fmt.Errorf("billing curve normal quota cannot be negative")
	}
	if relayInfo != nil {
		relayInfo.BillingCurveNormalPreConsumeQuota = normalQuota
	}
	curve := SnapshotBillingCurveConfig(relayInfo)
	if !curve.Enabled || normalQuota == 0 {
		return normalQuota, nil
	}
	return common.QuotaRoundStrict(float64(normalQuota) * curve.K2)
}

// ApplyBillingCurve applies the configured multiplier to a normal charge that
// already includes the model's configured price and the selected group ratio.
// baseQuotaBeforeGroup is captured before multiplying by group_ratio, avoiding
// rounding loss and keeping a free group from erasing real usage progress.
func ApplyBillingCurve(relayInfo *relaycommon.RelayInfo, normalQuota int, baseQuotaBeforeGroup decimal.Decimal) (int, error) {
	if relayInfo == nil {
		return normalQuota, fmt.Errorf("relay info is required for billing curve")
	}
	baseUsageMicroUSD, err := billingCurveBaseUsageMicroUSD(baseQuotaBeforeGroup)
	if err != nil {
		return 0, err
	}
	return ApplyBillingCurveForBaseUsage(relayInfo, normalQuota, baseUsageMicroUSD)
}

// ApplyBillingCurveForBaseUsage applies the curve to a base usage amount that
// has already been converted to fixed micro-USD. Async tasks persist this
// amount at submission so a later settlement cannot be affected by a change
// to QuotaPerUnit.
func ApplyBillingCurveForBaseUsage(relayInfo *relaycommon.RelayInfo, normalQuota int, baseUsageMicroUSD int64) (int, error) {
	if relayInfo == nil {
		return normalQuota, fmt.Errorf("relay info is required for billing curve")
	}
	if normalQuota < 0 {
		return 0, fmt.Errorf("billing curve normal quota cannot be negative")
	}
	if snapshot := relayInfo.BillingCurveSnapshot; snapshot != nil {
		if snapshot.NormalQuota != normalQuota {
			return 0, fmt.Errorf("billing curve already applied to a different quota")
		}
		if relayInfo.MonthlyDiscountSnapshot != nil {
			return relayInfo.MonthlyDiscountSnapshot.ChargedQuota, nil
		}
		return snapshot.ChargedQuota, nil
	}

	curve := SnapshotBillingCurveConfig(relayInfo)
	if !curve.Enabled || baseUsageMicroUSD <= 0 {
		return ApplyMonthlyDiscount(relayInfo, normalQuota)
	}
	applied := true
	if applied {
		// The interval average never exceeds K2. Validate the hard upper bound
		// before advancing progress so a saturated final charge cannot leave a
		// user's cumulative position ahead of an uncharged request.
		if _, err := common.QuotaRoundStrict(float64(normalQuota) * curve.K2); err != nil {
			return 0, err
		}
	}
	before, after, err := model.AdvanceUserBillingCurveProgress(relayInfo.UserId, baseUsageMicroUSD)
	if err != nil {
		return 0, err
	}

	effectiveMultiplier := EffectiveBillingCurveMultiplier(curve, before, after)
	chargedQuota := normalQuota
	if applied {
		chargedQuota, err = common.QuotaRoundStrict(float64(normalQuota) * effectiveMultiplier)
	}
	if err != nil {
		return 0, err
	}
	if chargedQuota < 0 {
		return 0, fmt.Errorf("billing curve produced a negative charge")
	}

	relayInfo.BillingCurveSnapshot = &hosttypes.BillingCurveSnapshot{
		Applied:                applied,
		NormalQuota:            normalQuota,
		ChargedQuota:           chargedQuota,
		BaseUsageMicroUSD:      baseUsageMicroUSD,
		ProgressBeforeMicroUSD: before,
		ProgressAfterMicroUSD:  after,
		EffectiveMultiplier:    effectiveMultiplier,
		K1:                     curve.K1,
		K2:                     curve.K2,
		ThresholdUSD:           curve.ThresholdUSD,
		WindowUSD:              curve.WindowUSD,
	}
	return ApplyMonthlyDiscount(relayInfo, chargedQuota)
}

// ApplyMonthlyDiscount applies the monthly discount to a charge that already
// includes model, group, and legacy linear-curve pricing. Progress is tracked
// using the actual discounted USD amount and is serialized per user/month.
func ApplyMonthlyDiscount(relayInfo *relaycommon.RelayInfo, normalQuota int) (int, error) {
	if relayInfo == nil {
		return normalQuota, fmt.Errorf("relay info is required for monthly discount")
	}
	if normalQuota < 0 {
		return 0, fmt.Errorf("monthly discount normal quota cannot be negative")
	}
	if relayInfo.MonthlyDiscountSnapshot != nil {
		if relayInfo.MonthlyDiscountSnapshot.NormalQuota != normalQuota {
			return 0, fmt.Errorf("monthly discount already applied to a different quota")
		}
		return relayInfo.MonthlyDiscountSnapshot.ChargedQuota, nil
	}
	curve := SnapshotBillingCurveConfig(relayInfo)
	if !curve.MonthlyEnabled || normalQuota == 0 || common.QuotaPerUnit <= 0 {
		return normalQuota, nil
	}
	monthStart := CurrentBillingMonthStart()
	if err := EnsureCurrentMonthlyBillingBackfill([]int{relayInfo.UserId}, curve, monthStart); err != nil {
		return 0, err
	}
	rawUSD := float64(normalQuota) / common.QuotaPerUnit
	if math.IsNaN(rawUSD) || math.IsInf(rawUSD, 0) || rawUSD <= 0 {
		return normalQuota, nil
	}

	settlementKey := strings.TrimSpace(relayInfo.RequestId)
	if settlementKey == "" {
		settlementKey = common.NewRequestId()
	}
	before, after, chargedQuota, settledMonthStart, err := model.SettleUserMonthlyBillingProgress(relayInfo.UserId, monthStart, settlementKey, normalQuota, func(current int64) (int64, int, error) {
		spent := float64(current) / billingCurveMicroUSD
		chargedUSD, _, calculateErr := calculateMonthlyDiscount(rawUSD, spent, curve.MonthlyTiers)
		if calculateErr != nil {
			return 0, 0, calculateErr
		}
		calculatedQuota, calculateErr := common.QuotaFromFloatStrict(chargedUSD * common.QuotaPerUnit)
		if calculateErr != nil {
			return 0, 0, calculateErr
		}
		actualChargedMicroUSD := int64(math.Round(float64(calculatedQuota) / common.QuotaPerUnit * billingCurveMicroUSD))
		if actualChargedMicroUSD < 0 || current > math.MaxInt64-actualChargedMicroUSD {
			return 0, 0, fmt.Errorf("monthly billing progress overflow")
		}
		return current + actualChargedMicroUSD, calculatedQuota, nil
	})
	if err != nil {
		return 0, err
	}
	if chargedQuota < 0 || chargedQuota > normalQuota {
		return 0, fmt.Errorf("monthly discount produced invalid charge")
	}
	effectiveMultiplier := float64(chargedQuota) / float64(normalQuota)
	discountPercent := (1 - effectiveMultiplier) * 100
	relayInfo.MonthlyDiscountSnapshot = &hosttypes.MonthlyDiscountSnapshot{
		Applied:                chargedQuota < normalQuota,
		NormalQuota:            normalQuota,
		ChargedQuota:           chargedQuota,
		ProgressBeforeMicroUSD: before,
		ProgressAfterMicroUSD:  after,
		DiscountPercent:        discountPercent,
		EffectiveMultiplier:    effectiveMultiplier,
		MonthStart:             settledMonthStart,
		FundingSource:          relayInfo.BillingSource,
		SettlementKey:          settlementKey,
	}
	return chargedQuota, nil
}

func calculateMonthlyDiscount(rawUSD float64, spentUSD float64, tiers []hosttypes.BillingDiscountTier) (chargedUSD float64, afterUSD float64, err error) {
	afterUSD = spentUSD
	remaining := rawUSD
	for remaining > 0 {
		discountPercent := 0.0
		nextThreshold := math.Inf(1)
		for _, tier := range tiers {
			if afterUSD >= tier.ThresholdUSD {
				discountPercent = tier.DiscountPercent
				continue
			}
			nextThreshold = tier.ThresholdUSD
			break
		}
		multiplier := 1 - discountPercent/100
		if multiplier <= 0 {
			return 0, spentUSD, fmt.Errorf("monthly discount multiplier must be positive")
		}
		segmentRaw := remaining
		if !math.IsInf(nextThreshold, 1) {
			next := nextThreshold - afterUSD
			if next > 0 && next/multiplier < segmentRaw {
				segmentRaw = next / multiplier
			}
		}
		chargedUSD += segmentRaw * multiplier
		afterUSD += segmentRaw * multiplier
		remaining -= segmentRaw
	}
	return chargedUSD, afterUSD, nil
}

func MonthlyDiscountPercent(curve billing_curve_setting.Config, spentMicroUSD int64) float64 {
	if !curve.MonthlyEnabled {
		return 0
	}
	percent := 0.0
	spent := float64(spentMicroUSD) / billingCurveMicroUSD
	for _, tier := range curve.MonthlyTiers {
		if spent >= tier.ThresholdUSD {
			percent = tier.DiscountPercent
		} else {
			break
		}
	}
	return percent
}

// PerCallBillingBaseUsageMicroUSD returns a task's known base usage before
// the group multiplier. It is used only as the fallback for task providers
// that complete without reporting actual token usage.
func PerCallBillingBaseUsageMicroUSD(priceData *hosttypes.PriceData) (int64, error) {
	if priceData == nil {
		return 0, fmt.Errorf("price data is required for billing curve")
	}
	baseQuotaBeforeGroup, err := perCallBaseQuotaBeforeGroup(*priceData)
	if err != nil {
		return 0, err
	}
	return billingCurveBaseUsageMicroUSD(baseQuotaBeforeGroup)
}

// ApplyBillingCurveToPerCallPrice applies the curve to a completed per-call
// charge. PriceData.Quota remains the normal configured charge until the
// upstream submission succeeds, then is replaced with the final charge.
func ApplyBillingCurveToPerCallPrice(relayInfo *relaycommon.RelayInfo, priceData *hosttypes.PriceData) error {
	if priceData == nil {
		return fmt.Errorf("price data is required for billing curve")
	}
	baseQuotaBeforeGroup, err := perCallBaseQuotaBeforeGroup(*priceData)
	if err != nil {
		return err
	}
	chargedQuota, err := ApplyBillingCurve(relayInfo, priceData.Quota, baseQuotaBeforeGroup)
	if err != nil {
		return err
	}
	priceData.Quota = chargedQuota
	return nil
}

func perCallBaseQuotaBeforeGroup(priceData hosttypes.PriceData) (decimal.Decimal, error) {
	if common.QuotaPerUnit <= 0 || math.IsNaN(common.QuotaPerUnit) || math.IsInf(common.QuotaPerUnit, 0) {
		return decimal.Zero, fmt.Errorf("invalid quota per unit for billing curve")
	}

	var baseQuota decimal.Decimal
	if priceData.UsePrice {
		if math.IsNaN(priceData.ModelPrice) || math.IsInf(priceData.ModelPrice, 0) {
			return decimal.Zero, fmt.Errorf("invalid per-call model price for billing curve")
		}
		baseQuota = decimal.NewFromFloat(priceData.ModelPrice).
			Mul(decimal.NewFromFloat(common.QuotaPerUnit))
	} else {
		if math.IsNaN(priceData.ModelRatio) || math.IsInf(priceData.ModelRatio, 0) {
			return decimal.Zero, fmt.Errorf("invalid per-call model ratio for billing curve")
		}
		baseQuota = decimal.NewFromFloat(priceData.ModelRatio).
			Div(decimal.NewFromInt(2)).
			Mul(decimal.NewFromFloat(common.QuotaPerUnit))
	}
	return priceData.ApplyOtherRatiosToDecimal(baseQuota), nil
}

func billingCurveBaseUsageMicroUSD(baseQuota decimal.Decimal) (int64, error) {
	if common.QuotaPerUnit <= 0 || math.IsNaN(common.QuotaPerUnit) || math.IsInf(common.QuotaPerUnit, 0) {
		return 0, fmt.Errorf("invalid quota per unit for billing curve")
	}
	baseUSD := baseQuota.Div(decimal.NewFromFloat(common.QuotaPerUnit))
	if baseUSD.LessThanOrEqual(decimal.Zero) {
		return 0, nil
	}
	micros := baseUSD.Mul(decimal.NewFromFloat(billingCurveMicroUSD)).Round(0)
	if micros.GreaterThanOrEqual(decimal.NewFromInt(math.MaxInt64)) {
		return math.MaxInt64, nil
	}
	value := micros.IntPart()
	if value <= 0 {
		return 0, nil
	}
	return value, nil
}

// EffectiveBillingCurveMultiplier returns the interval-average multiplier.
// Integrating the linear transition keeps one combined request equivalent to
// multiple requests that cover the same cumulative base usage.
func EffectiveBillingCurveMultiplier(curve billing_curve_setting.Config, beforeMicroUSD, afterMicroUSD int64) float64 {
	if afterMicroUSD <= beforeMicroUSD {
		return curve.K1
	}
	threshold := curve.ThresholdUSD * billingCurveMicroUSD
	window := curve.WindowUSD * billingCurveMicroUSD
	if window <= 0 || math.IsNaN(threshold) || math.IsNaN(window) || math.IsInf(threshold, 0) || math.IsInf(window, 0) {
		return curve.K1
	}

	start := float64(beforeMicroUSD)
	end := float64(afterMicroUSD)
	total := end - start
	area := 0.0

	if start < threshold {
		segmentEnd := math.Min(end, threshold)
		area += (segmentEnd - start) * curve.K1
		start = segmentEnd
	}

	transitionEnd := threshold + window
	if start < end && start < transitionEnd {
		segmentEnd := math.Min(end, transitionEnd)
		x0 := start - threshold
		x1 := segmentEnd - threshold
		slope := (curve.K2 - curve.K1) / window
		area += curve.K1*(x1-x0) + slope*(x1*x1-x0*x0)/2
		start = segmentEnd
	}

	if start < end {
		area += (end - start) * curve.K2
	}

	if total <= 0 || math.IsNaN(area) || math.IsInf(area, 0) {
		return curve.K1
	}
	return area / total
}

// CurrentBillingCurveMultiplier returns the point multiplier at a user's
// current cumulative base usage, for display in administrative user lists.
func CurrentBillingCurveMultiplier(curve billing_curve_setting.Config, usageMicroUSD int64) float64 {
	if !curve.Enabled {
		return 1
	}
	if usageMicroUSD <= 0 {
		return curve.K1
	}
	threshold := curve.ThresholdUSD * billingCurveMicroUSD
	window := curve.WindowUSD * billingCurveMicroUSD
	usage := float64(usageMicroUSD)
	if usage <= threshold {
		return curve.K1
	}
	if window <= 0 || usage >= threshold+window {
		return curve.K2
	}
	return curve.K1 + (curve.K2-curve.K1)*(usage-threshold)/window
}
