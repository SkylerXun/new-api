package model

import (
	"errors"
	"fmt"
	"math"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

// UserBillingCurveProgress is deliberately independent from wallet quota.
// Manual balance adjustments, refunds, and top-ups therefore never change a
// user's billing-curve position.
type UserBillingCurveProgress struct {
	UserId                 int   `json:"user_id" gorm:"primaryKey;autoIncrement:false"`
	TotalBaseUsageMicroUSD int64 `json:"total_base_usage_micro_usd;not null"`
	CreatedAt              int64 `json:"created_at;not null;index"`
	UpdatedAt              int64 `json:"updated_at;not null;index"`
}

func (UserBillingCurveProgress) TableName() string {
	return "user_billing_curve_progress"
}

func (progress *UserBillingCurveProgress) BeforeCreate(_ *gorm.DB) error {
	now := common.GetTimestamp()
	if progress.CreatedAt == 0 {
		progress.CreatedAt = now
	}
	if progress.UpdatedAt == 0 {
		progress.UpdatedAt = now
	}
	return nil
}

// GetUserBillingCurveProgresses returns the persisted progress for a page of
// users. Users without a row have not accumulated any base usage yet.
func GetUserBillingCurveProgresses(userIDs []int) (map[int]int64, error) {
	result := make(map[int]int64, len(userIDs))
	if len(userIDs) == 0 {
		return result, nil
	}

	var progresses []UserBillingCurveProgress
	if err := DB.Select("user_id", "total_base_usage_micro_usd").Where("user_id IN ?", userIDs).Find(&progresses).Error; err != nil {
		return nil, err
	}
	for _, progress := range progresses {
		result[progress.UserId] = progress.TotalBaseUsageMicroUSD
	}
	return result, nil
}

// AdvanceUserBillingCurveProgress atomically allocates one user's next curve
// interval. Locking the user row also serializes creation of the progress row
// on SQLite, MySQL, and PostgreSQL.
func AdvanceUserBillingCurveProgress(userID int, deltaMicroUSD int64) (before int64, after int64, err error) {
	if userID <= 0 {
		return 0, 0, errors.New("invalid user id for billing curve progress")
	}
	if deltaMicroUSD <= 0 {
		return 0, 0, errors.New("billing curve progress delta must be positive")
	}

	err = DB.Transaction(func(tx *gorm.DB) error {
		var user User
		if lockErr := lockForUpdate(tx).Select("id").Where("id = ?", userID).First(&user).Error; lockErr != nil {
			return lockErr
		}

		var progress UserBillingCurveProgress
		lookupErr := tx.Where("user_id = ?", userID).First(&progress).Error
		if errors.Is(lookupErr, gorm.ErrRecordNotFound) {
			progress = UserBillingCurveProgress{UserId: userID}
			if createErr := tx.Create(&progress).Error; createErr != nil {
				return createErr
			}
		} else if lookupErr != nil {
			return lookupErr
		}

		before = progress.TotalBaseUsageMicroUSD
		if before < 0 {
			return fmt.Errorf("billing curve progress is negative for user %d", userID)
		}
		if deltaMicroUSD > math.MaxInt64-before {
			return fmt.Errorf("billing curve progress overflow for user %d", userID)
		}
		after = before + deltaMicroUSD
		progress.TotalBaseUsageMicroUSD = after
		progress.UpdatedAt = common.GetTimestamp()
		return tx.Save(&progress).Error
	})
	return before, after, err
}
