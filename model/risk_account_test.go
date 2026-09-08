package model

import (
	"context"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupRiskAccountTest(t *testing.T) {
	previousDB, previousLogDB := DB, LOG_DB
	previousRedisEnabled := common.RedisEnabled
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	DB, LOG_DB = db, db
	require.NoError(t, DB.AutoMigrate(
		&User{}, &UserSession{}, &UserOAuthBinding{}, &Log{}, &ActivityGrant{},
		&TopUp{},
		&RiskExemption{}, &RiskAccountLink{}, &RiskAccountObservation{},
		&RiskAccountReview{}, &RiskAccountAction{}, &RiskRewardHold{}, &RiskAccountBackfillRun{},
	))
	common.RedisEnabled = false
	t.Cleanup(func() {
		DB, LOG_DB = previousDB, previousLogDB
		common.RedisEnabled = previousRedisEnabled
	})
}

func createRiskTestUser(t *testing.T, username, email string, role, status int, createdAt int64) *User {
	user := &User{Username: username, Password: "unused", Email: email, DisplayName: username, Role: role, Status: status, AffCode: "risk-aff-" + username, CreatedAt: createdAt}
	require.NoError(t, DB.Create(user).Error)
	return user
}

func TestRiskAccountLinkageKeepsEarliestAndExemptsAdminAndWhitelist(t *testing.T) {
	setupRiskAccountTest(t)
	now := time.Now().Unix()
	main := createRiskTestUser(t, "main", "same@example.com", common.RoleCommonUser, common.UserStatusEnabled, now-10)
	sub := createRiskTestUser(t, "sub", "same@example.com", common.RoleCommonUser, common.UserStatusEnabled, now)
	admin := createRiskTestUser(t, "admin", "same@example.com", common.RoleAdminUser, common.UserStatusEnabled, now-20)
	whitelisted := createRiskTestUser(t, "white", "same@example.com", common.RoleCommonUser, common.UserStatusEnabled, now-30)
	require.NoError(t, UpsertRiskExemption(&RiskExemption{UserID: whitelisted.Id, CreatedBy: admin.Id, Enabled: true}))

	require.NoError(t, EvaluateAndEnforceAccountLinkage(sub.Id, "203.0.113.10", "test", "device-1"))

	var reloaded User
	require.NoError(t, DB.First(&reloaded, sub.Id).Error)
	assert.Equal(t, common.UserStatusDisabled, reloaded.Status)
	reloaded = User{}
	require.NoError(t, DB.First(&reloaded, main.Id).Error)
	assert.Equal(t, common.UserStatusEnabled, reloaded.Status)
	for _, exemptID := range []int{admin.Id, whitelisted.Id} {
		var count int64
		require.NoError(t, DB.Model(&RiskAccountLink{}).Where("user_id = ?", exemptID).Count(&count).Error)
		assert.Zero(t, count)
	}
	view, err := GetAccountLinkage(main.Id)
	require.NoError(t, err)
	assert.Equal(t, "main", view.Relation)
	assert.Equal(t, 2, view.Total)
}

func TestRiskAccountBackfillDoesNotLinkSharedIPAlone(t *testing.T) {
	setupRiskAccountTest(t)
	now := time.Now().Unix()
	first := createRiskTestUser(t, "first", "first@example.com", common.RoleCommonUser, common.UserStatusEnabled, now-10)
	second := createRiskTestUser(t, "second", "second@example.com", common.RoleCommonUser, common.UserStatusEnabled, now)
	require.NoError(t, DB.Create(&UserSession{SID: "one", UserID: first.Id, Version: 1, UserAuthVersion: 1, Status: UserSessionStatusRevoked, RefreshHash: "a", IP: "203.0.113.20", UserAgent: "browser-a", CreatedAt: now}).Error)
	require.NoError(t, DB.Create(&UserSession{SID: "two", UserID: second.Id, Version: 1, UserAuthVersion: 1, Status: UserSessionStatusRevoked, RefreshHash: "b", IP: "203.0.113.20", UserAgent: "browser-b", CreatedAt: now}).Error)

	run, err := CreateRiskAccountBackfillRun(RiskBackfillOptions{}, first.Id)
	require.NoError(t, err)
	preview, err := PreviewRiskAccountBackfill(context.Background(), run, nil)
	require.NoError(t, err)
	assert.Zero(t, preview.CandidateGroups)
	var reviewCount int64
	require.NoError(t, DB.Model(&RiskAccountReview{}).Count(&reviewCount).Error)
	assert.Zero(t, reviewCount)
}

func TestRiskAccountIgnoresExemptDeviceObservations(t *testing.T) {
	setupRiskAccountTest(t)
	now := time.Now().Unix()
	admin := createRiskTestUser(t, "device-admin", "device-admin@example.com", common.RoleAdminUser, common.UserStatusEnabled, now-20)
	ordinary := createRiskTestUser(t, "device-user", "device-user@example.com", common.RoleCommonUser, common.UserStatusEnabled, now)
	whitelisted := createRiskTestUser(t, "device-white", "device-white@example.com", common.RoleCommonUser, common.UserStatusEnabled, now-10)
	require.NoError(t, UpsertRiskExemption(&RiskExemption{UserID: whitelisted.Id, CreatedBy: admin.Id, Enabled: true}))
	require.NoError(t, observeRiskAccount(admin.Id, "203.0.113.40", "browser", "shared-device", now-1))
	require.NoError(t, observeRiskAccount(whitelisted.Id, "203.0.113.41", "browser", "shared-device", now-1))
	require.NoError(t, EvaluateAndEnforceAccountLinkage(ordinary.Id, "203.0.113.42", "browser", "shared-device"))
	var count int64
	require.NoError(t, DB.Model(&RiskAccountLink{}).Where("user_id = ?", ordinary.Id).Count(&count).Error)
	assert.Zero(t, count)
}

func TestDisabledLinkedSubaccountPaymentCallbackCannotCredit(t *testing.T) {
	setupRiskAccountTest(t)
	now := time.Now().Unix()
	createRiskTestUser(t, "payment-main", "payment@example.com", common.RoleCommonUser, common.UserStatusEnabled, now-10)
	sub := createRiskTestUser(t, "payment-sub", "payment@example.com", common.RoleCommonUser, common.UserStatusEnabled, now)
	require.NoError(t, EvaluateAndEnforceAccountLinkage(sub.Id, "", "", ""))
	order := &TopUp{UserId: sub.Id, Amount: 1, Money: 1, TradeNo: "risk-disabled-callback", PaymentProvider: PaymentProviderEpay, PaymentMethod: "alipay", Status: common.TopUpStatusPending, CreateTime: now}
	require.NoError(t, DB.Create(order).Error)
	_, err := RechargeEpay(order.TradeNo, "alipay", "203.0.113.50")
	require.Error(t, err)
	var reloaded User
	require.NoError(t, DB.First(&reloaded, sub.Id).Error)
	assert.Equal(t, common.UserStatusDisabled, reloaded.Status)
	assert.Zero(t, reloaded.Quota)
	var reloadedOrder TopUp
	require.NoError(t, DB.Where("trade_no = ?", order.TradeNo).First(&reloadedOrder).Error)
	assert.Equal(t, common.TopUpStatusPending, reloadedOrder.Status)
}

func TestRiskAccountMainUsesIDAsTimestampTieBreaker(t *testing.T) {
	setupRiskAccountTest(t)
	now := time.Now().Unix()
	first := createRiskTestUser(t, "first", "tie@example.com", common.RoleCommonUser, common.UserStatusEnabled, now)
	second := createRiskTestUser(t, "second", "tie@example.com", common.RoleCommonUser, common.UserStatusEnabled, now)
	require.NoError(t, EvaluateAndEnforceAccountLinkage(second.Id, "", "", ""))

	view, err := GetAccountLinkage(first.Id)
	require.NoError(t, err)
	require.NotNil(t, view.MainAccount)
	assert.Equal(t, first.Id, view.MainAccount.ID)
}

func TestHistoricalPreviewRequiresApprovalBeforeDisablingAccounts(t *testing.T) {
	setupRiskAccountTest(t)
	now := time.Now().Unix()
	main := createRiskTestUser(t, "preview-main", "preview@example.com", common.RoleCommonUser, common.UserStatusEnabled, now-10)
	sub := createRiskTestUser(t, "preview-sub", "preview@example.com", common.RoleCommonUser, common.UserStatusEnabled, now)

	run, err := CreateRiskAccountBackfillRun(RiskBackfillOptions{PageSize: 50}, main.Id)
	require.NoError(t, err)
	preview, err := PreviewRiskAccountBackfill(context.Background(), run, nil)
	require.NoError(t, err)
	assert.Equal(t, int64(1), preview.CandidateGroups)

	var reloaded User
	require.NoError(t, DB.First(&reloaded, sub.Id).Error)
	assert.Equal(t, common.UserStatusEnabled, reloaded.Status)

	reviews, total, err := ListRiskAccountReviews(RiskAccountReviewStatusPending, 1, 20)
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, reviews, 1)
	require.NoError(t, ApproveRiskAccountReview(reviews[0].ID, main.Id, "confirmed in test"))
	require.NoError(t, DB.First(&reloaded, sub.Id).Error)
	assert.Equal(t, common.UserStatusDisabled, reloaded.Status)
}

