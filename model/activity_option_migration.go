package model

import (
	"errors"
	"strconv"
	"strings"

	"gorm.io/gorm"
)

const (
	legacyNewUserRedeemBonusWindowHoursOption = "activity_setting.new_user_redeem_bonus_window_hours"
	newUserRedeemBonusWindowDaysOption        = "activity_setting.new_user_redeem_bonus_window_days"
)

// MigrateLegacyActivityOptions converts the short-lived hours setting into
// the permanent activity's integer-day window. The old option is removed only
// after the new one is persisted, so a failed upgrade leaves the prior value
// intact for the next startup.
func MigrateLegacyActivityOptions() error {
	if DB == nil {
		return errors.New("database is not initialized")
	}

	return DB.Transaction(func(tx *gorm.DB) error {
		var legacy Option
		if err := tx.Where(&Option{Key: legacyNewUserRedeemBonusWindowHoursOption}).First(&legacy).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return err
		}

		var current Option
		if err := tx.Where(&Option{Key: newUserRedeemBonusWindowDaysOption}).First(&current).Error; err == nil {
			return tx.Delete(&legacy).Error
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		hours, err := strconv.Atoi(strings.TrimSpace(legacy.Value))
		if err != nil || hours < 1 {
			return nil
		}
		days := (hours + 23) / 24
		if days > 3650 {
			days = 3650
		}
		if err := tx.Create(&Option{
			Key:   newUserRedeemBonusWindowDaysOption,
			Value: strconv.Itoa(days),
		}).Error; err != nil {
			return err
		}
		return tx.Delete(&legacy).Error
	})
}
