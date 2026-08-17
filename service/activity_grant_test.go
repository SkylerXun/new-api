package service

import (
	"context"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupAllUsersActivityGrantTest(t *testing.T) {
	t.Helper()
	require.NoError(t, model.DB.AutoMigrate(&model.ActivityGrant{}, &model.ActivityCampaign{}))
	require.NoError(t, model.DB.Exec("DELETE FROM activity_grants").Error)
	require.NoError(t, model.DB.Exec("DELETE FROM activity_campaigns").Error)
	require.NoError(t, model.DB.Exec("DELETE FROM system_task_locks").Error)
	require.NoError(t, model.DB.Exec("DELETE FROM system_tasks").Error)
	require.NoError(t, model.DB.Exec("DELETE FROM users").Error)
	t.Cleanup(func() {
		_ = model.DB.Exec("DELETE FROM activity_grants").Error
		_ = model.DB.Exec("DELETE FROM activity_campaigns").Error
		_ = model.DB.Exec("DELETE FROM system_task_locks").Error
		_ = model.DB.Exec("DELETE FROM system_tasks").Error
		_ = model.DB.Exec("DELETE FROM users").Error
	})
}

func createAllUsersActivityGrantTestUser(t *testing.T, suffix string, status int) *model.User {
	t.Helper()
	user := &model.User{
		Username:    "all-grant-user-" + suffix,
		Password:    "password",
		DisplayName: "Grant User " + suffix,
		Role:        common.RoleCommonUser,
		Status:      status,
		Group:       "default",
		AffCode:     "all-grant-aff-" + suffix,
	}
	require.NoError(t, model.DB.Create(user).Error)
	return user
}

func runAllUsersActivityGrantTask(t *testing.T, task *model.SystemTask, runnerId string) {
	t.Helper()
	claimed, ok, err := model.ClaimSystemTask(task.ID, model.SystemTaskTypeQuotaGrantAll, runnerId, common.GetTimestamp()+60)
	require.NoError(t, err)
	require.True(t, ok)
	allUsersActivityGrantHandler{}.Run(context.Background(), claimed, runnerId)
}

func TestAllUsersActivityGrantTaskCreditsEnabledUsersAndIsIdempotent(t *testing.T) {
	setupAllUsersActivityGrantTest(t)
	first := createAllUsersActivityGrantTestUser(t, "enabled-1", common.UserStatusEnabled)
	disabled := createAllUsersActivityGrantTestUser(t, "disabled", common.UserStatusDisabled)
	second := createAllUsersActivityGrantTestUser(t, "enabled-2", common.UserStatusEnabled)

	request := EnqueueAllUsersActivityGrantRequest{
		AmountUSD:   1,
		Quota:       500000,
		Reason:      "launch credit",
		IssuedBy:    99,
		ActivityKey: "all-users-launch-credit",
		BatchSize:   1,
	}
	task, created, err := EnqueueAllUsersActivityGrant(request)
	require.NoError(t, err)
	require.True(t, created)
	runAllUsersActivityGrantTask(t, task, "activity-runner-1")

	for _, userId := range []int{first.Id, second.Id} {
		var user model.User
		require.NoError(t, model.DB.First(&user, userId).Error)
		assert.Equal(t, 500000, user.Quota)
	}
	var disabledUser model.User
	require.NoError(t, model.DB.First(&disabledUser, disabled.Id).Error)
	assert.Zero(t, disabledUser.Quota)

	finished, err := model.GetSystemTaskByTaskID(task.TaskID)
	require.NoError(t, err)
	require.NotNil(t, finished)
	assert.Equal(t, model.SystemTaskStatusSucceeded, finished.Status)
	state := AllUsersActivityGrantState{}
	require.NoError(t, finished.DecodeState(&state))
	assert.Equal(t, int64(2), state.Processed)
	assert.Equal(t, int64(2), state.Granted)
	assert.Zero(t, state.Skipped)
	assert.Equal(t, 100, state.Progress)

	retryTask, created, err := EnqueueAllUsersActivityGrant(request)
	require.NoError(t, err)
	require.True(t, created)
	runAllUsersActivityGrantTask(t, retryTask, "activity-runner-2")

	for _, userId := range []int{first.Id, second.Id} {
		var user model.User
		require.NoError(t, model.DB.First(&user, userId).Error)
		assert.Equal(t, 500000, user.Quota)
	}
	retried, err := model.GetSystemTaskByTaskID(retryTask.TaskID)
	require.NoError(t, err)
	require.NotNil(t, retried)
	retryState := AllUsersActivityGrantState{}
	require.NoError(t, retried.DecodeState(&retryState))
	assert.Zero(t, retryState.Granted)
	assert.Equal(t, int64(2), retryState.Skipped)

	grantCount, err := model.CountActivityGrants(context.Background(), request.ActivityKey)
	require.NoError(t, err)
	assert.Equal(t, int64(2), grantCount)
	for _, userID := range []int{first.Id, second.Id} {
		grant, grantErr := model.GetActivityGrantForUserSource(
			userID,
			request.ActivityKey,
			model.ActivityGrantSourceRefAllUsers,
		)
		require.NoError(t, grantErr)
		require.NotNil(t, grant)
	}
}

func TestAllUsersActivityGrantTaskSkipsQuotaLimitAndCompletes(t *testing.T) {
	setupAllUsersActivityGrantTest(t)
	overflow := createAllUsersActivityGrantTestUser(t, "overflow", common.UserStatusEnabled)
	eligible := createAllUsersActivityGrantTestUser(t, "eligible", common.UserStatusEnabled)
	require.NoError(t, model.DB.Model(&model.User{}).Where("id = ?", overflow.Id).
		Update("quota", common.MaxQuota-10).Error)

	task, created, err := EnqueueAllUsersActivityGrant(EnqueueAllUsersActivityGrantRequest{
		AmountUSD:   1,
		Quota:       20,
		Reason:      "skip quota limit",
		IssuedBy:    99,
		ActivityKey: "all-users-skip-quota-limit",
		BatchSize:   2,
	})
	require.NoError(t, err)
	require.True(t, created)
	runAllUsersActivityGrantTask(t, task, "quota-limit-runner")

	var overflowReloaded model.User
	require.NoError(t, model.DB.First(&overflowReloaded, overflow.Id).Error)
	assert.Equal(t, common.MaxQuota-10, overflowReloaded.Quota)
	var eligibleReloaded model.User
	require.NoError(t, model.DB.First(&eligibleReloaded, eligible.Id).Error)
	assert.Equal(t, 20, eligibleReloaded.Quota)

	finished, err := model.GetSystemTaskByTaskID(task.TaskID)
	require.NoError(t, err)
	require.NotNil(t, finished)
	assert.Equal(t, model.SystemTaskStatusSucceeded, finished.Status)
	state := AllUsersActivityGrantState{}
	require.NoError(t, finished.DecodeState(&state))
	assert.Equal(t, int64(2), state.Processed)
	assert.Equal(t, int64(1), state.Granted)
	assert.Equal(t, int64(1), state.Skipped)
}

func TestEnqueueAllUsersActivityGrantRejectsReservedActivityKey(t *testing.T) {
	setupAllUsersActivityGrantTest(t)

	_, _, err := EnqueueAllUsersActivityGrant(EnqueueAllUsersActivityGrantRequest{
		AmountUSD:   1,
		Quota:       500000,
		Reason:      "must not shadow new user bonus",
		IssuedBy:    99,
		ActivityKey: model.ActivityKeyNewUserRedeemBonus,
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "reserved")
}

func TestEnqueueAllUsersActivityGrantRequiresStableActivityKey(t *testing.T) {
	setupAllUsersActivityGrantTest(t)

	_, _, err := EnqueueAllUsersActivityGrant(EnqueueAllUsersActivityGrantRequest{
		AmountUSD: 1,
		Quota:     500000,
		Reason:    "idempotent campaign",
		IssuedBy:  99,
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "required")
}

func TestImmediateCampaignGrantUsesStableSourceAndCompletesCampaign(t *testing.T) {
	setupAllUsersActivityGrantTest(t)
	first := createAllUsersActivityGrantTestUser(t, "campaign-enabled-1", common.UserStatusEnabled)
	second := createAllUsersActivityGrantTestUser(t, "campaign-enabled-2", common.UserStatusEnabled)
	now := common.GetTimestamp()
	campaign := &model.ActivityCampaign{
		ActivityKey: "immediate-campaign",
		Type:        model.ActivityCampaignTypeImmediate,
		Title:       "Immediate campaign",
		Reason:      "campaign credit",
		AmountUSD:   "0.0008",
		Quota:       400,
		StartsAt:    now,
		CreatedBy:   99,
	}
	require.NoError(t, model.CreateActivityCampaign(context.Background(), campaign))
	require.Equal(t, second.Id, campaign.RecipientMaxUserID)
	require.Equal(t, int64(2), campaign.RecipientCount)

	task, created, err := EnqueueActivityCampaignImmediateGrant(campaign, 0.0008)
	require.NoError(t, err)
	require.True(t, created)
	runAllUsersActivityGrantTask(t, task, "campaign-runner")

	for _, userID := range []int{first.Id, second.Id} {
		var user model.User
		require.NoError(t, model.DB.First(&user, userID).Error)
		assert.Equal(t, 400, user.Quota)
		grant, grantErr := model.GetActivityGrantForUserSource(userID, campaign.ActivityKey, model.ActivityGrantSourceRefImmediate)
		require.NoError(t, grantErr)
		require.NotNil(t, grant)
		assert.Equal(t, model.ActivityGrantSourceCampaignImmediate, grant.SourceType)
	}

	finished, err := model.GetActivityCampaignByID(context.Background(), campaign.Id)
	require.NoError(t, err)
	require.NotNil(t, finished)
	assert.Equal(t, model.ActivityCampaignStatusCompleted, finished.Status)
	assert.Equal(t, task.TaskID, finished.TaskID)
}