func TestStrongSignalDoesNotSilentlyDemoteExistingMain(t *testing.T) {
	setupRiskAccountTest(t)
	now := time.Now().Unix()
	older := createRiskTestUser(t, "older", "older@example.com", common.RoleCommonUser, common.UserStatusEnabled, now-30)
	existingMain := createRiskTestUser(t, "existing-main", "cluster@example.com", common.RoleCommonUser, common.UserStatusEnabled, now-20)
	existingSub := createRiskTestUser(t, "existing-sub", "cluster@example.com", common.RoleCommonUser, common.UserStatusEnabled, now-10)
	require.NoError(t, EvaluateAndEnforceAccountLinkage(existingSub.Id, "", "", ""))

	// The older account touches the established cluster through a new device
	// signal. It would become main, so the merge must wait for admin approval.
	require.NoError(t, observeRiskAccount(existingMain.Id, "", "", "shared-device", now))
	require.NoError(t, EvaluateAndEnforceAccountLinkage(older.Id, "", "", "shared-device"))

	var link RiskAccountLink
	require.NoError(t, DB.Where("user_id = ?", existingMain.Id).First(&link).Error)
	assert.Equal(t, existingMain.Id, link.MainUserID)
	var olderLinks int64
	require.NoError(t, DB.Model(&RiskAccountLink{}).Where("user_id = ?", older.Id).Count(&olderLinks).Error)
	assert.Zero(t, olderLinks)
	var pending int64
	require.NoError(t, DB.Model(&RiskAccountReview{}).Where("kind = ? AND status = ?", RiskAccountReviewKindClusterMerge, RiskAccountReviewStatusPending).Count(&pending).Error)
	assert.Equal(t, int64(1), pending)
	reviews, total, err := ListRiskAccountReviews(RiskAccountReviewStatusPending, 1, 20)
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, reviews, 1)
	require.NoError(t, ApproveRiskAccountReview(reviews[0].ID, existingMain.Id, "approved merge in test"))
	var olderLink RiskAccountLink
	require.NoError(t, DB.Where("user_id = ?", older.Id).First(&olderLink).Error)
	assert.Equal(t, older.Id, olderLink.MainUserID)
	var demoted User
	require.NoError(t, DB.First(&demoted, existingMain.Id).Error)
	assert.Equal(t, common.UserStatusDisabled, demoted.Status)
}

