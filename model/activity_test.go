package model

import (
	"context"
	"testing"

	"github.com/QuantumNous/new-api/common"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupActivityGrantTest(t *testing.T) {
	t.Helper()
	require.NoError(t, DB.AutoMigrate(&ActivityGrant{}, &ActivityCampaign{}, &ActivityCampaignRecipient{}))
	require.NoError(t, DB.Exec("DELETE FROM activity_grants").Error)
	require.NoError(t, DB.Exec("DELETE FROM activity_campaigns").Error)
	require.NoError(t, DB.Exec("DELETE FROM activity_campaign_recipients").Error)
	require.NoError(t, DB.Exec("DELETE FROM users").Error)
	t.Cleanup(func() {
		_ = DB.Exec("DELETE FROM activity_grants").Error
		_ = DB.Exec("DELETE FROM activity_campaigns").Error
		_ = DB.Exec("DELETE FROM activity_campaign_recipients").Error
		_ = DB.Exec("DELETE FROM users").Error
	})
}

func createActivityGrantTestUser(t *testing.T, suffix string, status int, quota int) *User {
	t.Helper()
	user := &User{
		Username:    "activity-user-" + suffix,
		Password:    "password",
		DisplayName: "Activity User " + suffix,
		Role:        common.RoleCommonUser,
		Status:      status,
		Quota:       quota,
		Group:       "default",
		AffCode:     "activity-aff-" + suffix,
	}
	require.NoError(t, DB.Create(user).Error)
	return user
}

func TestGrantActivityQuotaTxCreditsEachSourceExactlyOnce(t *testing.T) {
	setupActivityGrantTest(t)
	user := createActivityGrantTestUser(t, "once", common.UserStatusEnabled, 100)

	granted := false
	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		var err error
		granted, err = GrantActivityQuotaTx(tx, user.Id, "new-user-redeem", ActivityGrantSourceRedeem, "redemption-1", 50)
		return err
	}))
	require.True(t, granted)

	var reloaded User
	require.NoError(t, DB.First(&reloaded, user.Id).Error)
	assert.Equal(t, 150, reloaded.Quota)
	count, err := CountActivityGrants(context.Background(), "new-user-redeem")
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)

	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		var err error
		granted, err = GrantActivityQuotaTx(tx, user.Id, "new-user-redeem", ActivityGrantSourceRedeem, "redemption-1", 50)
		return err
	}))
	assert.False(t, granted)
	require.NoError(t, DB.First(&reloaded, user.Id).Error)
	assert.Equal(t, 150, reloaded.Quota)

	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		var err error
		granted, err = GrantActivityQuotaTx(tx, user.Id, "new-user-redeem", ActivityGrantSourceRedeem, "redemption-2", 50)
		return err
	}))
	require.True(t, granted)
	require.NoError(t, DB.First(&reloaded, user.Id).Error)
	assert.Equal(t, 200, reloaded.Quota)
	count, err = CountActivityGrants(context.Background(), "new-user-redeem")
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)
}

func TestClaimActivityCampaignCreditsEligibleRecipientOnce(t *testing.T) {
	setupActivityGrantTest(t)
	recipient := createActivityGrantTestUser(t, "campaign-recipient", common.UserStatusEnabled, 0)
	now := common.GetTimestamp()
	campaign := &ActivityCampaign{
		ActivityKey: "claimable-campaign",
		Type:        ActivityCampaignTypeClaimable,
		Title:       "Claimable campaign",
		Description: "A test activity",
		AmountUSD:   "0.00025",
		Quota:       125,
		StartsAt:    now - 1,
		EndsAt:      now + 3600,
		CreatedBy:   99,
	}
	require.NoError(t, CreateActivityCampaign(context.Background(), campaign))
	require.Equal(t, recipient.Id, campaign.RecipientMaxUserID)
	require.Equal(t, int64(1), campaign.RecipientCount)

	grant, granted, err := ClaimActivityCampaignQuota(context.Background(), recipient.Id, campaign.ActivityKey)
	require.NoError(t, err)
	require.True(t, granted)
	require.NotNil(t, grant)
	assert.Equal(t, ActivityGrantSourceRefClaim, grant.SourceRef)
	assert.Equal(t, 125, grant.Quota)

	repeated, granted, err := ClaimActivityCampaignQuota(context.Background(), recipient.Id, campaign.ActivityKey)
	require.NoError(t, err)
	assert.False(t, granted)
	require.NotNil(t, repeated)
	assert.Equal(t, grant.Id, repeated.Id)

	var reloaded User
	require.NoError(t, DB.First(&reloaded, recipient.Id).Error)
	assert.Equal(t, 125, reloaded.Quota)

	lateUser := createActivityGrantTestUser(t, "campaign-late", common.UserStatusEnabled, 0)
	_, _, err = ClaimActivityCampaignQuota(context.Background(), lateUser.Id, campaign.ActivityKey)
	require.ErrorIs(t, err, ErrActivityCampaignUnavailable)
}

