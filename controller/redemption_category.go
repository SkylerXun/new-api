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
