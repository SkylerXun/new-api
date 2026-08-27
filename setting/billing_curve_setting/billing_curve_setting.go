package billing_curve_setting

import (
	"fmt"
	"math"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/config"
	hosttypes "github.com/QuantumNous/new-api/types"
)

const ConfigOptionKey = "billing_curve_setting.config"

// Config is a relay-safe snapshot type shared with RelayInfo.
type Config = hosttypes.BillingCurveConfig

type Setting struct {
	Config Config `json:"config"`
}

var setting = Setting{
	Config: Config{
		Enabled:               false,
		K1:                    5,
		K2:                    15,
		ThresholdUSD:          75,
		WindowUSD:             150,
		TargetAverageK:        10,
		MonthlyEnabled:        false,
		MonthlyTiers:          []hosttypes.BillingDiscountTier{},
		MonthlyBackfillCutoff: 0,
	},
}

func init() {
	config.GlobalConfig.Register("billing_curve_setting", &setting)
}

func GetConfig() Config {
	return CloneConfig(setting.Config)
}

func CloneConfig(source Config) Config {
	cloned := source
	cloned.MonthlyTiers = append([]hosttypes.BillingDiscountTier(nil), source.MonthlyTiers...)
	return cloned
}

// ValidateConfigJSON validates the single, atomic configuration payload used
// by the settings page before it can be persisted.
func ValidateConfigJSON(raw string) error {
	var candidate Config
	if err := common.UnmarshalJsonStr(raw, &candidate); err != nil {
		return fmt.Errorf("invalid billing curve configuration: %w", err)
	}
	return ValidateConfig(candidate)
}

// PrepareConfigJSON owns the internal first-enable cutoff. Clients preserve
// the value on later edits but cannot move it or manufacture an older cutoff.
func PrepareConfigJSON(raw string, now time.Time) (string, error) {
	var candidate Config
	if err := common.UnmarshalJsonStr(raw, &candidate); err != nil {
		return "", fmt.Errorf("invalid billing curve configuration: %w", err)
	}
	current := GetConfig()
	if current.MonthlyBackfillCutoff > 0 {
		candidate.MonthlyBackfillCutoff = current.MonthlyBackfillCutoff
	} else if !current.MonthlyEnabled && candidate.MonthlyEnabled {
		candidate.MonthlyBackfillCutoff = now.Unix()
	} else {
		candidate.MonthlyBackfillCutoff = 0
	}
	if err := ValidateConfig(candidate); err != nil {
		return "", err
	}
	encoded, err := common.Marshal(candidate)
	if err != nil {
		return "", fmt.Errorf("encode billing curve configuration: %w", err)
	}
	return string(encoded), nil
}

func ValidateConfig(candidate Config) error {
	values := []float64{
		candidate.K1,
		candidate.K2,
		candidate.ThresholdUSD,
		candidate.WindowUSD,
	}
	for _, value := range values {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return fmt.Errorf("billing curve values must be finite")
		}
	}
	if candidate.K1 <= 0 || candidate.K1 > 1_000_000 {
		return fmt.Errorf("billing curve K1 must be between 0 and 1000000")
	}
	if candidate.K2 < candidate.K1 || candidate.K2 > 1_000_000 {
		return fmt.Errorf("billing curve K2 must be between K1 and 1000000")
	}
	if candidate.ThresholdUSD < 0 || candidate.ThresholdUSD > 1_000_000_000_000 {
		return fmt.Errorf("billing curve threshold must be between 0 and 1000000000000")
	}
	if candidate.WindowUSD <= 0 || candidate.WindowUSD > 1_000_000_000_000 {
		return fmt.Errorf("billing curve window must be between 0 and 1000000000000")
	}
	if len(candidate.MonthlyTiers) > 100 {
		return fmt.Errorf("monthly billing tiers cannot exceed 100")
	}
	if candidate.MonthlyBackfillCutoff < 0 {
		return fmt.Errorf("monthly billing backfill cutoff cannot be negative")
	}
	if candidate.MonthlyEnabled && len(candidate.MonthlyTiers) == 0 {
		return fmt.Errorf("monthly billing requires at least one tier when enabled")
	}
	previousThreshold := 0.0
	previousDiscount := 0.0
	for i, tier := range candidate.MonthlyTiers {
		if math.IsNaN(tier.ThresholdUSD) || math.IsInf(tier.ThresholdUSD, 0) || math.IsNaN(tier.DiscountPercent) || math.IsInf(tier.DiscountPercent, 0) {
			return fmt.Errorf("monthly billing tier values must be finite")
		}
		if tier.ThresholdUSD <= previousThreshold || tier.ThresholdUSD > 1_000_000_000_000 {
			return fmt.Errorf("monthly billing tier %d threshold must be strictly increasing", i+1)
		}
		if tier.DiscountPercent < previousDiscount || tier.DiscountPercent < 0 || tier.DiscountPercent >= 100 {
			return fmt.Errorf("monthly billing tier %d discount must be between 0 and less than 100 and non-decreasing", i+1)
		}
		previousThreshold = tier.ThresholdUSD
		previousDiscount = tier.DiscountPercent
	}
	return nil
}