func TestSelectedActivityCampaignOnlyAllowsFrozenRecipients(t *testing.T) {
	setupActivityGrantTest(t)
	recipient := createActivityGrantTestUser(t, "selected-recipient", common.UserStatusEnabled, 0)
	other := createActivityGrantTestUser(t, "selected-other", common.UserStatusEnabled, 0)
	now := common.GetTimestamp()
	campaign := &ActivityCampaign{
		ActivityKey:  "selected-campaign",
		Type:         ActivityCampaignTypeClaimable,
		AudienceType: ActivityCampaignAudienceSelected,
		Title:        "Selected campaign",
		AmountUSD:    "0.01",
		Quota:        100,
		StartsAt:     now - 1,
		EndsAt:       now + 3600,
		CreatedBy:    1,
	}
	require.NoError(t, CreateActivityCampaignWithRecipients(context.Background(), campaign, []int{recipient.Id, recipient.Id}))
	assert.Equal(t, int64(1), campaign.RecipientCount)
	var recipients []ActivityCampaignRecipient
	require.NoError(t, DB.Where("campaign_id = ?", campaign.Id).Find(&recipients).Error)
	assert.Len(t, recipients, 1)

	_, granted, err := ClaimActivityCampaignQuota(context.Background(), other.Id, campaign.ActivityKey)
	require.ErrorIs(t, err, ErrActivityCampaignUnavailable)
	assert.False(t, granted)
	_, granted, err = ClaimActivityCampaignQuota(context.Background(), recipient.Id, campaign.ActivityKey)
	require.NoError(t, err)
	assert.True(t, granted)
}

func TestSelectedActivityCampaignRejectsDisabledRecipientWithoutCreatingCampaign(t *testing.T) {
	setupActivityGrantTest(t)
	disabled := createActivityGrantTestUser(t, "selected-disabled", common.UserStatusDisabled, 0)
	now := common.GetTimestamp()
	campaign := &ActivityCampaign{
		ActivityKey: "selected-disabled-campaign", Type: ActivityCampaignTypeClaimable,
		AudienceType: ActivityCampaignAudienceSelected, Title: "Disabled recipient",
		AmountUSD: "0.01", Quota: 100, StartsAt: now - 1, EndsAt: now + 3600, CreatedBy: 1,
	}

	err := CreateActivityCampaignWithRecipients(context.Background(), campaign, []int{disabled.Id})
	require.Error(t, err)
	var count int64
	require.NoError(t, DB.Model(&ActivityCampaign{}).Where("activity_key = ?", campaign.ActivityKey).Count(&count).Error)
	assert.Zero(t, count)
}

func TestGrantActivityQuotaTxRollsBackLedgerWhenUserIsMissing(t *testing.T) {
	setupActivityGrantTest(t)

	err := DB.Transaction(func(tx *gorm.DB) error {
		_, err := GrantActivityQuotaTx(tx, 99999, "missing-user", ActivityGrantSourceRedeem, "redemption-1", 50)
		return err
	})
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)

	count, countErr := CountActivityGrants(context.Background(), "missing-user")
	require.NoError(t, countErr)
	assert.Equal(t, int64(0), count)
}

func TestGrantActivityQuotaTxRollsBackLedgerOnQuotaOverflow(t *testing.T) {
	setupActivityGrantTest(t)
	user := createActivityGrantTestUser(t, "overflow", common.UserStatusEnabled, common.MaxQuota-10)

	err := DB.Transaction(func(tx *gorm.DB) error {
		_, err := GrantActivityQuotaTx(tx, user.Id, "overflow-grant", ActivityGrantSourceAllUsersGrant, "task-1", 20)
		return err
	})
	require.Error(t, err)

	var reloaded User
	require.NoError(t, DB.First(&reloaded, user.Id).Error)
	assert.Equal(t, common.MaxQuota-10, reloaded.Quota)
	count, countErr := CountActivityGrants(context.Background(), "overflow-grant")
	require.NoError(t, countErr)
	assert.Zero(t, count)
}

