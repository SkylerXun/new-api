package model

import (
	"errors"
	"fmt"
	"math"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

type UserMonthlyBillingProgress struct {
	UserId         int   `json:"user_id" gorm:"primaryKey;autoIncrement:false;uniqueIndex:ux_monthly_billing_progress,priority:1"`
	MonthStart     int64 `json:"month_start" gorm:"primaryKey;autoIncrement:false;uniqueIndex:ux_monthly_billing_progress,priority:2"`
	SpentMicroUSD  int64 `json:"spent_micro_usd"`
	BackfillCutoff int64 `json:"backfill_cutoff" gorm:"not null;default:0"`
	CreatedAt      int64 `json:"created_at"`
	UpdatedAt      int64 `json:"updated_at"`
}

type UserMonthlyBillingSettlement struct {
	SettlementKey          string `json:"settlement_key" gorm:"type:varchar(128);primaryKey"`
	UserId                 int    `json:"user_id" gorm:"not null;index"`
	MonthStart             int64  `json:"month_start" gorm:"not null;index"`
	NormalQuota            int    `json:"normal_quota" gorm:"not null"`
	ChargedQuota           int    `json:"charged_quota" gorm:"not null"`
	ProgressBeforeMicroUSD int64  `json:"progress_before_micro_usd" gorm:"not null"`
	ProgressAfterMicroUSD  int64  `json:"progress_after_micro_usd" gorm:"not null"`
	Refunded               bool   `json:"refunded" gorm:"not null"`
	CreatedAt              int64  `json:"created_at"`
	UpdatedAt              int64  `json:"updated_at"`
}

func (UserMonthlyBillingSettlement) TableName() string { return "user_monthly_billing_settlements" }

func (s *UserMonthlyBillingSettlement) BeforeCreate(_ *gorm.DB) error {
	now := common.GetTimestamp()
	if s.CreatedAt == 0 {
		s.CreatedAt = now
	}
	if s.UpdatedAt == 0 {
		s.UpdatedAt = now
	}
	return nil
}

func (UserMonthlyBillingProgress) TableName() string { return "user_monthly_billing_progress" }

func (p *UserMonthlyBillingProgress) BeforeCreate(_ *gorm.DB) error {
	now := common.GetTimestamp()
	if p.CreatedAt == 0 {
		p.CreatedAt = now
	}
	if p.UpdatedAt == 0 {
		p.UpdatedAt = now
	}
	return nil
}

func GetUserMonthlyBillingProgress(userID int, monthStart int64) (int64, error) {
	var progress UserMonthlyBillingProgress
	err := DB.Where("user_id = ? AND month_start = ?", userID, monthStart).First(&progress).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return progress.SpentMicroUSD, nil
}

func GetUserMonthlyBillingProgresses(userIDs []int, monthStart int64) (map[int]int64, error) {
	result := make(map[int]int64, len(userIDs))
	if len(userIDs) == 0 {
		return result, nil
	}
	var rows []UserMonthlyBillingProgress
	if err := DB.Where("month_start = ? AND user_id IN ?", monthStart, userIDs).Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		result[row.UserId] = row.SpentMicroUSD
	}
	return result, nil
}

type monthlyBillingBackfillTotal struct {
	UserId   int
	NetQuota int64
}

