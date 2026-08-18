package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// affiliateRedeemFixture isolates the account and redemption rows used by
// the rebate tests. The model package test suite shares one SQLite database,
// so each fixture also restores the process-wide settings it changes.
type affiliateRedeemFixture struct {
	inviter    *User
	invitee    *User
	redemption *Redemption
}

func setupAffiliateRedeemFixture(t *testing.T, quota int, withInviter bool) *affiliateRedeemFixture {
	t.Helper()
	require.NoError(t, DB.AutoMigrate(&Redemption{}, &ActivityGrant{}))
	clearAffiliateRedeemTables(t)

	affiliateSetting := operation_setting.GetAffiliateSetting()
	previousAffiliateSetting := *affiliateSetting
	activitySetting := operation_setting.GetActivitySetting()
	previousActivitySetting := *activitySetting
	previousQuotaForNewUser := common.QuotaForNewUser
	previousQuotaForInviter := common.QuotaForInviter
	previousQuotaForInvitee := common.QuotaForInvitee

	// Keep each test explicit about the feature it enables. In particular,
	// legacy fixed registration rewards must not leak into these scenarios.
	*affiliateSetting = operation_setting.AffiliateSetting{}
	activitySetting.NewUserRedeemBonusEnabled = false
	common.QuotaForNewUser = 0
	common.QuotaForInviter = 0
	common.QuotaForInvitee = 0

	t.Cleanup(func() {
		*affiliateSetting = previousAffiliateSetting
		*activitySetting = previousActivitySetting
		common.QuotaForNewUser = previousQuotaForNewUser
		common.QuotaForInviter = previousQuotaForInviter
		common.QuotaForInvitee = previousQuotaForInvitee
		clearAffiliateRedeemTables(t)
	})

	fixture := &affiliateRedeemFixture{}
	if withInviter {
		fixture.inviter = &User{
			Username:  "affiliate-rebate-inviter",
			Password:  "password",
			Status:    common.UserStatusEnabled,
			AffCode:   "affiliate-rebate-code",
			CreatedAt: common.GetTimestamp(),
		}
		require.NoError(t, DB.Create(fixture.inviter).Error)
	}

	fixture.invitee = &User{
		Username:  "affiliate-rebate-invitee",
		Password:  "password",
		Status:    common.UserStatusEnabled,
		CreatedAt: common.GetTimestamp(),
	}
	if fixture.inviter != nil {
		fixture.invitee.InviterId = fixture.inviter.Id
	}
	require.NoError(t, DB.Create(fixture.invitee).Error)

	fixture.redemption = &Redemption{
		Name:        "affiliate-rebate-redemption",
		Key:         "affiliate-rebate-key-0000000000000001",
		Status:      common.RedemptionCodeStatusEnabled,
		Quota:       quota,
		CreatedTime: common.GetTimestamp(),
	}
	require.NoError(t, DB.Create(fixture.redemption).Error)
	return fixture
}

func clearAffiliateRedeemTables(t *testing.T) {
	t.Helper()
	require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(&Redemption{}).Error)
	require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&ActivityGrant{}).Error)
	require.NoError(t, DB.Exec("DELETE FROM users").Error)
	require.NoError(t, DB.Exec("DELETE FROM logs").Error)
}

func loadAffiliateTestUser(t *testing.T, id int) User {
	t.Helper()
	var user User
	require.NoError(t, DB.First(&user, "id = ?", id).Error)
	return user
}

func TestRedeemCreditsInviterByConfiguredPercentOfOriginalQuota(t *testing.T) {
	fixture := setupAffiliateRedeemFixture(t, 1000, true)
	affiliateSetting := operation_setting.GetAffiliateSetting()
	affiliateSetting.RedeemRebateEnabled = true
	affiliateSetting.RedeemRebatePercent = 25

	totalQuota, err := Redeem(fixture.redemption.Key, fixture.invitee.Id)
	require.NoError(t, err)
	assert.Equal(t, 1000, totalQuota, "rebate belongs to the inviter, not the invitee")

	invitee := loadAffiliateTestUser(t, fixture.invitee.Id)
	assert.Equal(t, 1000, invitee.Quota)
	inviter := loadAffiliateTestUser(t, fixture.inviter.Id)
	assert.Equal(t, 250, inviter.AffQuota)
	assert.Equal(t, 250, inviter.AffHistoryQuota)
}

