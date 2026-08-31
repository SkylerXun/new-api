package model

import (
	"errors"
	"fmt"
	"math"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type Redemption struct {
	Id                   int            `json:"id"`
	UserId               int            `json:"user_id"`
	Key                  string         `json:"key" gorm:"type:char(32);uniqueIndex"`
	Status               int            `json:"status" gorm:"default:1"`
	Name                 string         `json:"name" gorm:"index"`
	Quota                int            `json:"quota" gorm:"default:100"`
	CreatedTime          int64          `json:"created_time" gorm:"bigint"`
	RedeemedTime         int64          `json:"redeemed_time" gorm:"bigint"`
	Count                int            `json:"count" gorm:"-:all"` // only for api request
	UsedUserId           int            `json:"used_user_id"`
	DeletedAt            gorm.DeletedAt `gorm:"index"`
	ExpiredTime          int64          `json:"expired_time" gorm:"bigint"` // 过期时间，0 表示不过期
	CategoryID           int            `json:"category_id" gorm:"index"`
	CategoryNameSnapshot string         `json:"category_name" gorm:"type:varchar(80)"`
	CategoryPriceCents   int64          `json:"category_price_cents"`
	CategoryPricedAt     int64          `json:"category_priced_at" gorm:"bigint;index"`
	CategoryPricedBy     int            `json:"category_priced_by"`
}

func GetAllRedemptions(startIdx int, num int) (redemptions []*Redemption, total int64, err error) {
	// 开始事务
	tx := DB.Begin()
	if tx.Error != nil {
		return nil, 0, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 获取总数
	err = tx.Model(&Redemption{}).Count(&total).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	// 获取分页数据
	err = tx.Order("id desc").Limit(num).Offset(startIdx).Find(&redemptions).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	// 提交事务
	if err = tx.Commit().Error; err != nil {
		return nil, 0, err
	}

	return redemptions, total, nil
}

func SearchRedemptions(keyword string, status string, startIdx int, num int) (redemptions []*Redemption, total int64, err error) {
	tx := DB.Begin()
	if tx.Error != nil {
		return nil, 0, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	query := tx.Model(&Redemption{})

	if keyword != "" {
		if id, err := strconv.Atoi(keyword); err == nil {
			query = query.Where("id = ? OR name LIKE ?", id, keyword+"%")
		} else {
			query = query.Where("name LIKE ?", keyword+"%")
		}
	}

	if status != "" {
		now := common.GetTimestamp()
		switch status {
		case "expired":
			query = query.Where(
				"status = ? AND expired_time != 0 AND expired_time < ?",
				common.RedemptionCodeStatusEnabled,
				now,
			)
		case strconv.Itoa(common.RedemptionCodeStatusEnabled):
			query = query.Where(
				"status = ? AND (expired_time = 0 OR expired_time >= ?)",
				common.RedemptionCodeStatusEnabled,
				now,
			)
		case strconv.Itoa(common.RedemptionCodeStatusDisabled):
			query = query.Where("status = ?", common.RedemptionCodeStatusDisabled)
		case strconv.Itoa(common.RedemptionCodeStatusUsed):
			query = query.Where("status = ?", common.RedemptionCodeStatusUsed)
		}
	}

	// Get total count
	err = query.Count(&total).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	// Get paginated data
	err = query.Order("id desc").Limit(num).Offset(startIdx).Find(&redemptions).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	if err = tx.Commit().Error; err != nil {
		return nil, 0, err
	}

	return redemptions, total, nil
}

func GetRedemptionById(id int) (*Redemption, error) {
	if id == 0 {
		return nil, errors.New("id 为空！")
	}
	redemption := Redemption{Id: id}
	var err error = nil
	err = DB.First(&redemption, "id = ?", id).Error
	return &redemption, err
}

func Redeem(key string, userId int) (quota int, err error) {
	if key == "" {
		return 0, errors.New("未提供兑换码")
	}
	if userId == 0 {
		return 0, errors.New("无效的 user id")
	}
	redemption := &Redemption{}
	bonusQuota := 0

	keyCol := "`key`"
	if common.UsingMainDatabase(common.DatabaseTypePostgreSQL) {
		keyCol = `"key"`
	}
	common.RandomSleep()
	rebateQuota := 0
	rebateInviterId := 0
	err = DB.Transaction(func(tx *gorm.DB) error {
		err := lockForUpdate(tx).Where(keyCol+" = ?", key).First(redemption).Error
		if err != nil {
			return errors.New("无效的兑换码")
		}
		if redemption.Status != common.RedemptionCodeStatusEnabled {
			return errors.New("该兑换码已被使用")
		}
		now := common.GetTimestamp()
		if redemption.ExpiredTime != 0 && redemption.ExpiredTime < now {
			return errors.New("该兑换码已过期")
		}

		var user User
		if err := lockForUpdate(tx).Select("id", "quota", "created_at", "inviter_id").Where("id = ?", userId).First(&user).Error; err != nil {
			return err
		}
		// Compare-and-swap on status: only the transaction that flips
		// enabled -> used may credit quota, so a concurrent redeem of the
		// same code loses here even without a row lock (e.g. on SQLite).
		result := tx.Model(&Redemption{}).
			Where("id = ? AND status = ?", redemption.Id, common.RedemptionCodeStatusEnabled).
			Updates(map[string]interface{}{
				"redeemed_time": now,
				"status":        common.RedemptionCodeStatusUsed,
				"used_user_id":  userId,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return errors.New("该兑换码已被使用")
		}

		result = tx.Model(&User{}).
			Where("id = ? AND quota <= ?", userId, common.MaxQuota-redemption.Quota).
			Update("quota", gorm.Expr("quota + ?", redemption.Quota))
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return errors.New("用户额度超出可支持范围")
		}

		activitySetting := operation_setting.GetActivitySetting()
		bonusPercent := activitySetting.NewUserRedeemBonusPercent
		windowDays := activitySetting.NewUserRedeemBonusWindowDays
		windowSeconds := int64(windowDays) * 24 * 60 * 60
		eligible := activitySetting.NewUserRedeemBonusEnabled &&
			!math.IsNaN(bonusPercent) &&
			!math.IsInf(bonusPercent, 0) &&
			bonusPercent > 0 && bonusPercent <= 1000 &&
			windowDays >= 1 && windowDays <= 3650 &&
			user.CreatedAt > 0 &&
			now < user.CreatedAt+windowSeconds
		if eligible {
			calculatedBonus, err := common.QuotaFromDecimalStrict(
				decimal.NewFromInt(int64(redemption.Quota)).
					Mul(decimal.NewFromFloat(bonusPercent)).
					Div(decimal.NewFromInt(100)),
			)
			if err != nil {
				return err
			}
			if calculatedBonus > 0 {
				granted, err := GrantActivityQuotaTx(
					tx,
					userId,
					ActivityKeyNewUserRedeemBonus,
					ActivityGrantSourceRedeem,
					"redemption:"+strconv.Itoa(redemption.Id),
					calculatedBonus,
				)
				if err != nil {
					return err
				}
				if granted {
					bonusQuota = calculatedBonus
				}
			}
		}

		// Invitation rebate is independent from the new-user activity bonus.
		// It is calculated from the original redemption quota only, so an
		// invitee's promotional bonus never increases the inviter's rebate.
		affiliateSetting := operation_setting.GetAffiliateSetting()
		rebatePercent := affiliateSetting.RedeemRebatePercent
		rebateEligible := affiliateSetting.RedeemRebateEnabled &&
			!math.IsNaN(rebatePercent) &&
			!math.IsInf(rebatePercent, 0) &&
			rebatePercent > 0 && rebatePercent <= 100 &&
			redemption.Quota > 0 &&
			user.InviterId > 0 && user.InviterId != user.Id
		if !rebateEligible {
			return nil
		}

		calculatedRebate, err := common.QuotaFromDecimalStrict(
			decimal.NewFromInt(int64(redemption.Quota)).
				Mul(decimal.NewFromFloat(rebatePercent)).
				Div(decimal.NewFromInt(100)),
		)
		if err != nil {
			// A malformed/oversized historical redemption must not prevent the
			// invitee from using the code. The strict conversion protects the
			// int32 quota columns; skip only this optional rebate on saturation.
			var clamp *common.QuotaClamp
			if errors.As(err, &clamp) {
				common.SysError("affiliate rebate skipped: " + clamp.Error())
				return nil
			}
			return err
		}
		if calculatedRebate <= 0 {
			return nil
		}

		var inviter User
		if err := lockForUpdate(tx).
			Select("id", "aff_quota", "aff_history").
			Where("id = ?", user.InviterId).
			First(&inviter).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return err
		}
		if inviter.Id == user.Id ||
			inviter.AffQuota > common.MaxQuota-calculatedRebate ||
			inviter.AffHistoryQuota > common.MaxQuota-calculatedRebate {
			// Rebate balances are bounded by the same int32 quota ceiling as
			// wallet balances. Reaching the ceiling skips this optional credit.
			return nil
		}

		result = tx.Model(&User{}).
			Where("id = ? AND aff_quota <= ? AND aff_history <= ?", user.InviterId, common.MaxQuota-calculatedRebate, common.MaxQuota-calculatedRebate).
			Updates(map[string]interface{}{
				"aff_quota":   gorm.Expr("aff_quota + ?", calculatedRebate),
				"aff_history": gorm.Expr("aff_history + ?", calculatedRebate),
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return nil
		}
		rebateQuota = calculatedRebate
		rebateInviterId = inviter.Id
		return nil
	})
	if err != nil {
		common.SysError("redemption failed: " + err.Error())
		return 0, ErrRedeemFailed
	}
	totalQuota := redemption.Quota + bonusQuota
	syncCreditUserQuotaCache(userId, totalQuota, "redemption")
	content := fmt.Sprintf("通过兑换码充值 %s，兑换码ID %d", logger.LogQuota(redemption.Quota), redemption.Id)
	if bonusQuota > 0 {
		content += fmt.Sprintf("，新用户活动赠送 %s", logger.LogQuota(bonusQuota))
	}
	RecordLog(userId, LogTypeTopup, content)
	if rebateQuota > 0 {
		RecordLog(
			rebateInviterId,
			LogTypeTopup,
			fmt.Sprintf("邀请返利 %s（被邀请人兑换码ID %d）", logger.LogQuota(rebateQuota), redemption.Id),
		)
	}
	return totalQuota, nil
}

func (redemption *Redemption) Insert() error {
	var err error
	err = DB.Create(redemption).Error
	return err
}

func (redemption *Redemption) SelectUpdate() error {
	// This can update zero values
	return DB.Model(redemption).Select("redeemed_time", "status").Updates(redemption).Error
}

// Update Make sure your token's fields is completed, because this will update non-zero values
func (redemption *Redemption) Update() error {
	var err error
	err = DB.Model(redemption).Select("name", "status", "quota", "redeemed_time", "expired_time").Updates(redemption).Error
	return err
}

func (redemption *Redemption) Delete() error {
	var err error
	err = DB.Delete(redemption).Error
	return err
}

func DeleteRedemptionById(id int) (err error) {
	if id == 0 {
		return errors.New("id 为空！")
	}
	redemption := Redemption{Id: id}
	err = DB.Where(redemption).First(&redemption).Error
	if err != nil {
		return err
	}
	return redemption.Delete()
}

func DeleteInvalidRedemptions() (int64, error) {
	now := common.GetTimestamp()
	result := DB.Where("status IN ? OR (status = ? AND expired_time != 0 AND expired_time < ?)", []int{common.RedemptionCodeStatusUsed, common.RedemptionCodeStatusDisabled}, common.RedemptionCodeStatusEnabled, now).Delete(&Redemption{})
	return result.RowsAffected, result.Error
}
