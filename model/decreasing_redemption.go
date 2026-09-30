package model

import (
	"errors"
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

// DecreasingRedemption is owned by the optional decreasing-redemption feature.
// It is deliberately separate from Redemption so ordinary codes never change.
type DecreasingRedemption struct {
	ID                   int    `json:"id"`
	Key                  string `json:"key" gorm:"type:char(32);uniqueIndex"`
	Name                 string `json:"name"`
	TotalQuota           int    `json:"total_quota"`
	RemainingQuota       int    `json:"remaining_quota"`
	MonthlyLimitQuota    int    `json:"monthly_limit_quota"`
	MonthlyRedeemedQuota int    `json:"monthly_redeemed_quota"`
	MonthlyPeriod        int64  `json:"monthly_period" gorm:"bigint;index"`
	Enabled              bool   `json:"enabled"`
	CreatedAt            int64  `json:"created_at" gorm:"bigint"`
}

type DecreasingRedeemResult struct {
	Quota          int   `json:"quota"`
	RequestedQuota int   `json:"requested_quota"`
	MonthlyLimit   int   `json:"monthly_limit"`
	MonthlyRemaining int  `json:"monthly_remaining"`
	NextRefreshAt  int64 `json:"next_refresh_at"`
	Decreasing     bool  `json:"decreasing"`
}

func currentMonth() (int64, int64) {
	now := time.Now()
	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	next := start.AddDate(0, 1, 0)
	return start.Unix(), next.Unix()
}

// RedeemDecreasing handles only keys registered by the optional feature.
// handled=false means the caller must use the legacy redemption path.
func RedeemDecreasing(key string, userID int) (result DecreasingRedeemResult, handled bool, err error) {
	var code DecreasingRedemption
	if err = DB.Where("key = ? AND enabled = ?", key, true).First(&code).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) { return result, false, nil }
		return result, true, err
	}
	handled = true
	month, nextRefresh := currentMonth()
	err = DB.Transaction(func(tx *gorm.DB) error {
		if err := lockForUpdate(tx).Where("id = ?", code.ID).First(&code).Error; err != nil { return err }
		used := code.MonthlyRedeemedQuota
		if code.MonthlyPeriod != month { used = 0 }
		remaining := code.MonthlyLimitQuota - used
		if remaining <= 0 { return errors.New("本月兑换额度已用完") }
		if code.RemainingQuota <= 0 { return errors.New("兑换码额度已用完") }
		credit := code.RemainingQuota
		if credit > remaining { credit = remaining }
		var user User
		if err := lockForUpdate(tx).Select("id, quota").Where("id = ?", userID).First(&user).Error; err != nil { return err }
		if int64(user.Quota) > common.MaxUserQuota-int64(credit) { return errors.New("用户额度超出可支持范围") }
		if err := tx.Model(&User{}).Where("id = ?", userID).Update("quota", gorm.Expr("quota + ?", credit)).Error; err != nil { return err }
		if err := tx.Model(&code).Updates(map[string]any{"remaining_quota": code.RemainingQuota - credit, "monthly_redeemed_quota": used + credit, "monthly_period": month}).Error; err != nil { return err }
		result = DecreasingRedeemResult{Quota: credit, RequestedQuota: code.TotalQuota, MonthlyLimit: code.MonthlyLimitQuota, MonthlyRemaining: remaining - credit, NextRefreshAt: nextRefresh, Decreasing: true}
		return nil
	})
	if err == nil {
		syncCreditUserQuotaCache(userID, result.Quota, "decreasing_redemption")
		RecordLog(userID, LogTypeTopup, "通过递减兑换码充值")
	}
	return result, handled, err
}