func TestRedeemRebateUsesOriginalQuotaNotNewUserBonus(t *testing.T) {
	fixture := setupAffiliateRedeemFixture(t, 1000, true)
	affiliateSetting := operation_setting.GetAffiliateSetting()
	affiliateSetting.RedeemRebateEnabled = true
	affiliateSetting.RedeemRebatePercent = 20
	activitySetting := operation_setting.GetActivitySetting()
	activitySetting.NewUserRedeemBonusEnabled = true
	activitySetting.NewUserRedeemBonusPercent = 30
	activitySetting.NewUserRedeemBonusWindowDays = 1

	totalQuota, err := Redeem(fixture.redemption.Key, fixture.invitee.Id)
	require.NoError(t, err)
	assert.Equal(t, 1300, totalQuota)

	invitee := loadAffiliateTestUser(t, fixture.invitee.Id)
	assert.Equal(t, 1300, invitee.Quota, "invitee receives the independent activity bonus")
	inviter := loadAffiliateTestUser(t, fixture.inviter.Id)
	assert.Equal(t, 200, inviter.AffQuota, "rebate must use 20% of the 1000-code value")
	assert.Equal(t, 200, inviter.AffHistoryQuota)
}

func TestRedeemRebateIsAppliedOnlyOnceForAUsedCode(t *testing.T) {
	fixture := setupAffiliateRedeemFixture(t, 500, true)
	affiliateSetting := operation_setting.GetAffiliateSetting()
	affiliateSetting.RedeemRebateEnabled = true
	affiliateSetting.RedeemRebatePercent = 50

	quota, err := Redeem(fixture.redemption.Key, fixture.invitee.Id)
	require.NoError(t, err)
	assert.Equal(t, 500, quota)

	_, err = Redeem(fixture.redemption.Key, fixture.invitee.Id)
	require.Error(t, err)
	inviter := loadAffiliateTestUser(t, fixture.inviter.Id)
	assert.Equal(t, 250, inviter.AffQuota)
	assert.Equal(t, 250, inviter.AffHistoryQuota)
	invitee := loadAffiliateTestUser(t, fixture.invitee.Id)
	assert.Equal(t, 500, invitee.Quota)
}

func TestRedeemSkipsRebateWhenDisabledOrInviteeHasNoInviter(t *testing.T) {
	t.Run("disabled", func(t *testing.T) {
		fixture := setupAffiliateRedeemFixture(t, 1000, true)
		affiliateSetting := operation_setting.GetAffiliateSetting()
		affiliateSetting.RedeemRebatePercent = 25
		// Enabled remains false.

		_, err := Redeem(fixture.redemption.Key, fixture.invitee.Id)
		require.NoError(t, err)
		inviter := loadAffiliateTestUser(t, fixture.inviter.Id)
		assert.Zero(t, inviter.AffQuota)
		assert.Zero(t, inviter.AffHistoryQuota)
	})

	t.Run("no inviter", func(t *testing.T) {
		fixture := setupAffiliateRedeemFixture(t, 1000, false)
		affiliateSetting := operation_setting.GetAffiliateSetting()
		affiliateSetting.RedeemRebateEnabled = true
		affiliateSetting.RedeemRebatePercent = 25

		_, err := Redeem(fixture.redemption.Key, fixture.invitee.Id)
		require.NoError(t, err)
		invitee := loadAffiliateTestUser(t, fixture.invitee.Id)
		assert.Equal(t, 1000, invitee.Quota)
	})

	t.Run("missing inviter", func(t *testing.T) {
		fixture := setupAffiliateRedeemFixture(t, 1000, false)
		require.NoError(t, DB.Model(&User{}).Where("id = ?", fixture.invitee.Id).Update("inviter_id", 999999).Error)
		affiliateSetting := operation_setting.GetAffiliateSetting()
		affiliateSetting.RedeemRebateEnabled = true
		affiliateSetting.RedeemRebatePercent = 25

		_, err := Redeem(fixture.redemption.Key, fixture.invitee.Id)
		require.NoError(t, err)
		invitee := loadAffiliateTestUser(t, fixture.invitee.Id)
		assert.Equal(t, 1000, invitee.Quota)
	})
}

