package controller

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

// GetLinkedAccounts returns the account-linking view safe for the current
// user. Main accounts see the whole ordinary-user cluster; subaccounts see
// only their main account.
func GetLinkedAccounts(c *gin.Context) {
	view, err := model.GetAccountLinkage(c.GetInt("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, view)
}

type riskExemptionRequest struct {
	UserID    int    `json:"user_id"`
	Reason    string `json:"reason"`
	ExpiresAt int64  `json:"expires_at"`
}

func ListRiskExemptions(c *gin.Context) {
	rows, err := model.ListRiskExemptions()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, rows)
}

func AddRiskExemption(c *gin.Context) {
	var request riskExemptionRequest
	if err := common.DecodeJson(c.Request.Body, &request); err != nil || request.UserID <= 0 {
		common.ApiErrorMsg(c, "用户 ID 无效")
		return
	}
	var user model.User
	if err := model.DB.Unscoped().Where("id = ?", request.UserID).First(&user).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	if user.Role >= common.RoleAdminUser {
		common.ApiErrorMsg(c, "管理员账号已自动排除，无需加入白名单")
		return
	}
	if request.ExpiresAt < 0 {
		common.ApiErrorMsg(c, "过期时间无效")
		return
	}
	row := &model.RiskExemption{UserID: request.UserID, Reason: strings.TrimSpace(request.Reason), CreatedBy: c.GetInt("id"), ExpiresAt: request.ExpiresAt, Enabled: true, CreatedAt: time.Now().Unix()}
	affected, err := model.UpsertRiskExemptionAndDetach(row)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "risk.exemption_add", map[string]any{"user_id": request.UserID, "reason": row.Reason, "expires_at": row.ExpiresAt, "restored_user_ids": affected})
	common.ApiSuccess(c, row)
}

func RemoveRiskExemption(c *gin.Context) {
	userID := 0
	if _, err := fmt.Sscanf(c.Param("id"), "%d", &userID); err != nil || userID <= 0 {
		common.ApiErrorMsg(c, "用户 ID 无效")
		return
	}
	if err := model.DisableRiskExemption(userID); err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "risk.exemption_remove", map[string]any{"user_id": userID})
	common.ApiSuccess(c, gin.H{"user_id": userID})
}

func BackfillRiskAccountLinks(c *gin.Context) {
	options := model.RiskBackfillOptions{}
	if c.Request.Body != nil && c.Request.ContentLength != 0 {
		if err := common.DecodeJson(c.Request.Body, &options); err != nil && !errors.Is(err, io.EOF) {
			common.ApiErrorMsg(c, "历史扫描参数无效")
			return
		}
	}
	run, task, created, err := service.StartRiskAccountBackfill(options, c.GetInt("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if run == nil || task == nil {
		common.ApiErrorMsg(c, "历史扫描任务状态不可用")
		return
	}
	recordManageAudit(c, "risk.account_backfill_preview", map[string]any{"run_id": run.RunID, "task_id": task.TaskID, "created": created})
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": gin.H{"run": run, "task": task.ToResponse(), "created": created}})
}

func GetRiskAccountBackfill(c *gin.Context) {
	var run *model.RiskAccountBackfillRun
	var err error
	if c.Param("run_id") == "latest" {
		run, err = model.GetLatestRiskAccountBackfillRun()
	} else {
		run, err = model.GetRiskAccountBackfillRun(c.Param("run_id"))
	}
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, run)
}

func PauseRiskAccountBackfill(c *gin.Context) {
	if err := model.RequestRiskAccountBackfillPause(c.Param("run_id")); err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "risk.account_backfill_pause", map[string]any{"run_id": c.Param("run_id")})
	run, _ := model.GetRiskAccountBackfillRun(c.Param("run_id"))
	common.ApiSuccess(c, run)
}

func ResumeRiskAccountBackfill(c *gin.Context) {
	run, task, err := service.ResumeRiskAccountBackfill(c.Param("run_id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "risk.account_backfill_resume", map[string]any{"run_id": run.RunID, "task_id": task.TaskID})
	common.ApiSuccess(c, gin.H{"run": run, "task": task.ToResponse()})
}

func ListRiskAccountReviews(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	rows, total, err := model.ListRiskAccountReviews(c.Query("status"), page, pageSize)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"items": rows, "total": total, "page": page, "page_size": pageSize})
}

type riskReviewResolutionRequest struct {
	Resolution string `json:"resolution"`
}

func ApproveRiskAccountReview(c *gin.Context) {
	reviewID, err := strconv.ParseInt(c.Param("review_id"), 10, 64)
	if err != nil || reviewID <= 0 {
		common.ApiErrorMsg(c, "审核记录 ID 无效")
		return
	}
	request := riskReviewResolutionRequest{}
	if c.Request.Body != nil && c.Request.ContentLength != 0 {
		if err := common.DecodeJson(c.Request.Body, &request); err != nil && !errors.Is(err, io.EOF) {
			common.ApiErrorMsg(c, "审核参数无效")
			return
		}
	}
	if err := model.ApproveRiskAccountReview(reviewID, c.GetInt("id"), request.Resolution); err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "risk.review_approve", map[string]any{"review_id": reviewID, "resolution": request.Resolution})
	common.ApiSuccess(c, gin.H{"review_id": reviewID})
}

func RejectRiskAccountReview(c *gin.Context) {
	reviewID, err := strconv.ParseInt(c.Param("review_id"), 10, 64)
	if err != nil || reviewID <= 0 {
		common.ApiErrorMsg(c, "审核记录 ID 无效")
		return
	}
	request := riskReviewResolutionRequest{}
	if c.Request.Body != nil && c.Request.ContentLength != 0 {
		if err := common.DecodeJson(c.Request.Body, &request); err != nil && !errors.Is(err, io.EOF) {
			common.ApiErrorMsg(c, "审核参数无效")
			return
		}
	}
	if err := model.RejectRiskAccountReview(reviewID, c.GetInt("id"), request.Resolution); err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "risk.review_reject", map[string]any{"review_id": reviewID, "resolution": request.Resolution})
	common.ApiSuccess(c, gin.H{"review_id": reviewID})
}

type riskAccountDetachRequest struct {
	Reenable bool   `json:"reenable"`
	Reason   string `json:"reason"`
}

func DetachRiskAccount(c *gin.Context) {
	userID, err := strconv.Atoi(c.Param("id"))
	if err != nil || userID <= 0 {
		common.ApiErrorMsg(c, "用户 ID 无效")
		return
	}
	request := riskAccountDetachRequest{Reenable: true}
	if c.Request.Body != nil && c.Request.ContentLength != 0 {
		if err := common.DecodeJson(c.Request.Body, &request); err != nil && !errors.Is(err, io.EOF) {
			common.ApiErrorMsg(c, "解除关联参数无效")
			return
		}
	}
	affected, err := model.DetachRiskAccount(userID, request.Reenable, c.GetInt("id"), request.Reason)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "risk.account_detach", map[string]any{"user_id": userID, "affected_user_ids": affected, "reenable": request.Reenable, "reason": request.Reason})
	common.ApiSuccess(c, gin.H{"affected_user_ids": affected})
}
