package operation_setting

import (
	"fmt"
	"math"

	"github.com/QuantumNous/new-api/setting/config"
)

// AffiliateSetting controls the global invitation rebate applied when an
// invitee redeems a balance redemption code. The feature is deliberately
// disabled by default so existing installations keep their current behavior
// until an administrator opts in.
type AffiliateSetting struct {
	RedeemRebateEnabled bool    `json:"redeem_rebate_enabled"`
	RedeemRebatePercent float64 `json:"redeem_rebate_percent"`
}

var affiliateSetting = AffiliateSetting{}

func init() {
	config.GlobalConfig.Register("affiliate_setting", &affiliateSetting)
}

func GetAffiliateSetting() *AffiliateSetting {
	return &affiliateSetting
}

// ValidateRedeemRebatePercent validates the administrator-facing percentage
// before it is persisted. Percentages are finite and inclusive of 0 and 100.
func ValidateRedeemRebatePercent(percent float64) error {
	if math.IsNaN(percent) || math.IsInf(percent, 0) || percent < 0 || percent > 100 {
		return fmt.Errorf("邀请兑换返利比例必须在 0 到 100 之间")
	}
	return nil
}
