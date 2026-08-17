package controller

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupActivityControllerTest(t *testing.T) *model.User {
	t.Helper()
	gin.SetMode(gin.TestMode)
	previousRedisEnabled := common.RedisEnabled
	previousMainDatabase := common.MainDatabaseType()
	previousLogDatabase := common.LogDatabaseType()
	previousDB := model.DB
	previousLogDB := model.LOG_DB
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	common.RedisEnabled = false
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	model.DB = db
	model.LOG_DB = db
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.ActivityGrant{}, &model.ActivityCampaign{}))

	activitySetting := operation_setting.GetActivitySetting()
	previousSetting := *activitySetting
	activitySetting.NewUserRedeemBonusEnabled = true
	activitySetting.NewUserRedeemBonusPercent = 25
	activitySetting.NewUserRedeemBonusWindowDays = 2
	t.Cleanup(func() {
		*activitySetting = previousSetting
		common.RedisEnabled = previousRedisEnabled
		common.SetDatabaseTypes(previousMainDatabase, previousLogDatabase)
		sqlDB, dbErr := db.DB()
		if dbErr == nil {
			_ = sqlDB.Close()
		}
		model.DB = previousDB
		model.LOG_DB = previousLogDB
	})

	user := &model.User{
		Username: "activity-controller-user",
		Password: "password",
		Status:   common.UserStatusEnabled,
		Group:    "default",
	}
	require.NoError(t, db.Create(user).Error)
	return user
}

func getUserActivitiesForTest(t *testing.T, userId int) userActivity {
	t.Helper()
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/user/activities", nil)
	context.Set("id", userId)

	GetUserActivities(context)

	require.Equal(t, http.StatusOK, recorder.Code)
	var payload struct {
		Success bool `json:"success"`
		Data    struct {
			ServerTime int64          `json:"server_time"`
			Activities []userActivity `json:"activities"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &payload))
	require.True(t, payload.Success)
	require.Len(t, payload.Data.Activities, 1)
	assert.Greater(t, payload.Data.ServerTime, int64(0))
	return payload.Data.Activities[0]
}

func TestGetUserActivitiesReturnsNewUserWindowAndCumulativeBonus(t *testing.T) {
	user := setupActivityControllerTest(t)

	activity := getUserActivitiesForTest(t, user.Id)
	assert.Equal(t, model.ActivityKeyNewUserRedeemBonus, activity.Id)
	assert.Equal(t, "active", activity.Status)
	assert.Equal(t, float64(25), activity.BonusPercent)
	assert.Equal(t, user.CreatedAt+2*24*60*60, activity.EndsAt)
	assert.NotNil(t, activity.Action)

	require.NoError(t, model.DB.Create(&model.ActivityGrant{
		ActivityKey: model.ActivityKeyNewUserRedeemBonus,
		UserId:      user.Id,
		SourceType:  model.ActivityGrantSourceRedeem,
		SourceRef:   "redemption-1",
		Quota:       250,
	}).Error)
	updated := getUserActivitiesForTest(t, user.Id)
	assert.Equal(t, "active", updated.Status)
	assert.Equal(t, int64(250), updated.RewardQuota)
	assert.NotNil(t, updated.Action)
}

func TestGetUserActivitiesIncludesClaimableCampaign(t *testing.T) {
	user := setupActivityControllerTest(t)
	now := common.GetTimestamp()
	campaign := &model.ActivityCampaign{
		ActivityKey: "controller-claimable-campaign",
		Type:        model.ActivityCampaignTypeClaimable,
		Title:       "Campaign title",
		Description: "Campaign description",
		AmountUSD:   "0.0006",
		Quota:       300,
		StartsAt:    now - 1,
		EndsAt:      now + 3600,
		CreatedBy:   99,
	}
	require.NoError(t, model.CreateActivityCampaign(context.Background(), campaign))

	recorder := httptest.NewRecorder()
	requestContext, _ := gin.CreateTestContext(recorder)
	requestContext.Request = httptest.NewRequest(http.MethodGet, "/api/user/activities", nil)
	requestContext.Set("id", user.Id)
	GetUserActivities(requestContext)

	var payload struct {
		Success bool `json:"success"`
		Data    struct {
			Activities []userActivity `json:"activities"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &payload))
	require.True(t, payload.Success)
	require.Len(t, payload.Data.Activities, 2)
	activity := payload.Data.Activities[1]
	assert.Equal(t, campaign.ActivityKey, activity.Id)
	assert.Equal(t, "claimable", activity.Status)
	require.NotNil(t, activity.Action)
	assert.Equal(t, "claim", activity.Action.Type)
	assert.Equal(t, "/api/user/activities/"+campaign.ActivityKey+"/claim", activity.Action.Endpoint)
}

func TestCreateActivityCampaignGeneratesActivityKey(t *testing.T) {
	user := setupActivityControllerTest(t)
	recorder := httptest.NewRecorder()
	requestContext, _ := gin.CreateTestContext(recorder)
	requestContext.Request = httptest.NewRequest(
		http.MethodPost,
		"/api/activity/admin/campaigns",
		strings.NewReader(fmt.Sprintf(`{"type":"claimable","title":"Generated key campaign","amount_usd":"0.01","ends_at":%d}`, common.GetTimestamp()+3600)),
	)
	requestContext.Request.Header.Set("Content-Type", "application/json")
	requestContext.Set("id", user.Id)

	CreateActivityCampaign(requestContext)

	require.Equal(t, http.StatusOK, recorder.Code)
	var payload struct {
		Success bool `json:"success"`
		Data    struct {
			Campaign model.ActivityCampaign `json:"campaign"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &payload))
	require.True(t, payload.Success)
	assert.True(t, strings.HasPrefix(payload.Data.Campaign.ActivityKey, "campaign_"))
	assert.NotEmpty(t, payload.Data.Campaign.ActivityKey)
}
