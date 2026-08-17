package billing_curve_setting

import (
	"fmt"
	"math"

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
		Enabled:        false,
		K1:             5,
		K2:             15,
		ThresholdUSD:   75,
		WindowUSD:      150,
		TargetAverageK: 10,
	},
}

func init() {
	config.GlobalConfig.Register("billing_curve_setting", &setting)
}

func GetConfig() Config {
	return setting.Config
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

func ValidateConfig(candidate Config) error {
	values := []float64{
		candidate.K1,
		candidate.K2,
		candidate.ThresholdUSD,
		candidate.WindowUSD,
		candidate.TargetAverageK,
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
	if candidate.TargetAverageK < candidate.K1 || candidate.TargetAverageK > candidate.K2 {
		return fmt.Errorf("billing curve target average must be between K1 and K2")
	}
	return nil
}