// EnsureUserMonthlyBillingBackfill adds the activation month's historical
// successful deductions exactly once per user. It only advances progress;
// balances, used quota, tokens, and historical logs are left untouched.
func EnsureUserMonthlyBillingBackfill(userIDs []int, monthStart int64, cutoff int64) error {
	if len(userIDs) == 0 || cutoff <= monthStart {
		return nil
	}
	uniqueUserIDs := make([]int, 0, len(userIDs))
	seen := make(map[int]struct{}, len(userIDs))
	for _, userID := range userIDs {
		if userID <= 0 {
			continue
		}
		if _, exists := seen[userID]; exists {
			continue
		}
		seen[userID] = struct{}{}
		uniqueUserIDs = append(uniqueUserIDs, userID)
	}
	if len(uniqueUserIDs) == 0 {
		return nil
	}
	var completedUserIDs []int
	if err := DB.Model(&UserMonthlyBillingProgress{}).
		Where("month_start = ? AND backfill_cutoff = ? AND user_id IN ?", monthStart, cutoff, uniqueUserIDs).
		Pluck("user_id", &completedUserIDs).Error; err != nil {
		return fmt.Errorf("query monthly billing backfill markers: %w", err)
	}
	completed := make(map[int]struct{}, len(completedUserIDs))
	for _, userID := range completedUserIDs {
		completed[userID] = struct{}{}
	}
	pendingUserIDs := make([]int, 0, len(uniqueUserIDs)-len(completedUserIDs))
	for _, userID := range uniqueUserIDs {
		if _, exists := completed[userID]; !exists {
			pendingUserIDs = append(pendingUserIDs, userID)
		}
	}
	if len(pendingUserIDs) == 0 {
		return nil
	}

	var totals []monthlyBillingBackfillTotal
	err := LOG_DB.Model(&Log{}).
		Select("user_id, SUM(CASE WHEN type = ? THEN quota WHEN type = ? THEN -quota ELSE 0 END) AS net_quota", LogTypeConsume, LogTypeRefund).
		Where("user_id IN ? AND created_at >= ? AND created_at < ? AND type IN ?", pendingUserIDs, monthStart, cutoff, []int{LogTypeConsume, LogTypeRefund}).
		Group("user_id").Scan(&totals).Error
	if err != nil {
		return fmt.Errorf("aggregate monthly billing backfill: %w", err)
	}
	netQuotaByUser := make(map[int]int64, len(totals))
	for _, total := range totals {
		if total.NetQuota > 0 {
			netQuotaByUser[total.UserId] = total.NetQuota
		}
	}
	if common.QuotaPerUnit <= 0 || math.IsNaN(common.QuotaPerUnit) || math.IsInf(common.QuotaPerUnit, 0) {
		return errors.New("invalid quota per unit for monthly billing backfill")
	}

	for _, userID := range pendingUserIDs {
		seedMicroUSD := int64(math.Round(float64(netQuotaByUser[userID]) / common.QuotaPerUnit * 1_000_000))
		if seedMicroUSD < 0 {
			seedMicroUSD = 0
		}
		if err := ensureUserMonthlyBillingBackfill(userID, monthStart, cutoff, seedMicroUSD); err != nil {
			return err
		}
	}
	return nil
}

func ensureUserMonthlyBillingBackfill(userID int, monthStart int64, cutoff int64, seedMicroUSD int64) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		var user User
		if err := lockForUpdate(tx).Select("id").Where("id = ?", userID).First(&user).Error; err != nil {
			return err
		}
		var progress UserMonthlyBillingProgress
		lookupErr := tx.Where("user_id = ? AND month_start = ?", userID, monthStart).First(&progress).Error
		if errors.Is(lookupErr, gorm.ErrRecordNotFound) {
			progress = UserMonthlyBillingProgress{UserId: userID, MonthStart: monthStart}
		} else if lookupErr != nil {
			return lookupErr
		}
		if progress.BackfillCutoff == cutoff {
			return nil
		}
		if progress.BackfillCutoff != 0 {
			return fmt.Errorf("monthly billing backfill cutoff conflict for user %d", userID)
		}
		if seedMicroUSD > math.MaxInt64-progress.SpentMicroUSD {
			return fmt.Errorf("monthly billing backfill overflow for user %d", userID)
		}
		progress.SpentMicroUSD += seedMicroUSD
		progress.BackfillCutoff = cutoff
		progress.UpdatedAt = common.GetTimestamp()
		if progress.CreatedAt == 0 {
			return tx.Create(&progress).Error
		}
		return tx.Save(&progress).Error
	})
}