func TestActivityGrantTargetSnapshotAndBatchListingExcludeIneligibleUsers(t *testing.T) {
	setupActivityGrantTest(t)
	first := createActivityGrantTestUser(t, "enabled-1", common.UserStatusEnabled, 0)
	createActivityGrantTestUser(t, "disabled", common.UserStatusDisabled, 0)
	second := createActivityGrantTestUser(t, "enabled-2", common.UserStatusEnabled, 0)
	deleted := createActivityGrantTestUser(t, "deleted", common.UserStatusEnabled, 0)
	require.NoError(t, DB.Delete(deleted).Error)

	maxUserId, total, err := GetActivityGrantTargetSnapshot(context.Background())
	require.NoError(t, err)
	assert.Equal(t, second.Id, maxUserId)
	assert.Equal(t, int64(2), total)

	firstPage, err := ListActivityGrantEligibleUserIds(context.Background(), 0, maxUserId, 1)
	require.NoError(t, err)
	assert.Equal(t, []int{first.Id}, firstPage)

	secondPage, err := ListActivityGrantEligibleUserIds(context.Background(), first.Id, maxUserId, 10)
	require.NoError(t, err)
	assert.Equal(t, []int{second.Id}, secondPage)
}

func TestGrantActivityQuotaBatchSkipsMissingAndOverflowUsers(t *testing.T) {
	setupActivityGrantTest(t)
	eligible := createActivityGrantTestUser(t, "batch-eligible", common.UserStatusEnabled, 100)
	overflow := createActivityGrantTestUser(t, "batch-overflow", common.UserStatusEnabled, common.MaxQuota-10)

	granted, skipped, err := GrantActivityQuotaBatch(
		context.Background(),
		[]int{eligible.Id, overflow.Id, 99999},
		"batch-skip-test",
		ActivityGrantSourceAllUsersGrant,
		"task-1",
		20,
	)
	require.NoError(t, err)
	assert.Equal(t, 1, granted)
	assert.Equal(t, 2, skipped)

	var eligibleReloaded User
	require.NoError(t, DB.First(&eligibleReloaded, eligible.Id).Error)
	assert.Equal(t, 120, eligibleReloaded.Quota)
	var overflowReloaded User
	require.NoError(t, DB.First(&overflowReloaded, overflow.Id).Error)
	assert.Equal(t, common.MaxQuota-10, overflowReloaded.Quota)

	grantCount, countErr := CountActivityGrants(context.Background(), "batch-skip-test")
	require.NoError(t, countErr)
	assert.Equal(t, int64(1), grantCount)
}

func TestListActivityCampaignsPageUsesLastCampaignIDAsCursor(t *testing.T) {
	setupActivityGrantTest(t)
	createActivityGrantTestUser(t, "campaign-page", common.UserStatusEnabled, 0)
	now := common.GetTimestamp()
	for _, key := range []string{"campaign-page-one", "campaign-page-two", "campaign-page-three"} {
		campaign := &ActivityCampaign{
			ActivityKey: key,
			Type:        ActivityCampaignTypeClaimable,
			Title:       key,
			AmountUSD:   "0.01",
			Quota:       100,
			StartsAt:    now,
			EndsAt:      now + 3600,
			CreatedBy:   1,
		}
		require.NoError(t, CreateActivityCampaign(context.Background(), campaign))
	}

	firstPage, nextCursor, err := ListActivityCampaignsPage(context.Background(), 0, 2)
	require.NoError(t, err)
	require.Len(t, firstPage, 2)
	assert.Equal(t, "campaign-page-three", firstPage[0].ActivityKey)
	assert.Equal(t, "campaign-page-two", firstPage[1].ActivityKey)
	assert.Equal(t, firstPage[1].Id, nextCursor)

	secondPage, nextCursor, err := ListActivityCampaignsPage(context.Background(), nextCursor, 2)
	require.NoError(t, err)
	require.Len(t, secondPage, 1)
	assert.Equal(t, "campaign-page-one", secondPage[0].ActivityKey)
	assert.Zero(t, nextCursor)
}

