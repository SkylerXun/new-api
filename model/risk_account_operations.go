package model

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm"
)

func recordRiskAccountActionTx(tx *gorm.DB, userID, mainUserID int, clusterID, action, rule string, createdBy int) error {
	actionKey := common.GenerateHMAC(strings.Join([]string{
		RiskAccountPolicyVersion, action, clusterID, fmt.Sprintf("%d", userID), fmt.Sprintf("%d", mainUserID),
	}, ":"))
	var count int64
	if err := tx.Model(&RiskAccountAction{}).Where("action_key = ?", actionKey).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	return tx.Create(&RiskAccountAction{
		ActionKey: actionKey, UserID: userID, MainUserID: mainUserID,
		ClusterID: clusterID, Action: action, Rule: rule,
		PolicyVersion: RiskAccountPolicyVersion, CreatedBy: createdBy, CreatedAt: time.Now().Unix(),
	}).Error
}

func recordRiskRewardHoldTx(tx *gorm.DB, user User, clusterID string) error {
	var promotionalQuota int64
	if err := tx.Model(&ActivityGrant{}).
		Where("user_id = ? AND activity_key = ?", user.Id, ActivityKeyNewUserRedeemBonus).
		Select("COALESCE(SUM(quota), 0)").Scan(&promotionalQuota).Error; err != nil {
		return err
	}
	if promotionalQuota <= 0 || user.Quota <= 0 {
		return nil
	}
	frozenQuota := promotionalQuota
	if int64(user.Quota) < frozenQuota {
		frozenQuota = int64(user.Quota)
	}
	var existing RiskRewardHold
	err := tx.Where("user_id = ? AND cluster_id = ?", user.Id, clusterID).First(&existing).Error
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		return tx.Create(&RiskRewardHold{
			UserID: user.Id, ClusterID: clusterID, FrozenQuota: int(frozenQuota),
			Status: RiskRewardHoldStatusFrozen, PolicyVersion: RiskAccountPolicyVersion,
			CreatedAt: time.Now().Unix(),
		}).Error
	case err != nil:
		return err
	default:
		return tx.Model(&RiskRewardHold{}).Where("id = ?", existing.ID).Updates(map[string]any{
			"frozen_quota": int(frozenQuota), "status": RiskRewardHoldStatusFrozen,
			"resolved_at": 0,
		}).Error
	}
}

func detachRiskAccountTx(tx *gorm.DB, userID int, reenable bool, createdBy int, reason string) ([]int, error) {
	var link RiskAccountLink
	if err := tx.Where("user_id = ?", userID).First(&link).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if err := tx.Where("user_id = ?", userID).Delete(&RiskAccountObservation{}).Error; err != nil {
				return nil, err
			}
			if reenable {
				if err := tx.Model(&User{}).Where("id = ? AND role < ?", userID, common.RoleAdminUser).Update("status", common.UserStatusEnabled).Error; err != nil {
					return nil, err
				}
			}
			return []int{userID}, nil
		}
		return nil, err
	}
	targetIDs := []int{userID}
	if link.Relation == "main" || link.MainUserID == userID {
		if err := tx.Model(&RiskAccountLink{}).Where("main_user_id = ?", link.MainUserID).Pluck("user_id", &targetIDs).Error; err != nil {
			return nil, err
		}
		if err := tx.Where("main_user_id = ?", link.MainUserID).Delete(&RiskAccountLink{}).Error; err != nil {
			return nil, err
		}
	} else {
		if err := tx.Where("user_id = ?", userID).Delete(&RiskAccountLink{}).Error; err != nil {
			return nil, err
		}
		var remaining []RiskAccountLink
		if err := tx.Where("main_user_id = ?", link.MainUserID).Find(&remaining).Error; err != nil {
			return nil, err
		}
		if len(remaining) <= 1 {
			if err := tx.Where("main_user_id = ?", link.MainUserID).Delete(&RiskAccountLink{}).Error; err != nil {
				return nil, err
			}
		}
	}
	if err := tx.Where("user_id IN ?", targetIDs).Delete(&RiskAccountObservation{}).Error; err != nil {
		return nil, err
	}
	if reenable {
		if err := tx.Model(&User{}).Where("id IN ? AND role < ?", targetIDs, common.RoleAdminUser).Update("status", common.UserStatusEnabled).Error; err != nil {
			return nil, err
		}
	}
	if err := tx.Model(&RiskRewardHold{}).
		Where("user_id IN ? AND status = ?", targetIDs, RiskRewardHoldStatusFrozen).
		Updates(map[string]any{"status": RiskRewardHoldStatusReleased, "resolved_at": time.Now().Unix()}).Error; err != nil {
		return nil, err
	}
	for _, id := range targetIDs {
		if err := recordRiskAccountActionTx(tx, id, link.MainUserID, link.ClusterID, "unlink_restore", strings.TrimSpace(reason), createdBy); err != nil {
			return nil, err
		}
	}
	return targetIDs, nil
}

func DetachRiskAccount(userID int, reenable bool, createdBy int, reason string) ([]int, error) {
	if userID <= 0 || createdBy <= 0 {
		return nil, errors.New("invalid risk account detach request")
	}
	var affected []int
	err := DB.Transaction(func(tx *gorm.DB) error {
		var err error
		affected, err = detachRiskAccountTx(tx, userID, reenable, createdBy, reason)
		return err
	})
	if err != nil {
		return nil, err
	}
	for _, id := range affected {
		_ = PublishUserAuthCache(id)
	}
	return affected, nil
}

func UpsertRiskExemptionAndDetach(row *RiskExemption) ([]int, error) {
	if row == nil || row.UserID <= 0 || row.CreatedBy <= 0 {
		return nil, errors.New("invalid risk exemption")
	}
	if row.CreatedAt == 0 {
		row.CreatedAt = time.Now().Unix()
	}
	row.Reason = strings.TrimSpace(row.Reason)
	row.Enabled = true
	var affected []int
	err := DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(row).Error; err != nil {
			return err
		}
		var err error
		affected, err = detachRiskAccountTx(tx, row.UserID, true, row.CreatedBy, "risk exemption: "+row.Reason)
		return err
	})
	if err != nil {
		return nil, err
	}
	for _, id := range affected {
		_ = PublishUserAuthCache(id)
	}
	return affected, nil
}