func AdvanceUserMonthlyBillingProgress(userID int, monthStart int64, deltaMicroUSD int64) (before int64, after int64, err error) {
	if userID <= 0 || monthStart <= 0 {
		return 0, 0, errors.New("invalid monthly billing progress key")
	}
	if deltaMicroUSD <= 0 {
		return 0, 0, errors.New("monthly billing progress delta must be positive")
	}
	err = DB.Transaction(func(tx *gorm.DB) error {
		var user User
		if err := lockForUpdate(tx).Select("id").Where("id = ?", userID).First(&user).Error; err != nil {
			return err
		}
		var progress UserMonthlyBillingProgress
		lookupErr := tx.Where("user_id = ? AND month_start = ?", userID, monthStart).First(&progress).Error
		if errors.Is(lookupErr, gorm.ErrRecordNotFound) {
			progress = UserMonthlyBillingProgress{UserId: userID, MonthStart: monthStart}
			if err := tx.Create(&progress).Error; err != nil {
				return err
			}
		} else if lookupErr != nil {
			return lookupErr
		}
		before = progress.SpentMicroUSD
		if before < 0 || deltaMicroUSD > math.MaxInt64-before {
			return fmt.Errorf("monthly billing progress overflow for user %d", userID)
		}
		after = before + deltaMicroUSD
		progress.SpentMicroUSD = after
		progress.UpdatedAt = common.GetTimestamp()
		return tx.Save(&progress).Error
	})
	return before, after, err
}

// UpdateUserMonthlyBillingProgress locks one user's month row and lets the
// caller derive the new value from the locked balance. This keeps tier
// crossing calculations serializable under concurrent requests.
func UpdateUserMonthlyBillingProgress(userID int, monthStart int64, update func(int64) (int64, error)) (before int64, after int64, err error) {
	if userID <= 0 || monthStart <= 0 || update == nil {
		return 0, 0, errors.New("invalid monthly billing progress update")
	}
	err = DB.Transaction(func(tx *gorm.DB) error {
		var user User
		if err := lockForUpdate(tx).Select("id").Where("id = ?", userID).First(&user).Error; err != nil {
			return err
		}
		var progress UserMonthlyBillingProgress
		lookupErr := tx.Where("user_id = ? AND month_start = ?", userID, monthStart).First(&progress).Error
		if errors.Is(lookupErr, gorm.ErrRecordNotFound) {
			progress = UserMonthlyBillingProgress{UserId: userID, MonthStart: monthStart}
			if err := tx.Create(&progress).Error; err != nil {
				return err
			}
		} else if lookupErr != nil {
			return lookupErr
		}
		before = progress.SpentMicroUSD
		after, err = update(before)
		if err != nil {
			return err
		}
		if after < 0 || after < before {
			return errors.New("monthly billing progress cannot decrease")
		}
		progress.SpentMicroUSD = after
		progress.UpdatedAt = common.GetTimestamp()
		return tx.Save(&progress).Error
	})
	return before, after, err
}

