package controller

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type statementGenerateRequest struct {
	UserID         int    `json:"user_id"`
	Month          string `json:"month"`
	BillingTitle   string `json:"billing_title"`
	BillingAddress string `json:"billing_address"`
}

func GenerateSelfCurrentStatement(c *gin.Context) {
	var request statementGenerateRequest
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&request); err != nil {
			common.ApiError(c, err)
			return
		}
	}
	statement, err := model.GenerateConsumptionStatement(model.StatementGenerateInput{
		UserID: c.GetInt("id"), Source: model.StatementSourceUserExport, GeneratedBy: c.GetInt("id"),
		BillingTitle: request.BillingTitle, BillingAddress: request.BillingAddress, Now: time.Now(),
	})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, statement)
}

func GetSelfPreviousStatement(c *gin.Context) {
	statement, err := model.GetPreviousMonthSystemStatement(c.GetInt("id"), time.Now())
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "上月账单尚未生成，请稍后再试"})
			return
		}
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, statement)
}

func getAccessibleStatement(c *gin.Context) (*model.ConsumptionStatement, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		common.ApiErrorMsg(c, "账单 ID 无效")
		return nil, false
	}
	statement, err := model.GetConsumptionStatement(id)
	if err != nil {
		common.ApiError(c, err)
		return nil, false
	}
	if !model.CanAccessConsumptionStatement(statement, c.GetInt("id"), c.GetInt("role"), time.Now()) {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "无权访问该账单"})
		return nil, false
	}
	return statement, true
}

func GetStatement(c *gin.Context) {
	statement, ok := getAccessibleStatement(c)
	if !ok {
		return
	}
	common.ApiSuccess(c, statement)
}

func DownloadStatementPDF(c *gin.Context) {
	statement, ok := getAccessibleStatement(c)
	if !ok {
		return
	}
	pdf, err := service.RenderConsumptionStatementPDF(statement)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	filename := strings.ReplaceAll(statement.StatementNo, "\"", "") + ".pdf"
	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	c.Header("Cache-Control", "private, no-store")
	c.Data(http.StatusOK, "application/pdf", pdf)
}

func AdminListStatementMonthly(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	items, total, err := model.ListStatementMonthlySummaries(c.Query("month"), c.Query("keyword"), pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(items)
	common.ApiSuccess(c, pageInfo)
}

func AdminGenerateStatement(c *gin.Context) {
	var request statementGenerateRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		common.ApiError(c, err)
		return
	}
	statement, err := model.GenerateConsumptionStatement(model.StatementGenerateInput{
		UserID: request.UserID, Month: request.Month, Source: model.StatementSourceAdmin,
		GeneratedBy: c.GetInt("id"), BillingTitle: request.BillingTitle,
		BillingAddress: request.BillingAddress, Now: time.Now(),
	})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "statement.generate", map[string]any{"statement_id": statement.ID, "statement_no": statement.StatementNo, "user_id": statement.UserID, "month_start": statement.MonthStart})
	common.ApiSuccess(c, statement)
}

func AdminListStatementHistory(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	items, total, err := model.ListConsumptionStatements(model.StatementHistoryFilter{
		Month: c.Query("month"), Keyword: c.Query("keyword"), Source: c.Query("source"),
		Offset: pageInfo.GetStartIdx(), Limit: pageInfo.GetPageSize(),
	})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(items)
	common.ApiSuccess(c, pageInfo)
}
