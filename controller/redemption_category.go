package controller

import (
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

func ListRedemptionCategories(c *gin.Context) {
	categories, err := model.ListRedemptionCategories(c.Query("include_disabled") == "true")
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, categories)
}

// CreateDecreasingRedemption creates a code owned by the optional plugin.
// It never writes to the legacy redemption table.
func CreateDecreasingRedemption(c *gin.Context) {
	var req struct { Name string `json:"name"`; TotalQuota int `json:"total_quota"`; MonthlyLimitQuota int `json:"monthly_limit_quota"` }
	if err := c.ShouldBindJSON(&req); err != nil || req.TotalQuota <= 0 || req.MonthlyLimitQuota <= 0 { common.ApiErrorMsg(c, "递减兑换码参数无效"); return }
	// The database column deliberately matches legacy redemption keys (CHAR(32)).
	key := common.GetUUID()
	code := model.DecreasingRedemption{Key: key, Name: req.Name, TotalQuota: req.TotalQuota, RemainingQuota: req.TotalQuota, MonthlyLimitQuota: req.MonthlyLimitQuota, Enabled: true, CreatedAt: common.GetTimestamp()}
	if err := model.DB.Create(&code).Error; err != nil { common.ApiError(c, err); return }
	common.ApiSuccess(c, code)
}

func ListDecreasingRedemptions(c *gin.Context) {
	var codes []model.DecreasingRedemption
	if err := model.DB.Order("id desc").Find(&codes).Error; err != nil { common.ApiError(c, err); return }
	common.ApiSuccess(c, codes)
}

func UpdateDecreasingRedemptionStatus(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id")); if err != nil { common.ApiError(c, err); return }
	var request struct { Enabled *bool `json:"enabled"` }
	if err := c.ShouldBindJSON(&request); err != nil || request.Enabled == nil { common.ApiErrorMsg(c, "enabled 必填"); return }
	result := model.DB.Model(&model.DecreasingRedemption{}).Where("id = ?", id).Update("enabled", *request.Enabled)
	if result.Error != nil { common.ApiError(c, result.Error); return }
	common.ApiSuccess(c, nil)
}

func DeleteDecreasingRedemption(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id")); if err != nil { common.ApiError(c, err); return }
	if err := model.DB.Delete(&model.DecreasingRedemption{}, id).Error; err != nil { common.ApiError(c, err); return }
	common.ApiSuccess(c, nil)
}

func CreateRedemptionCategory(c *gin.Context) {
	var category model.RedemptionCategory
	if err := c.ShouldBindJSON(&category); err != nil {
		common.ApiError(c, err)
		return
	}
	if err := model.CreateRedemptionCategory(&category); err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "redemption.category.create", map[string]any{
		"category_id": category.ID,
		"name":        category.Name,
		"price_cents": category.PriceCents,
	})
	common.ApiSuccess(c, category)
}

func UpdateRedemptionCategory(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	var category model.RedemptionCategory
	if err := c.ShouldBindJSON(&category); err != nil {
		common.ApiError(c, err)
		return
	}
	category.ID = id
	if err := model.UpdateRedemptionCategory(&category); err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "redemption.category.update", map[string]any{
		"category_id": id,
		"name":        category.Name,
		"price_cents": category.PriceCents,
	})
	common.ApiSuccess(c, category)
}

func UpdateRedemptionCategoryStatus(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	var request struct {
		Enabled *bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&request); err != nil || request.Enabled == nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "enabled 必填"})
		return
	}
	if err := model.SetRedemptionCategoryStatus(id, *request.Enabled); err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "redemption.category.status", map[string]any{
		"category_id": id,
		"enabled":     *request.Enabled,
	})
	common.ApiSuccess(c, nil)
}

func AssignRedemptionCategories(c *gin.Context) {
	var request struct {
		RedemptionIDs []int `json:"redemption_ids"`
		CategoryID    int   `json:"category_id"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		common.ApiError(c, err)
		return
	}
	assigned, err := model.AssignRedemptionCategory(request.RedemptionIDs, request.CategoryID, c.GetInt("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "redemption.category.assign", map[string]any{
		"category_id": request.CategoryID,
		"assigned":    assigned,
	})
	common.ApiSuccess(c, gin.H{"assigned": assigned})
}