func TestAddingWhitelistDetachesAndRestoresSubaccount(t *testing.T) {
	setupRiskAccountTest(t)
	now := time.Now().Unix()
	admin := createRiskTestUser(t, "whitelist-admin", "admin@example.com", common.RoleAdminUser, common.UserStatusEnabled, now-30)
	main := createRiskTestUser(t, "whitelist-main", "whitelist@example.com", common.RoleCommonUser, common.UserStatusEnabled, now-20)
	sub := createRiskTestUser(t, "whitelist-sub", "whitelist@example.com", common.RoleCommonUser, common.UserStatusEnabled, now-10)
	require.NoError(t, EvaluateAndEnforceAccountLinkage(sub.Id, "", "", ""))

	affected, err := UpsertRiskExemptionAndDetach(&RiskExemption{UserID: sub.Id, CreatedBy: admin.Id, Reason: "founder test"})
	require.NoError(t, err)
	assert.Contains(t, affected, sub.Id)
	var reloaded User
	require.NoError(t, DB.First(&reloaded, sub.Id).Error)
	assert.Equal(t, common.UserStatusEnabled, reloaded.Status)
	view, err := GetAccountLinkage(sub.Id)
	require.NoError(t, err)
	assert.Equal(t, "none", view.Relation)
	mainView, err := GetAccountLinkage(main.Id)
	require.NoError(t, err)
	assert.Equal(t, "none", mainView.Relation)
}

func TestSubaccountViewOnlyContainsMainAccount(t *testing.T) {
	setupRiskAccountTest(t)
	now := time.Now().Unix()
	main := createRiskTestUser(t, "view-main", "view@example.com", common.RoleCommonUser, common.UserStatusEnabled, now-30)
	firstSub := createRiskTestUser(t, "view-sub-1", "view@example.com", common.RoleCommonUser, common.UserStatusEnabled, now-20)
	secondSub := createRiskTestUser(t, "view-sub-2", "view@example.com", common.RoleCommonUser, common.UserStatusEnabled, now-10)
	require.NoError(t, EvaluateAndEnforceAccountLinkage(secondSub.Id, "", "", ""))

	mainView, err := GetAccountLinkage(main.Id)
	require.NoError(t, err)
	assert.Equal(t, 3, mainView.Total)
	assert.Equal(t, "enabled", mainView.MainAccount.Status)
	subView, err := GetAccountLinkage(firstSub.Id)
	require.NoError(t, err)
	require.Len(t, subView.LinkedAccounts, 1)
	assert.Equal(t, main.Id, subView.LinkedAccounts[0].ID)
	assert.Equal(t, "enabled", subView.LinkedAccounts[0].Status)
}
