package model

import (
	"errors"
	"fmt"
	"math"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type affiliateRebateCredit struct {
	InviterID int
	Quota     int
}

func recordAffiliateRebateLog(credit affiliateRebateCredit, source string) {
	if credit.InviterID <= 0 || credit.Quota <= 0 {
		return
	}
	RecordLog(
		credit.InviterID,
		LogTypeTopup,
		fmt.Sprintf("邀请返利 %s（%s）", logger.LogQuota(credit.Quota), source),
	)
}

// grantAffiliateRebateTx credits the inviter for one newly completed invitee
// transaction. Callers invoke it only from the transaction that consumes a
// redemption code or changes a top-up order from pending to successful, so
// historical transactions are never backfilled and repeated callbacks remain
// idempotent.
func grantAffiliateRebateTx(tx *gorm.DB, inviteeID int, baseQuota int) (affiliateRebateCredit, error) {
	credit := affiliateRebateCredit{}
	if tx == nil || inviteeID <= 0 || baseQuota <= 0 {
		return credit, nil
	}

	setting := operation_setting.GetAffiliateSetting()
	rebatePercent := setting.RedeemRebatePercent
	if !setting.RedeemRebateEnabled ||
		math.IsNaN(rebatePercent) || math.IsInf(rebatePercent, 0) ||
		rebatePercent <= 0 || rebatePercent > 100 {
		return credit, nil
	}

	var invitee User
	if err := lockForUpdate(tx).
		Select("id", "inviter_id").
		Where("id = ?", inviteeID).
		First(&invitee).Error; err != nil {
		return credit, err
	}
	if invitee.InviterId <= 0 || invitee.InviterId == invitee.Id {
		return credit, nil
	}

	rebateQuota, err := common.QuotaFromDecimalStrict(
		decimal.NewFromInt(int64(baseQuota)).
			Mul(decimal.NewFromFloat(rebatePercent)).
			Div(decimal.NewFromInt(100)),
	)
	if err != nil {
		var clamp *common.QuotaClamp
		if errors.As(err, &clamp) {
			common.SysError("affiliate rebate skipped: " + clamp.Error())
			return credit, nil
		}
		return credit, err
	}
	if rebateQuota <= 0 {
		return credit, nil
	}

	var inviter User
	if err := lockForUpdate(tx).
		Select("id", "aff_quota", "aff_history").
		Where("id = ?", invitee.InviterId).
		First(&inviter).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return credit, nil
		}
		return credit, err
	}
	if inviter.Id == invitee.Id ||
		int64(inviter.AffQuota) > common.MaxUserQuota-int64(rebateQuota) ||
		int64(inviter.AffHistoryQuota) > common.MaxUserQuota-int64(rebateQuota) {
		return credit, nil
	}

	result := tx.Model(&User{}).
		Where(
			"id = ? AND aff_quota <= ? AND aff_history <= ?",
			inviter.Id,
			common.MaxUserQuota-int64(rebateQuota),
			common.MaxUserQuota-int64(rebateQuota),
		).
		Updates(map[string]interface{}{
			"aff_quota":   gorm.Expr("aff_quota + ?", rebateQuota),
			"aff_history": gorm.Expr("aff_history + ?", rebateQuota),
		})
	if result.Error != nil {
		return credit, result.Error
	}
	if result.RowsAffected != 1 {
		return credit, nil
	}

	credit.InviterID = inviter.Id
	credit.Quota = rebateQuota
	return credit, nil
}