func TestRedeemSkipsRebateAtAffiliateQuotaLimit(t *testing.T) {
	fixture := setupAffiliateRedeemFixture(t, 1000, true)
	require.NoError(t, DB.Model(&User{}).Where("id = ?", fixture.inviter.Id).Updates(map[string]interface{}{
		"aff_quota":   common.MaxQuota,
		"aff_history": common.MaxQuota,
	}).Error)
	affiliateSetting := operation_setting.GetAffiliateSetting()
	affiliateSetting.RedeemRebateEnabled = true
	affiliateSetting.RedeemRebatePercent = 100

	_, err := Redeem(fixture.redemption.Key, fixture.invitee.Id)
	require.NoError(t, err)
	invitee := loadAffiliateTestUser(t, fixture.invitee.Id)
	assert.Equal(t, 1000, invitee.Quota, "optional rebate overflow must not reject redemption")
	inviter := loadAffiliateTestUser(t, fixture.inviter.Id)
	assert.Equal(t, common.MaxQuota, inviter.AffQuota)
	assert.Equal(t, common.MaxQuota, inviter.AffHistoryQuota)
}

func TestInsertPersistsInviterAndDoesNotGrantLegacyFixedRewards(t *testing.T) {
	clearAffiliateRedeemTables(t)
	previousNewUser := common.QuotaForNewUser
	previousInviter := common.QuotaForInviter
	previousInvitee := common.QuotaForInvitee
	previousPayment := *operation_setting.GetPaymentSetting()
	common.QuotaForNewUser = 100
	common.QuotaForInviter = 700
	common.QuotaForInvitee = 800
	payment := operation_setting.GetPaymentSetting()
	payment.ComplianceConfirmed = true
	payment.ComplianceTermsVersion = operation_setting.CurrentComplianceTermsVersion
	t.Cleanup(func() {
		common.QuotaForNewUser = previousNewUser
		common.QuotaForInviter = previousInviter
		common.QuotaForInvitee = previousInvitee
		*payment = previousPayment
		clearAffiliateRedeemTables(t)
	})

	inviter := &User{
		Username: "legacy-reward-inviter",
		Password: "password",
		Status:   common.UserStatusEnabled,
		AffCode:  "legacy-reward-code",
	}
	require.NoError(t, DB.Create(inviter).Error)

	invitee := &User{
		Username: "legacy-reward-invitee",
		Password: "password",
		Status:   common.UserStatusEnabled,
	}
	require.NoError(t, invitee.Insert(inviter.Id))

	storedInvitee := loadAffiliateTestUser(t, invitee.Id)
	assert.Equal(t, inviter.Id, storedInvitee.InviterId)
	assert.Equal(t, 100, storedInvitee.Quota, "legacy invitee fixed quota is no longer granted at registration")
	storedInviter := loadAffiliateTestUser(t, inviter.Id)
	assert.Equal(t, 1, storedInviter.AffCount)
	assert.Zero(t, storedInviter.AffQuota)
	assert.Zero(t, storedInviter.AffHistoryQuota)
}

func TestInsertWithTxPersistsInviterForOAuthRegistration(t *testing.T) {
	clearAffiliateRedeemTables(t)
	previousNewUser := common.QuotaForNewUser
	common.QuotaForNewUser = 0
	t.Cleanup(func() {
		common.QuotaForNewUser = previousNewUser
		clearAffiliateRedeemTables(t)
	})

	inviter := &User{
		Username: "oauth-inviter",
		Password: "password",
		Status:   common.UserStatusEnabled,
		AffCode:  "oauth-inviter-code",
	}
	require.NoError(t, DB.Create(inviter).Error)
	oauthUser := &User{
		Username: "oauth-invitee",
		Password: "password",
		Status:   common.UserStatusEnabled,
	}
	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		return oauthUser.InsertWithTx(tx, inviter.Id)
	}))

	stored := loadAffiliateTestUser(t, oauthUser.Id)
	assert.Equal(t, inviter.Id, stored.InviterId)

	// The post-commit OAuth finalizer records the invitation count separately
	// from the persisted relationship.
	oauthUser.FinalizeOAuthUserCreation(inviter.Id)
	storedInviter := loadAffiliateTestUser(t, inviter.Id)
	assert.Equal(t, 1, storedInviter.AffCount)
	assert.Zero(t, storedInviter.AffQuota)
	assert.Zero(t, storedInviter.AffHistoryQuota)
}