func TestActivityCampaignGrantDetailsAndCountsUseOnlyCampaignLedgerSources(t *testing.T) {
	setupActivityGrantTest(t)
	first := createActivityGrantTestUser(t, "details-first", common.UserStatusEnabled, 0)
	second := createActivityGrantTestUser(t, "details-second", common.UserStatusEnabled, 0)
	now := common.GetTimestamp()
	campaign := &ActivityCampaign{
		ActivityKey: "campaign-details", Type: ActivityCampaignTypeClaimable,
		Title: "Campaign details", AmountUSD: "0.01", Quota: 100,
		StartsAt: now - 10, EndsAt: now + 3600, CreatedBy: 1,
	}
	require.NoError(t, CreateActivityCampaign(context.Background(), campaign))
	require.NoError(t, DB.Create(&ActivityGrant{
		ActivityKey: campaign.ActivityKey, UserId: first.Id, SourceType: ActivityGrantSourceCampaignClaim,
		SourceRef: ActivityGrantSourceRefClaim, Quota: 100, CreatedAt: now - 2,
	}).Error)
	require.NoError(t, DB.Create(&ActivityGrant{
		ActivityKey: campaign.ActivityKey, UserId: second.Id, SourceType: ActivityGrantSourceCampaignClaim,
		SourceRef: ActivityGrantSourceRefClaim, Quota: 100, CreatedAt: now - 1,
	}).Error)
	require.NoError(t, DB.Create(&ActivityGrant{
		ActivityKey: campaign.ActivityKey, UserId: second.Id, SourceType: ActivityGrantSourceRedeem,
		SourceRef: "unrelated-redemption", Quota: 50, CreatedAt: now,
	}).Error)
	require.NoError(t, DB.Create(&ActivityGrant{
		ActivityKey: campaign.ActivityKey, UserId: first.Id, SourceType: ActivityGrantSourceCampaignClaim,
		SourceRef: "unrelated-claim-source", Quota: 50, CreatedAt: now,
	}).Error)

	campaigns, _, err := ListActivityCampaignsPage(context.Background(), 0, 50)
	require.NoError(t, err)
	require.Len(t, campaigns, 1)
	assert.Equal(t, int64(2), campaigns[0].GrantedCount)

	firstPage, nextCursor, err := ListActivityCampaignGrantDetailsPage(context.Background(), campaign, 0, 1)
	require.NoError(t, err)
	require.Len(t, firstPage, 1)
	assert.Equal(t, second.Id, firstPage[0].UserId)
	assert.Equal(t, second.Username, firstPage[0].Username)
	assert.Equal(t, now-1, firstPage[0].GrantedAt)
	assert.NotZero(t, nextCursor)

	secondPage, nextCursor, err := ListActivityCampaignGrantDetailsPage(context.Background(), campaign, nextCursor, 1)
	require.NoError(t, err)
	require.Len(t, secondPage, 1)
	assert.Equal(t, first.Id, secondPage[0].UserId)
	assert.Zero(t, nextCursor)
}

func TestHasUnclaimedActivityCampaignForUserHonorsAudienceStatusAndClaims(t *testing.T) {
	setupActivityGrantTest(t)
	selected := createActivityGrantTestUser(t, "attention-selected", common.UserStatusEnabled, 0)
	other := createActivityGrantTestUser(t, "attention-other", common.UserStatusEnabled, 0)
	now := common.GetTimestamp()
	campaign := &ActivityCampaign{
		ActivityKey: "attention-selected-campaign", Type: ActivityCampaignTypeClaimable,
		AudienceType: ActivityCampaignAudienceSelected, Title: "Selected attention",
		AmountUSD: "0.01", Quota: 100, StartsAt: now - 10, EndsAt: now + 3600, CreatedBy: 1,
	}
	require.NoError(t, CreateActivityCampaignWithRecipients(context.Background(), campaign, []int{selected.Id}))

	hasPending, err := HasUnclaimedActivityCampaignForUser(context.Background(), selected.Id, now)
	require.NoError(t, err)
	assert.True(t, hasPending)
	hasPending, err = HasUnclaimedActivityCampaignForUser(context.Background(), other.Id, now)
	require.NoError(t, err)
	assert.False(t, hasPending)

	_, granted, err := ClaimActivityCampaignQuota(context.Background(), selected.Id, campaign.ActivityKey)
	require.NoError(t, err)
	require.True(t, granted)
	hasPending, err = HasUnclaimedActivityCampaignForUser(context.Background(), selected.Id, now)
	require.NoError(t, err)
	assert.False(t, hasPending)

	for _, status := range []string{ActivityCampaignStatusClosed, ActivityCampaignStatusCompleted} {
		allCampaign := &ActivityCampaign{
			ActivityKey: "attention-" + status, Type: ActivityCampaignTypeClaimable, Status: status,
			AudienceType: ActivityCampaignAudienceAll, Title: status, AmountUSD: "0.01", Quota: 100,
			StartsAt: now - 10, EndsAt: now + 3600, RecipientMaxUserID: other.Id, RecipientCount: 2, CreatedBy: 1,
		}
		require.NoError(t, DB.Create(allCampaign).Error)
	}
	hasPending, err = HasUnclaimedActivityCampaignForUser(context.Background(), other.Id, now)
	require.NoError(t, err)
	assert.False(t, hasPending)
}
