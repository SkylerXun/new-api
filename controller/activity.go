package controller

import (
	"fmt"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

type userActivityAction struct {
	To       string `json:"to,omitempty"`
	Label    string `json:"label"`
	Type     string `json:"type,omitempty"`
	Endpoint string `json:"endpoint,omitempty"`
}

type userActivity struct {
	Id               string              `json:"id"`
	Type             string              `json:"type"`
	Title            string              `json:"title"`
	Description      string              `json:"description"`
	Status           string              `json:"status"`
	StartsAt         int64               `json:"starts_at"`
	EndsAt           int64               `json:"ends_at"`
	RemainingSeconds int64               `json:"remaining_seconds"`
	BonusPercent     float64             `json:"bonus_percent"`
	RewardQuota      int64               `json:"reward_quota,omitempty"`
	Action           *userActivityAction `json:"action,omitempty"`
	GrantedAt        int64               `json:"granted_at,omitempty"`
}

func GetUserActivities(c *gin.Context) {
	userId := c.GetInt("id")
	cursor := int64(0)
	view := strings.ToLower(strings.TrimSpace(c.Query("view")))
	if view == "" {
		view = "ongoing"
	}
	if view != "ongoing" && view != "participated" {
		common.ApiErrorMsg(c, "activity view is invalid")
		return
	}
	if cursorText := strings.TrimSpace(c.Query("cursor")); cursorText != "" {
		parsedCursor, parseErr := strconv.ParseInt(cursorText, 10, 64)
		if parseErr != nil || parsedCursor < 1 {
			common.ApiErrorMsg(c, "activity cursor is invalid")
			return
		}
		cursor = parsedCursor
	}
	var user model.User
	if err := model.DB.Select("id", "created_at").Where("id = ?", userId).First(&user).Error; err != nil {
		common.ApiError(c, err)
		return
	}

	now := common.GetTimestamp()
	activitySetting := operation_setting.GetActivitySetting()
	windowDays := activitySetting.NewUserRedeemBonusWindowDays
	bonusPercent := activitySetting.NewUserRedeemBonusPercent
	validConfiguration := windowDays >= 1 && windowDays <= 3650 &&
		!math.IsNaN(bonusPercent) && !math.IsInf(bonusPercent, 0) &&
		bonusPercent >= 0 && bonusPercent <= 1000
	if !validConfiguration {
		windowDays = 0
		bonusPercent = 0
	}
	endsAt := user.CreatedAt + int64(windowDays)*24*60*60
	remainingSeconds := endsAt - now
	if remainingSeconds < 0 {
		remainingSeconds = 0
	}

	newUserActivity := userActivity{
		Id:               model.ActivityKeyNewUserRedeemBonus,
		Type:             "new_user_topup_bonus",
		Title:            "新用户兑换加赠",
		Description:      fmt.Sprintf("注册后 %d 天内每次兑换码充值，额外赠送 %g%% 额度。", windowDays, bonusPercent),
		Status:           "active",
		StartsAt:         user.CreatedAt,
		EndsAt:           endsAt,
		RemainingSeconds: remainingSeconds,
		BonusPercent:     bonusPercent,
	}

	cumulativeReward, err := model.SumActivityGrantQuotaForUser(c.Request.Context(), userId, model.ActivityKeyNewUserRedeemBonus)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	newUserActivity.RewardQuota = cumulativeReward
	if !activitySetting.NewUserRedeemBonusEnabled || !validConfiguration || bonusPercent <= 0 || user.CreatedAt <= 0 {
		newUserActivity.Status = "unavailable"
	} else if now >= endsAt {
		newUserActivity.Status = "expired"
	} else {
		newUserActivity.Action = &userActivityAction{To: "/wallet", Label: "立即充值", Type: "navigate"}
	}

	activities := make([]userActivity, 0, 50)
	if view == "participated" && newUserActivity.RewardQuota > 0 {
		newUserActivity.Status = "credited"
		newUserActivity.RemainingSeconds = 0
		newUserActivity.Action = nil
	}
	if (view == "ongoing" && newUserActivity.Status == "active") ||
		(view == "participated" && newUserActivity.RewardQuota > 0) {
		activities = append(activities, newUserActivity)
	}
	var campaigns []*model.ActivityCampaign
	var nextCursor int64
	var listErr error
	if view == "participated" {
		campaigns, nextCursor, listErr = model.ListActivityCampaignsForUserPage(c.Request.Context(), userId, cursor, 50)
	} else {
		campaigns, nextCursor, listErr = model.ListActivityCampaignsPage(c.Request.Context(), cursor, 50)
	}
	if listErr != nil {
		common.ApiError(c, listErr)
		return
	}
	activityKeys := make([]string, 0, len(campaigns))
	for _, campaign := range campaigns {
		activityKeys = append(activityKeys, campaign.ActivityKey)
	}
	grantsByKey, err := model.ListActivityGrantsForUserActivityKeys(c.Request.Context(), userId, activityKeys)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	for _, campaign := range campaigns {
		campaignGrants := grantsByKey[campaign.ActivityKey]
		if view == "participated" {
			if len(campaignGrants) == 0 {
				continue
			}
		} else {
			if campaign.Type != model.ActivityCampaignTypeClaimable || campaign.Status != model.ActivityCampaignStatusActive || campaign.EndsAt <= now || now < campaign.StartsAt {
				continue
			}
			if len(campaignGrants) == 0 {
				eligible, eligibilityErr := model.IsActivityCampaignUserEligible(c.Request.Context(), campaign, userId)
				if eligibilityErr != nil {
					common.ApiError(c, eligibilityErr)
					return
				}
				if !eligible {
					continue
				}
			}
		}
		if view == "participated" && campaign.AudienceType == model.ActivityCampaignAudienceSelected && len(campaignGrants) == 0 {
			continue
		}
		activities = append(activities, userActivityFromCampaign(campaign, campaignGrants, userId, now, view))
	}
	if view == "participated" && cursor == 0 {
		allGrants, grantsErr := model.ListActivityGrantsForUser(c.Request.Context(), userId)
		if grantsErr != nil {
			common.ApiError(c, grantsErr)
			return
		}
		knownKeys := make(map[string]struct{}, len(campaigns)+1)
		knownKeys[model.ActivityKeyNewUserRedeemBonus] = struct{}{}
		grantKeys := make([]string, 0, len(allGrants))
		for _, grant := range allGrants {
			grantKeys = append(grantKeys, grant.ActivityKey)
			if grant.ActivityKey == model.ActivityKeyNewUserRedeemBonus {
				for index := range activities {
					if activities[index].Id == model.ActivityKeyNewUserRedeemBonus && grant.CreatedAt > activities[index].GrantedAt {
						activities[index].GrantedAt = grant.CreatedAt
					}
				}
			}
		}
		existingKeys, keyErr := model.ListExistingActivityCampaignKeys(c.Request.Context(), grantKeys)
		if keyErr != nil {
			common.ApiError(c, keyErr)
			return
		}
		for key := range existingKeys {
			knownKeys[key] = struct{}{}
		}
		legacyByKey := make(map[string]*userActivity)
		for _, grant := range allGrants {
			if _, known := knownKeys[grant.ActivityKey]; known {
				continue
			}
			activity := legacyByKey[grant.ActivityKey]
			if activity == nil {
				activity = &userActivity{
					Id: grant.ActivityKey, Type: grant.SourceType, Title: "管理员活动赠送",
					Description: "管理员通过活动中心发放的奖励。", Status: "credited", GrantedAt: grant.CreatedAt,
				}
				legacyByKey[grant.ActivityKey] = activity
			}
			activity.RewardQuota += int64(grant.Quota)
			if grant.CreatedAt > activity.GrantedAt {
				activity.GrantedAt = grant.CreatedAt
			}
		}
		for _, activity := range legacyByKey {
			activities = append(activities, *activity)
		}
		sort.SliceStable(activities, func(i, j int) bool {
			return activities[i].GrantedAt > activities[j].GrantedAt
		})
	}

	response := gin.H{
		"server_time": now,
		"activities":  activities,
	}
	if nextCursor > 0 {
		response["next_cursor"] = strconv.FormatInt(nextCursor, 10)
	}
	common.ApiSuccess(c, response)
}

func ClaimUserActivity(c *gin.Context) {
	grant, granted, err := model.ClaimActivityCampaignQuota(c.Request.Context(), c.GetInt("id"), c.Param("key"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{
		"granted":      granted,
		"reward_quota": grant.Quota,
		"grant":        grant,
	})
}

func userActivityFromCampaign(campaign *model.ActivityCampaign, grants []model.ActivityGrant, userId int, now int64, view string) userActivity {
	activity := userActivity{
		Id:          campaign.ActivityKey,
		Type:        campaign.Type,
		Title:       campaign.Title,
		Description: campaign.Description,
		Status:      "unavailable",
		StartsAt:    campaign.StartsAt,
		EndsAt:      campaign.EndsAt,
	}
	if view == "participated" {
		activity.Status = "credited"
		var total int64
		for _, grant := range grants {
			total += int64(grant.Quota)
			if activity.GrantedAt == 0 || grant.CreatedAt > activity.GrantedAt {
				activity.GrantedAt = grant.CreatedAt
			}
		}
		activity.RewardQuota = total
		return activity
	}
	if campaign.EndsAt > now {
		activity.RemainingSeconds = campaign.EndsAt - now
	}

	sourceRef := model.ActivityGrantSourceRefImmediate
	if campaign.Type == model.ActivityCampaignTypeClaimable {
		sourceRef = model.ActivityGrantSourceRefClaim
	}
	for _, grant := range grants {
		if grant.SourceRef == sourceRef {
			activity.Status = "credited"
			if campaign.Type == model.ActivityCampaignTypeClaimable {
				activity.Status = "claimed"
			}
			activity.RewardQuota = int64(grant.Quota)
			return activity
		}
	}

	if campaign.Type != model.ActivityCampaignTypeClaimable {
		if campaign.Status == model.ActivityCampaignStatusClosed {
			activity.Status = "closed"
		}
		return activity
	}
	if campaign.AudienceType != model.ActivityCampaignAudienceSelected && (campaign.RecipientMaxUserID <= 0 || userId > campaign.RecipientMaxUserID) {
		return activity
	}
	if campaign.Status == model.ActivityCampaignStatusClosed {
		activity.Status = "closed"
		return activity
	}
	if campaign.Status != model.ActivityCampaignStatusActive || now < campaign.StartsAt {
		return activity
	}
	if campaign.EndsAt <= now {
		activity.Status = "expired"
		return activity
	}
	activity.Status = "claimable"
	activity.Action = &userActivityAction{
		Label:    "立即领取",
		Type:     "claim",
		Endpoint: "/api/user/activities/" + campaign.ActivityKey + "/claim",
	}
	return activity
}

type activityCampaignRequest struct {
	ActivityKey      string `json:"activity_key"`
	Key              string `json:"key"`
	Type             string `json:"type"`
	Title            string `json:"title"`
	Description      string `json:"description"`
	Reason           string `json:"reason"`
	AmountUSD        string `json:"amount_usd"`
	Quota            *int   `json:"quota"`
	StartsAt         int64  `json:"starts_at"`
	EndsAt           int64  `json:"ends_at"`
	AudienceType     string `json:"audience_type"`
	RecipientUserIDs []int  `json:"recipient_user_ids"`
}

func ListActivityCampaigns(c *gin.Context) {
	limit, _ := strconv.Atoi(c.Query("limit"))
	campaigns, err := model.ListActivityCampaigns(c.Request.Context(), limit)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, campaigns)
}

func CreateActivityCampaign(c *gin.Context) {
	var request activityCampaignRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		common.ApiError(c, err)
		return
	}
	if strings.EqualFold(strings.TrimSpace(request.Type), model.ActivityCampaignTypeImmediate) {
		activeTask, err := model.GetActiveSystemTask(model.SystemTaskTypeQuotaGrantAll)
		if err != nil {
			common.ApiError(c, err)
			return
		}
		if activeTask != nil {
			common.ApiErrorMsg(c, "an all-user activity grant is already running")
			return
		}
	}
	quota, amountUSD, err := activityCampaignRequestQuota(request)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	activityKey := strings.TrimSpace(request.ActivityKey)
	if activityKey == "" {
		activityKey = strings.TrimSpace(request.Key)
	}
	if activityKey == "" {
		randomKey, randomKeyErr := common.GenerateRandomCharsKey(24)
		if randomKeyErr != nil {
			common.ApiError(c, randomKeyErr)
			return
		}
		activityKey = "campaign_" + randomKey
	}
	campaign := &model.ActivityCampaign{
		ActivityKey:  activityKey,
		Type:         request.Type,
		Title:        request.Title,
		Description:  request.Description,
		Reason:       request.Reason,
		AmountUSD:    strings.TrimSpace(request.AmountUSD),
		Quota:        quota,
		StartsAt:     request.StartsAt,
		EndsAt:       request.EndsAt,
		CreatedBy:    c.GetInt("id"),
		AudienceType: strings.TrimSpace(request.AudienceType),
	}
	if campaign.AmountUSD == "" {
		campaign.AmountUSD = strconv.FormatFloat(amountUSD, 'f', -1, 64)
	}
	if err := model.CreateActivityCampaignWithRecipients(c.Request.Context(), campaign, request.RecipientUserIDs); err != nil {
		common.ApiError(c, err)
		return
	}

	var task *model.SystemTask
	if campaign.Type == model.ActivityCampaignTypeImmediate {
		task, created, err := service.EnqueueActivityCampaignImmediateGrant(campaign, amountUSD)
		if err != nil {
			_ = model.FailActivityCampaignEnqueue(c.Request.Context(), campaign.Id, err.Error())
			common.ApiError(c, err)
			return
		}
		if !created {
			message := "an all-user activity grant is already running"
			_ = model.FailActivityCampaignEnqueue(c.Request.Context(), campaign.Id, message)
			common.ApiErrorMsg(c, message)
			return
		}
		campaign.TaskID = task.TaskID
	}

	recordManageAudit(c, "activity.campaign_create", map[string]interface{}{
		"activity_key": campaign.ActivityKey,
		"type":         campaign.Type,
		"quota":        logger.LogQuota(campaign.Quota),
		"task_id":      campaign.TaskID,
	})
	data := gin.H{"campaign": campaign}
	if task != nil {
		data["task"] = task.ToResponse()
	}
	common.ApiSuccess(c, data)
}

