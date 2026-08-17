package operation_setting

import "github.com/QuantumNous/new-api/setting/config"

type ActivitySetting struct {
	NewUserRedeemBonusEnabled    bool    `json:"new_user_redeem_bonus_enabled"`
	NewUserRedeemBonusPercent    float64 `json:"new_user_redeem_bonus_percent"`
	NewUserRedeemBonusWindowDays int     `json:"new_user_redeem_bonus_window_days"`
}

var activitySetting = ActivitySetting{
	NewUserRedeemBonusEnabled:    true,
	NewUserRedeemBonusPercent:    30,
	NewUserRedeemBonusWindowDays: 1,
}

func init() {
	config.GlobalConfig.Register("activity_setting", &activitySetting)
}

func GetActivitySetting() *ActivitySetting {
	return &activitySetting
}
