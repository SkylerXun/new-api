package types

// BillingCurveConfig controls the global progressive multiplier. It is kept
// here so RelayInfo can snapshot it without importing a setting package.
type BillingCurveConfig struct {
	Enabled        bool    `json:"enabled"`
	K1             float64 `json:"k1"`
	K2             float64 `json:"k2"`
	ThresholdUSD   float64 `json:"threshold_usd"`
	WindowUSD      float64 `json:"window_usd"`
	TargetAverageK float64 `json:"target_average_k"`
}

// BillingCurveSnapshot records the exact internal curve calculation attached
// to one relay request. It is written only under a consume log's admin_info.
type BillingCurveSnapshot struct {
	Applied                bool    `json:"applied"`
	NormalQuota            int     `json:"normal_quota"`
	ChargedQuota           int     `json:"charged_quota"`
	BaseUsageMicroUSD      int64   `json:"base_usage_micro_usd"`
	ProgressBeforeMicroUSD int64   `json:"progress_before_micro_usd"`
	ProgressAfterMicroUSD  int64   `json:"progress_after_micro_usd"`
	EffectiveMultiplier    float64 `json:"effective_multiplier"`
	K1                     float64 `json:"k1"`
	K2                     float64 `json:"k2"`
	ThresholdUSD           float64 `json:"threshold_usd"`
	WindowUSD              float64 `json:"window_usd"`
}
