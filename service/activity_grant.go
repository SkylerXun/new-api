package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

const (
	allUsersActivityGrantDefaultBatchSize = 100
	allUsersActivityGrantMaxBatchSize     = 1000
)

type EnqueueAllUsersActivityGrantRequest struct {
	AmountUSD   float64
	Quota       int
	Reason      string
	IssuedBy    int
	ActivityKey string
	CampaignID  int64
	BatchSize   int
}

type AllUsersActivityGrantPayload struct {
	AmountUSD      float64 `json:"amount_usd"`
	Quota          int     `json:"quota"`
	Reason         string  `json:"reason"`
	IssuedBy       int     `json:"issued_by"`
	MaxUserId      int     `json:"max_user_id"`
	RecipientCount int64   `json:"recipient_count,omitempty"`
	BatchSize      int     `json:"batch_size"`
	ActivityKey    string  `json:"activity_key"`
	CampaignID     int64   `json:"campaign_id,omitempty"`
}

type AllUsersActivityGrantState struct {
	Total      int64 `json:"total"`
	Processed  int64 `json:"processed"`
	Granted    int64 `json:"granted"`
	Skipped    int64 `json:"skipped"`
	LastUserId int   `json:"last_user_id"`
	Progress   int   `json:"progress"`
}

type AllUsersActivityGrantResult struct {
	ActivityKey string  `json:"activity_key"`
	CampaignID  int64   `json:"campaign_id,omitempty"`
	AmountUSD   float64 `json:"amount_usd"`
	Quota       int     `json:"quota"`
	Reason      string  `json:"reason"`
	IssuedBy    int     `json:"issued_by"`
	Total       int64   `json:"total"`
	Processed   int64   `json:"processed"`
	Granted     int64   `json:"granted"`
	Skipped     int64   `json:"skipped"`
	TotalQuota  int64   `json:"total_quota"`
}

type allUsersActivityGrantHandler struct{}

func (allUsersActivityGrantHandler) Type() string {
	return model.SystemTaskTypeQuotaGrantAll
}

func (allUsersActivityGrantHandler) Run(ctx context.Context, task *model.SystemTask, runnerId string) {
	payload := AllUsersActivityGrantPayload{}
	if err := task.DecodePayload(&payload); err != nil {
		failAllUsersActivityGrantTask(task, runnerId, 0, err)
		return
	}
	if err := validateAllUsersActivityGrantPayload(payload); err != nil {
		failAllUsersActivityGrantTask(task, runnerId, payload.CampaignID, err)
		return
	}
	if payload.CampaignID > 0 {
		if err := model.MarkActivityCampaignTaskRunning(ctx, payload.CampaignID, task.TaskID); err != nil {
			failAllUsersActivityGrantTask(task, runnerId, payload.CampaignID, err)
			return
		}
	}

	state := AllUsersActivityGrantState{}
	if err := task.DecodeState(&state); err != nil {
		failAllUsersActivityGrantTask(task, runnerId, payload.CampaignID, err)
		return
	}

	if payload.CampaignID > 0 {
		state.Total = payload.RecipientCount
	} else {
		total, err := model.CountActivityGrantEligibleUsers(ctx, payload.MaxUserId)
		if err != nil {
			failAllUsersActivityGrantTask(task, runnerId, payload.CampaignID, err)
			return
		}
		state.Total = total
	}

	sourceType := model.ActivityGrantSourceAllUsersGrant
	// An operator retry creates a new system-task ID. The activity key is the
	// idempotency boundary, so keep a stable source reference across retries.
	sourceRef := model.ActivityGrantSourceRefAllUsers
	if payload.CampaignID > 0 {
		sourceType = model.ActivityGrantSourceCampaignImmediate
		sourceRef = model.ActivityGrantSourceRefImmediate
	}

	for {
		if err := ctx.Err(); err != nil {
			failAllUsersActivityGrantTask(task, runnerId, payload.CampaignID, err)
			return
		}
		userIds, err := model.ListActivityGrantEligibleUserIds(
			ctx,
			state.LastUserId,
			payload.MaxUserId,
			payload.BatchSize,
		)
		if err != nil {
			failAllUsersActivityGrantTask(task, runnerId, payload.CampaignID, err)
			return
		}
		if len(userIds) == 0 {
			break
		}

		granted, skipped, err := model.GrantActivityQuotaBatch(
			ctx,
			userIds,
			payload.ActivityKey,
			sourceType,
			sourceRef,
			payload.Quota,
		)
		if err != nil {
			failAllUsersActivityGrantTask(task, runnerId, payload.CampaignID, err)
			return
		}

		state.Processed += int64(len(userIds))
		state.Granted += int64(granted)
		state.Skipped += int64(skipped)
		state.LastUserId = userIds[len(userIds)-1]
		state.Progress = allUsersActivityGrantProgress(state.Processed, state.Total)
		if err := model.UpdateSystemTaskState(task.TaskID, runnerId, state); err != nil {
			logSystemTaskLockError(ctx, task, err)
			return
		}
	}

	state.Progress = 100
	if err := model.UpdateSystemTaskState(task.TaskID, runnerId, state); err != nil {
		logSystemTaskLockError(ctx, task, err)
		return
	}

	result := AllUsersActivityGrantResult{
		ActivityKey: payload.ActivityKey,
		CampaignID:  payload.CampaignID,
		AmountUSD:   payload.AmountUSD,
		Quota:       payload.Quota,
		Reason:      payload.Reason,
		IssuedBy:    payload.IssuedBy,
		Total:       state.Total,
		Processed:   state.Processed,
		Granted:     state.Granted,
		Skipped:     state.Skipped,
	}
	if state.Granted > math.MaxInt64/int64(payload.Quota) {
		result.TotalQuota = math.MaxInt64
	} else {
		result.TotalQuota = state.Granted * int64(payload.Quota)
	}
	if err := model.FinishSystemTask(task.TaskID, runnerId, model.SystemTaskStatusSucceeded, result, ""); err != nil {
		logSystemTaskLockError(ctx, task, err)
		return
	}
	if payload.CampaignID > 0 {
		if err := model.CompleteActivityCampaignTask(ctx, payload.CampaignID, task.TaskID); err != nil {
			common.SysError(fmt.Sprintf("failed to mark activity campaign %d complete: %v", payload.CampaignID, err))
		}
	}
}