func CloseActivityCampaign(c *gin.Context) {
	campaign, err := model.CloseActivityCampaign(c.Request.Context(), c.Param("key"), c.GetInt("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "activity.campaign_close", map[string]interface{}{
		"activity_key": campaign.ActivityKey,
	})
	common.ApiSuccess(c, campaign)
}

func activityCampaignRequestQuota(request activityCampaignRequest) (int, float64, error) {
	if common.QuotaPerUnit <= 0 || math.IsNaN(common.QuotaPerUnit) || math.IsInf(common.QuotaPerUnit, 0) {
		return 0, 0, fmt.Errorf("quota conversion is unavailable")
	}
	amountText := strings.TrimSpace(request.AmountUSD)
	if request.Quota != nil && amountText != "" {
		return 0, 0, fmt.Errorf("provide either quota or amount_usd")
	}
	if request.Quota != nil {
		if *request.Quota <= 0 || *request.Quota > common.MaxQuota {
			return 0, 0, fmt.Errorf("activity quota is invalid")
		}
		return *request.Quota, float64(*request.Quota) / common.QuotaPerUnit, nil
	}
	amountUSD, err := decimal.NewFromString(amountText)
	if err != nil || amountUSD.LessThanOrEqual(decimal.Zero) {
		return 0, 0, fmt.Errorf("activity amount must be positive")
	}
	quota, err := common.QuotaFromDecimalStrict(amountUSD.Mul(decimal.NewFromFloat(common.QuotaPerUnit)))
	if err != nil || quota <= 0 {
		return 0, 0, fmt.Errorf("activity amount exceeds the supported range")
	}
	amountFloat, _ := amountUSD.Float64()
	return quota, amountFloat, nil
}

