package model

import (
	"context"
	"errors"
	"math"

	"github.com/QuantumNous/new-api/common"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// GetUserRechargeQuota returns the cumulative quota purchased through
// successful top-ups and redeemed quota codes. Promotional grants and usage
// are intentionally excluded from this total.
func GetUserRechargeQuota(ctx context.Context, userID int) (int64, error) {
	if userID <= 0 {
		return 0, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return getUserRechargeQuotaDB(DB.WithContext(ctx), userID)
}

func getUserRechargeQuotaDB(db *gorm.DB, userID int) (int64, error) {
	var topUpQuota int64
	if err := db.Model(&TopUp{}).
		Where("user_id = ? AND status = ?", userID, common.TopUpStatusSuccess).
		Select("COALESCE(SUM(amount), 0)").Scan(&topUpQuota).Error; err != nil {
		return 0, err
	}
	var redemptionQuota int64
	if err := db.Model(&Redemption{}).
		Where("used_user_id = ? AND status = ?", userID, common.RedemptionCodeStatusUsed).
		Select("COALESCE(SUM(quota), 0)").Scan(&redemptionQuota).Error; err != nil {
		return 0, err
	}
	if topUpQuota > math.MaxInt64-redemptionQuota {
		return math.MaxInt64, nil
	}
	return topUpQuota + redemptionQuota, nil
}

func GetUserRechargeAmountUSD(ctx context.Context, userID int) (float64, error) {
	quota, err := GetUserRechargeQuota(ctx, userID)
	if err != nil {
		return 0, err
	}
	if common.QuotaPerUnit <= 0 || math.IsNaN(common.QuotaPerUnit) || math.IsInf(common.QuotaPerUnit, 0) {
		return 0, errors.New("quota conversion is unavailable")
	}
	return float64(quota) / common.QuotaPerUnit, nil
}

func RechargeThresholdQuotaUSD(amountUSD float64) (int64, error) {
	if amountUSD <= 0 {
		return 0, nil
	}
	if math.IsNaN(amountUSD) || math.IsInf(amountUSD, 0) || common.QuotaPerUnit <= 0 {
		return 0, errors.New("recharge threshold is invalid")
	}
	quota, err := common.QuotaFromDecimalStrict(
		decimal.NewFromFloat(amountUSD).Mul(decimal.NewFromFloat(common.QuotaPerUnit)),
	)
	if err != nil {
		return 0, err
	}
	return int64(quota), nil
}

func RechargeThresholdQuotaString(amountUSD string) (int64, error) {
	if amountUSD == "" {
		return 0, nil
	}
	amount, err := decimal.NewFromString(amountUSD)
	if err != nil || amount.IsNegative() {
		return 0, errors.New("recharge threshold is invalid")
	}
	if amount.IsZero() {
		return 0, nil
	}
	quota, err := common.QuotaFromDecimalStrict(amount.Mul(decimal.NewFromFloat(common.QuotaPerUnit)))
	if err != nil {
		return 0, err
	}
	return int64(quota), nil
}

func HasUserReachedRechargeThresholdUSD(ctx context.Context, userID int, thresholdUSD float64) (bool, int64, error) {
	thresholdQuota, err := RechargeThresholdQuotaUSD(thresholdUSD)
	if err != nil {
		return false, 0, err
	}
	quota, err := GetUserRechargeQuota(ctx, userID)
	if err != nil {
		return false, 0, err
	}
	return thresholdQuota <= 0 || quota >= thresholdQuota, quota, nil
}

func HasUserReachedRechargeThresholdQuota(ctx context.Context, userID int, thresholdQuota int64) (bool, error) {
	if thresholdQuota <= 0 {
		return true, nil
	}
	quota, err := GetUserRechargeQuota(ctx, userID)
	if err != nil {
		return false, err
	}
	return quota >= thresholdQuota, nil
}

func hasUserReachedRechargeThresholdQuotaDB(db *gorm.DB, userID int, thresholdQuota int64) (bool, error) {
	if thresholdQuota <= 0 {
		return true, nil
	}
	quota, err := getUserRechargeQuotaDB(db, userID)
	if err != nil {
		return false, err
	}
	return quota >= thresholdQuota, nil
}
