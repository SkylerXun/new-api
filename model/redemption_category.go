package model

import (
	"errors"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

type RedemptionCategory struct {
	ID         int    `json:"id"`
	Name       string `json:"name" gorm:"type:varchar(80);uniqueIndex"`
	PriceCents int64  `json:"price_cents" gorm:"not null"`
	Enabled    bool   `json:"enabled"`
	CreatedAt  int64  `json:"created_at" gorm:"bigint"`
	UpdatedAt  int64  `json:"updated_at" gorm:"bigint"`
}

type RedemptionPricingAudit struct {
	ID               int64  `json:"id"`
	RedemptionID     int    `json:"redemption_id" gorm:"uniqueIndex"`
	CategoryID       int    `json:"category_id"`
	CategoryName     string `json:"category_name" gorm:"type:varchar(80)"`
	PriceCents       int64  `json:"price_cents"`
	PreviousCategory string `json:"previous_category" gorm:"type:varchar(80)"`
	PreviousCents    int64  `json:"previous_cents"`
	AssignedBy       int    `json:"assigned_by"`
	AssignedAt       int64  `json:"assigned_at" gorm:"bigint"`
}

func (category *RedemptionCategory) normalize() error {
	category.Name = strings.TrimSpace(category.Name)
	if category.Name == "" || len([]rune(category.Name)) > 80 {
		return errors.New("兑换码类别名称长度必须为 1-80 个字符")
	}
	if category.PriceCents < 0 {
		return errors.New("兑换码类别价格不能为负数")
	}
	return nil
}

func ListRedemptionCategories(includeDisabled bool) ([]RedemptionCategory, error) {
	query := DB.Model(&RedemptionCategory{})
	if !includeDisabled {
		query = query.Where("enabled = ?", true)
	}
	var categories []RedemptionCategory
	err := query.Order("enabled desc, id desc").Find(&categories).Error
	return categories, err
}

func CreateRedemptionCategory(category *RedemptionCategory) error {
	if category == nil {
		return errors.New("兑换码类别不能为空")
	}
	if err := category.normalize(); err != nil {
		return err
	}
	category.Enabled = true
	now := common.GetTimestamp()
	category.CreatedAt = now
	category.UpdatedAt = now
	return DB.Create(category).Error
}

func UpdateRedemptionCategory(category *RedemptionCategory) error {
	if category == nil || category.ID <= 0 {
		return errors.New("无效的兑换码类别")
	}
	if err := category.normalize(); err != nil {
		return err
	}
	return DB.Model(&RedemptionCategory{}).Where("id = ?", category.ID).Updates(map[string]any{
		"name":        category.Name,
		"price_cents": category.PriceCents,
		"updated_at":  common.GetTimestamp(),
	}).Error
}

func SetRedemptionCategoryStatus(id int, enabled bool) error {
	if id <= 0 {
		return errors.New("无效的兑换码类别")
	}
	result := DB.Model(&RedemptionCategory{}).Where("id = ?", id).Updates(map[string]any{
		"enabled":    enabled,
		"updated_at": common.GetTimestamp(),
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func GetEnabledRedemptionCategory(id int) (*RedemptionCategory, error) {
	var category RedemptionCategory
	err := DB.Where("id = ? AND enabled = ?", id, true).First(&category).Error
	return &category, err
}

// AssignRedemptionCategory snapshots the current category name and price onto
// legacy redemption rows. A row can be priced only once, including used rows.
func AssignRedemptionCategory(ids []int, categoryID int, operatorID int) (int64, error) {
	if len(ids) == 0 {
		return 0, errors.New("请选择待补价兑换码")
	}
	category, err := GetEnabledRedemptionCategory(categoryID)
	if err != nil {
		return 0, err
	}
	now := common.GetTimestamp()
	var assigned int64
	err = DB.Transaction(func(tx *gorm.DB) error {
		for _, id := range ids {
			if id <= 0 {
				return errors.New("兑换码 ID 无效")
			}
			var redemption Redemption
			if err := lockForUpdate(tx).Unscoped().Where("id = ?", id).First(&redemption).Error; err != nil {
				return err
			}
			// CategoryPricedAt is the durable completion marker. Some older rows
			// may already contain a category_id but lack the pricing timestamp;
			// those rows still need to be repairable.
			if redemption.CategoryPricedAt != 0 && redemption.Status == common.RedemptionCodeStatusUsed {
				return errors.New("兑换码已完成补价，不能重复修改")
			}
			// Older schemas may contain NULL in category_priced_at. Treat NULL
			// exactly like zero so those rows can be repaired as well.
			result := tx.Unscoped().Model(&Redemption{}).
				Where("id = ? AND (category_priced_at = ? OR category_priced_at IS NULL OR status != ?)", id, 0, common.RedemptionCodeStatusUsed).
				Updates(map[string]any{
					"category_id":            category.ID,
					"category_name_snapshot": category.Name,
					"category_price_cents":   category.PriceCents,
					"category_priced_at":     now,
					"category_priced_by":     operatorID,
				})
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected != 1 {
				return errors.New("兑换码已被其他管理员补价")
			}
			audit := RedemptionPricingAudit{
				RedemptionID:     id,
				CategoryID:       category.ID,
				CategoryName:     category.Name,
				PriceCents:       category.PriceCents,
				PreviousCategory: redemption.CategoryNameSnapshot,
				PreviousCents:    redemption.CategoryPriceCents,
				AssignedBy:       operatorID,
				AssignedAt:       now,
			}
			var existingAudit RedemptionPricingAudit
			if err := tx.Where("redemption_id = ?", id).First(&existingAudit).Error; err == nil {
				if err := tx.Model(&existingAudit).Updates(map[string]any{
					"category_id": category.ID, "category_name": category.Name,
					"price_cents": category.PriceCents, "previous_category": redemption.CategoryNameSnapshot,
					"previous_cents": redemption.CategoryPriceCents, "assigned_by": operatorID, "assigned_at": now,
				}).Error; err != nil {
					return err
				}
			} else if errors.Is(err, gorm.ErrRecordNotFound) {
				if err := tx.Create(&audit).Error; err != nil {
					return err
				}
			} else {
				return err
			}
			assigned++
		}
		return nil
	})
	return assigned, err
}