func SettleUserMonthlyBillingProgress(userID int, monthStart int64, settlementKey string, normalQuota int, update func(int64) (int64, int, error)) (before int64, after int64, chargedQuota int, settledMonthStart int64, err error) {
	if userID <= 0 || monthStart <= 0 || settlementKey == "" || normalQuota < 0 || update == nil {
		return 0, 0, 0, 0, errors.New("invalid monthly billing settlement")
	}
	settledMonthStart = monthStart
	err = DB.Transaction(func(tx *gorm.DB) error {
		var user User
		if lockErr := lockForUpdate(tx).Select("id").Where("id = ?", userID).First(&user).Error; lockErr != nil {
			return lockErr
		}
		var existing UserMonthlyBillingSettlement
		lookupErr := tx.Where("settlement_key = ?", settlementKey).First(&existing).Error
		if lookupErr == nil {
			if existing.UserId != userID || existing.NormalQuota != normalQuota {
				return errors.New("monthly billing settlement key conflict")
			}
			if existing.Refunded {
				return errors.New("monthly billing settlement was already refunded")
			}
			before, after, chargedQuota = existing.ProgressBeforeMicroUSD, existing.ProgressAfterMicroUSD, existing.ChargedQuota
			settledMonthStart = existing.MonthStart
			return nil
		}
		if !errors.Is(lookupErr, gorm.ErrRecordNotFound) {
			return lookupErr
		}
		var progress UserMonthlyBillingProgress
		progressErr := tx.Where("user_id = ? AND month_start = ?", userID, monthStart).First(&progress).Error
		if errors.Is(progressErr, gorm.ErrRecordNotFound) {
			progress = UserMonthlyBillingProgress{UserId: userID, MonthStart: monthStart}
			if createErr := tx.Create(&progress).Error; createErr != nil {
				return createErr
			}
		} else if progressErr != nil {
			return progressErr
		}
		before = progress.SpentMicroUSD
		var updateErr error
		after, chargedQuota, updateErr = update(before)
		if updateErr != nil {
			return updateErr
		}
		if after < before || chargedQuota < 0 {
			return errors.New("invalid monthly billing settlement result")
		}
		progress.SpentMicroUSD = after
		progress.UpdatedAt = common.GetTimestamp()
		if saveErr := tx.Save(&progress).Error; saveErr != nil {
			return saveErr
		}
		return tx.Create(&UserMonthlyBillingSettlement{SettlementKey: settlementKey, UserId: userID, MonthStart: monthStart, NormalQuota: normalQuota, ChargedQuota: chargedQuota, ProgressBeforeMicroUSD: before, ProgressAfterMicroUSD: after}).Error
	})
	return before, after, chargedQuota, settledMonthStart, err
}

func RefundUserMonthlyBillingSettlement(settlementKey string) error {
	if settlementKey == "" {
		return errors.New("monthly billing settlement key is required")
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		var settlement UserMonthlyBillingSettlement
		if err := tx.Where("settlement_key = ?", settlementKey).First(&settlement).Error; err != nil {
			return err
		}
		var user User
		if err := lockForUpdate(tx).Select("id").Where("id = ?", settlement.UserId).First(&user).Error; err != nil {
			return err
		}
		if err := lockForUpdate(tx).Where("settlement_key = ?", settlementKey).First(&settlement).Error; err != nil {
			return err
		}
		if settlement.Refunded {
			return nil
		}
		var progress UserMonthlyBillingProgress
		if err := tx.Where("user_id = ? AND month_start = ?", settlement.UserId, settlement.MonthStart).First(&progress).Error; err != nil {
			return err
		}
		delta := settlement.ProgressAfterMicroUSD - settlement.ProgressBeforeMicroUSD
		if delta >= progress.SpentMicroUSD {
			progress.SpentMicroUSD = 0
		} else {
			progress.SpentMicroUSD -= delta
		}
		progress.UpdatedAt = common.GetTimestamp()
		if err := tx.Save(&progress).Error; err != nil {
			return err
		}
		settlement.Refunded = true
		settlement.UpdatedAt = common.GetTimestamp()
		return tx.Save(&settlement).Error
	})
}

func RevertUserMonthlyBillingProgress(userID int, monthStart int64, deltaMicroUSD int64) error {
	if userID <= 0 || monthStart <= 0 || deltaMicroUSD <= 0 {
		return errors.New("invalid monthly billing progress refund")
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		var user User
		if err := lockForUpdate(tx).Select("id").Where("id = ?", userID).First(&user).Error; err != nil {
			return err
		}
		var progress UserMonthlyBillingProgress
		if err := tx.Where("user_id = ? AND month_start = ?", userID, monthStart).First(&progress).Error; err != nil {
			return err
		}
		if deltaMicroUSD >= progress.SpentMicroUSD {
			progress.SpentMicroUSD = 0
		} else {
			progress.SpentMicroUSD -= deltaMicroUSD
		}
		progress.UpdatedAt = common.GetTimestamp()
		return tx.Save(&progress).Error
	})
}