func init() {
	RegisterSystemTaskHandler(allUsersActivityGrantHandler{})
}

// EnqueueActivityCampaignImmediateGrant schedules an already-created immediate
// campaign. The campaign key is used as the ledger activity key, while every
// recipient shares the stable immediate source reference.
func EnqueueActivityCampaignImmediateGrant(campaign *model.ActivityCampaign, amountUSD float64) (*model.SystemTask, bool, error) {
	if campaign == nil {
		return nil, false, errors.New("activity campaign is required")
	}
	reason := strings.TrimSpace(campaign.Reason)
	if reason == "" {
		reason = campaign.Title
	}
	return EnqueueAllUsersActivityGrant(EnqueueAllUsersActivityGrantRequest{
		AmountUSD:   amountUSD,
		Quota:       campaign.Quota,
		Reason:      reason,
		IssuedBy:    campaign.CreatedBy,
		ActivityKey: campaign.ActivityKey,
		CampaignID:  campaign.Id,
	})
}

// EnqueueAllUsersActivityGrant snapshots the current enabled-user ID ceiling
// and schedules an idempotent grant. Quota must already be converted from USD
// to the site's internal quota units by the caller.
func EnqueueAllUsersActivityGrant(request EnqueueAllUsersActivityGrantRequest) (*model.SystemTask, bool, error) {
	request.Reason = strings.TrimSpace(request.Reason)
	request.ActivityKey = strings.TrimSpace(request.ActivityKey)
	if request.Quota <= 0 || request.Quota > common.MaxQuota || request.AmountUSD <= 0 || math.IsNaN(request.AmountUSD) || math.IsInf(request.AmountUSD, 0) {
		return nil, false, errors.New("activity grant amount must be positive")
	}
	if request.Reason == "" {
		return nil, false, errors.New("activity grant reason is required")
	}
	if request.IssuedBy <= 0 {
		return nil, false, errors.New("activity grant issuer is required")
	}
	if request.CampaignID < 0 {
		return nil, false, errors.New("activity campaign id is invalid")
	}
	if request.BatchSize <= 0 {
		request.BatchSize = allUsersActivityGrantDefaultBatchSize
	}
	if request.BatchSize > allUsersActivityGrantMaxBatchSize {
		return nil, false, fmt.Errorf("activity grant batch size cannot exceed %d", allUsersActivityGrantMaxBatchSize)
	}
	if request.ActivityKey == "" {
		return nil, false, errors.New("activity grant key is required")
	}
	if request.ActivityKey == model.ActivityKeyNewUserRedeemBonus {
		return nil, false, errors.New("activity grant key is reserved")
	}
	if len(request.ActivityKey) > 128 {
		return nil, false, errors.New("activity grant key is too long")
	}
	maxUserId := 0
	recipientCount := int64(0)
	if request.CampaignID > 0 {
		campaign, err := model.GetActivityCampaignByID(context.Background(), request.CampaignID)
		if err != nil {
			return nil, false, err
		}
		if campaign == nil {
			return nil, false, model.ErrActivityCampaignNotFound
		}
		if campaign.ActivityKey != request.ActivityKey || campaign.Type != model.ActivityCampaignTypeImmediate || campaign.Status != model.ActivityCampaignStatusQueued {
			return nil, false, model.ErrActivityCampaignInvalidStatus
		}
		maxUserId = campaign.RecipientMaxUserID
		recipientCount = campaign.RecipientCount
	} else {
		var err error
		maxUserId, recipientCount, err = model.GetActivityGrantTargetSnapshot(context.Background())
		if err != nil {
			return nil, false, err
		}
	}
	payload := AllUsersActivityGrantPayload{
		AmountUSD:      request.AmountUSD,
		Quota:          request.Quota,
		Reason:         request.Reason,
		IssuedBy:       request.IssuedBy,
		MaxUserId:      maxUserId,
		RecipientCount: recipientCount,
		BatchSize:      request.BatchSize,
		ActivityKey:    request.ActivityKey,
		CampaignID:     request.CampaignID,
	}
	return enqueueAllUsersActivityGrantTask(payload)
}