type grantAllUsersRequest struct {
	AmountUSD   string `json:"amount_usd"`
	Reason      string `json:"reason"`
	ActivityKey string `json:"activity_key"`
}

func GrantAllUsersActivityQuota(c *gin.Context) {
	var request grantAllUsersRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的参数"})
		return
	}

	amountText := strings.TrimSpace(request.AmountUSD)
	amountUSD, err := decimal.NewFromString(amountText)
	if err != nil || amountUSD.LessThanOrEqual(decimal.Zero) {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "赠送金额必须大于 0 美元"})
		return
	}
	quota, err := common.QuotaFromDecimalStrict(amountUSD.Mul(decimal.NewFromFloat(common.QuotaPerUnit)))
	if err != nil || quota <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "赠送金额超出可支持范围"})
		return
	}

	reason := strings.TrimSpace(request.Reason)
	if reason == "" {
		reason = "活动中心全员赠送"
	}
	activityKey := strings.TrimSpace(request.ActivityKey)
	if activityKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "缺少发放请求标识，请刷新页面后重试"})
		return
	}
	amountFloat, _ := amountUSD.Float64()
	task, created, err := service.EnqueueAllUsersActivityGrant(service.EnqueueAllUsersActivityGrantRequest{
		AmountUSD:   amountFloat,
		Quota:       quota,
		Reason:      reason,
		IssuedBy:    c.GetInt("id"),
		ActivityKey: activityKey,
	})
	if err != nil {
		common.ApiError(c, err)
		return
	}

	recordManageAudit(c, "activity.grant_all", map[string]interface{}{
		"amount_usd":   amountUSD.String(),
		"quota":        logger.LogQuota(quota),
		"reason":       reason,
		"task_id":      task.TaskID,
		"task_created": created,
	})
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"created": created,
			"task":    task.ToResponse(),
		},
	})
}