func enqueueAllUsersActivityGrantTask(payload AllUsersActivityGrantPayload) (*model.SystemTask, bool, error) {
	activeTask, err := model.GetActiveSystemTask(model.SystemTaskTypeQuotaGrantAll)
	if err != nil {
		return nil, false, err
	}
	if activeTask != nil {
		return activeTask, false, nil
	}

	task, err := model.CreateSystemTask(model.SystemTaskTypeQuotaGrantAll, payload, nil)
	if err != nil {
		activeTask, activeErr := model.GetActiveSystemTask(model.SystemTaskTypeQuotaGrantAll)
		if activeErr == nil && activeTask != nil {
			return activeTask, false, nil
		}
		return nil, false, err
	}
	if payload.CampaignID > 0 {
		if err := model.AttachActivityCampaignTask(context.Background(), payload.CampaignID, task.TaskID); err != nil {
			// The task payload remains authoritative. Its handler retries this
			// attachment at start, so a transient association failure cannot
			// turn a valid task into an untracked credit operation.
			common.SysError(fmt.Sprintf("failed to attach activity campaign %d to task %s: %v", payload.CampaignID, task.TaskID, err))
		}
	}
	notifySystemTaskRunner()
	return task, true, nil
}

func validateAllUsersActivityGrantPayload(payload AllUsersActivityGrantPayload) error {
	if payload.Quota <= 0 || payload.Quota > common.MaxQuota || payload.AmountUSD <= 0 || math.IsNaN(payload.AmountUSD) || math.IsInf(payload.AmountUSD, 0) {
		return errors.New("activity grant amount must be positive")
	}
	activityKey := strings.TrimSpace(payload.ActivityKey)
	if strings.TrimSpace(payload.Reason) == "" || payload.IssuedBy <= 0 || activityKey == "" || len(activityKey) > 128 {
		return errors.New("activity grant metadata is invalid")
	}
	if activityKey == model.ActivityKeyNewUserRedeemBonus {
		return errors.New("activity grant key is reserved")
	}
	if payload.CampaignID < 0 || payload.MaxUserId < 0 || payload.RecipientCount < 0 {
		return errors.New("activity grant user snapshot is invalid")
	}
	if payload.BatchSize <= 0 || payload.BatchSize > allUsersActivityGrantMaxBatchSize {
		return errors.New("activity grant batch size is invalid")
	}
	return nil
}

func failAllUsersActivityGrantTask(task *model.SystemTask, runnerID string, campaignID int64, err error) {
	failSystemTask(task, runnerID, err)
	if campaignID <= 0 {
		return
	}
	if updateErr := model.FailActivityCampaignTask(context.Background(), campaignID, task.TaskID, err.Error()); updateErr != nil {
		common.SysError(fmt.Sprintf("failed to mark activity campaign %d failed: %v", campaignID, updateErr))
	}
}

func allUsersActivityGrantProgress(processed int64, total int64) int {
	if total <= 0 || processed >= total {
		return 100
	}
	if processed <= 0 {
		return 0
	}
	return int(processed * 100 / total)
}
