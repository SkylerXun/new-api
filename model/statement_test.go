package model

import (
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupStatementTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	previousDB, previousLogDB := DB, LOG_DB
	previousMainType, previousLogType := common.MainDatabaseType(), common.LogDatabaseType()
	db, err := gorm.Open(sqlite.Open("file:"+strings.ReplaceAll(t.Name(), "/", "_")+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	DB, LOG_DB = db, db
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	require.NoError(t, db.AutoMigrate(
		&User{}, &Option{}, &Log{}, &Redemption{}, &RedemptionCategory{}, &RedemptionPricingAudit{},
		&TopUp{}, &StatementUsageMonthly{}, &ConsumptionStatement{},
	))
	t.Cleanup(func() {
		DB, LOG_DB = previousDB, previousLogDB
		common.SetDatabaseTypes(previousMainType, previousLogType)
		sqlDB, sqlErr := db.DB()
		if sqlErr == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

func TestStatementMonthBoundsUsesShanghaiAndRejectsFuture(t *testing.T) {
	location := time.FixedZone("Asia/Shanghai", 8*60*60)
	now := time.Date(2026, time.August, 31, 23, 59, 30, 0, location)
	start, end, current, err := StatementMonthBounds("2026-08", now)
	require.NoError(t, err)
	assert.True(t, current)
	assert.Equal(t, "2026-08-01 00:00:00", start.Format("2006-01-02 15:04:05"))
	assert.Equal(t, now.Unix(), end.Unix())

	start, end, current, err = StatementMonthBounds("2026-07", now)
	require.NoError(t, err)
	assert.False(t, current)
	assert.Equal(t, "2026-07-01 00:00:00", start.Format("2006-01-02 15:04:05"))
	assert.Equal(t, "2026-08-01 00:00:00", end.Format("2006-01-02 15:04:05"))

	_, _, _, err = StatementMonthBounds("2026-09", now)
	assert.Error(t, err)
}

func TestStatementUsageHistoricalBackfillRunsOnce(t *testing.T) {
	db := setupStatementTestDB(t)
	location := time.FixedZone("Asia/Shanghai", 8*60*60)
	monthStart := time.Date(2026, time.July, 1, 0, 0, 0, 0, location).Unix()
	user := User{Username: "backfill-user", Password: "unused", AffCode: "statement-backfill"}
	require.NoError(t, db.Create(&user).Error)
	require.NoError(t, db.Create(&[]Log{
		{UserId: user.Id, CreatedAt: monthStart + 1, Type: LogTypeConsume, ModelName: "backfill-model", PromptTokens: 10, CompletionTokens: 2},
		{UserId: user.Id, CreatedAt: monthStart + 2, Type: LogTypeConsume, ModelName: "backfill-model", PromptTokens: 20, CompletionTokens: 3},
	}).Error)

	require.NoError(t, BackfillStatementUsageFromLogsOnce())
	var usage StatementUsageMonthly
	require.NoError(t, db.Where("user_id = ? AND month_start = ? AND model_name = ?", user.Id, monthStart, "backfill-model").First(&usage).Error)
	assert.EqualValues(t, 30, usage.InputTokens)
	assert.EqualValues(t, 5, usage.OutputTokens)
	assert.EqualValues(t, 2, usage.BillingCount)

	require.NoError(t, db.Create(&Log{UserId: user.Id, CreatedAt: monthStart + 3, Type: LogTypeConsume, ModelName: "backfill-model", PromptTokens: 99}).Error)
	require.NoError(t, BackfillStatementUsageFromLogsOnce())
	require.NoError(t, db.Where("user_id = ? AND month_start = ? AND model_name = ?", user.Id, monthStart, "backfill-model").First(&usage).Error)
	assert.EqualValues(t, 30, usage.InputTokens)
}

func TestGenerateConsumptionStatementFiltersAmountsAndFreezesSnapshot(t *testing.T) {
	db := setupStatementTestDB(t)
	location := time.FixedZone("Asia/Shanghai", 8*60*60)
	now := time.Date(2026, time.August, 20, 12, 0, 0, 0, location)
	monthStart := time.Date(2026, time.August, 1, 0, 0, 0, 0, location).Unix()
	user := User{Username: "statement-user", Email: "statement@example.com", Password: "unused"}
	require.NoError(t, db.Create(&user).Error)
	require.NoError(t, db.Create(&[]Log{
		{UserId: user.Id, CreatedAt: monthStart + 10, Type: LogTypeConsume, ModelName: "gpt-test", PromptTokens: 120, CompletionTokens: 30},
		{UserId: user.Id, CreatedAt: monthStart + 20, Type: LogTypeConsume, ModelName: "gpt-test", PromptTokens: 80, CompletionTokens: 20},
		{UserId: user.Id, CreatedAt: now.Unix(), Type: LogTypeConsume, ModelName: "excluded-at-right-bound", PromptTokens: 999},
	}).Error)
	require.NoError(t, db.Create(&[]Redemption{
		{Key: "11111111111111111111111111111111", Status: common.RedemptionCodeStatusUsed, UsedUserId: user.Id, RedeemedTime: monthStart + 30, CategoryID: 1, CategoryNameSnapshot: "会员兑换", CategoryPriceCents: 1999, CategoryPricedAt: monthStart},
		{Key: "22222222222222222222222222222222", Status: common.RedemptionCodeStatusUsed, UsedUserId: user.Id, RedeemedTime: monthStart + 40},
	}).Error)
	require.NoError(t, db.Create(&[]TopUp{
		{UserId: user.Id, Amount: 100, Money: 8.88, PaidCurrency: "CNY", PaidAmountMinor: 888, Status: common.TopUpStatusSuccess, CompleteTime: monthStart + 50, TradeNo: "cny-ok"},
		{UserId: user.Id, Amount: 100, Money: 9.99, Status: common.TopUpStatusSuccess, CompleteTime: monthStart + 60, TradeNo: "unknown"},
		{UserId: user.Id, Amount: 100, Money: 10, PaidCurrency: "USD", PaidAmountMinor: 1000, Status: common.TopUpStatusSuccess, CompleteTime: monthStart + 70, TradeNo: "usd"},
		{UserId: user.Id, Amount: 0, Money: 66, PaidCurrency: "CNY", PaidAmountMinor: 6600, Status: common.TopUpStatusSuccess, CompleteTime: monthStart + 80, TradeNo: "subscription-mirror"},
	}).Error)

	statement, err := GenerateConsumptionStatement(StatementGenerateInput{
		UserID: user.Id, Source: StatementSourceUserExport, GeneratedBy: user.Id,
		BillingTitle: "广东研发部", BillingAddress: "广东省深圳市", Now: now,
	})
	require.NoError(t, err)
	require.Len(t, statement.Snapshot.Tokens, 1)
	assert.EqualValues(t, 200, statement.Snapshot.Tokens[0].InputTokens)
	assert.EqualValues(t, 50, statement.Snapshot.Tokens[0].OutputTokens)
	assert.EqualValues(t, 2, statement.Snapshot.Tokens[0].RecordCount)
	assert.EqualValues(t, 1999, statement.Snapshot.RedemptionTotalCents)
	assert.EqualValues(t, 888, statement.Snapshot.TopUpTotalCents)
	assert.EqualValues(t, 2887, statement.Snapshot.TotalCents)
	assert.EqualValues(t, 1, statement.Snapshot.Warnings.UnpricedRedemptions)
	assert.EqualValues(t, 1, statement.Snapshot.Warnings.UnknownCurrencyTopUps)
	assert.Equal(t, "广东研发部", statement.Snapshot.Recipient.BillingTitle)
	assert.True(t, statement.Snapshot.Recipient.UserSupplied)
	assert.Len(t, statement.ContentHash, 64)

	lowerSnapshot := strings.ToLower(statement.SnapshotJSON)
	for _, forbidden := range []string{"quota", "usd", "payment_provider", "channel_id", "upstream"} {
		assert.NotContains(t, lowerSnapshot, forbidden)
	}

	reloaded, err := GetConsumptionStatement(statement.ID)
	require.NoError(t, err)
	assert.Equal(t, statement.ContentHash, reloaded.ContentHash)
	assert.Equal(t, statement.Snapshot.TotalCents, reloaded.Snapshot.TotalCents)
	assert.True(t, CanAccessConsumptionStatement(statement, user.Id, common.RoleCommonUser, now))
	assert.False(t, CanAccessConsumptionStatement(statement, user.Id+1, common.RoleCommonUser, now))
	assert.True(t, CanAccessConsumptionStatement(statement, user.Id+1, common.RoleAdminUser, now))
	assert.False(t, CanAccessConsumptionStatement(statement, user.Id, common.RoleCommonUser, time.Date(2026, time.September, 1, 0, 0, 0, 0, location)))
}

func TestRedemptionCategorySnapshotAndLegacyAssignmentAreImmutable(t *testing.T) {
	db := setupStatementTestDB(t)
	category := RedemptionCategory{Name: "付费类别", PriceCents: 2500}
	require.NoError(t, CreateRedemptionCategory(&category))
	redemption := Redemption{Key: "33333333333333333333333333333333", CategoryID: category.ID, CategoryNameSnapshot: category.Name, CategoryPriceCents: category.PriceCents, CategoryPricedAt: common.GetTimestamp()}
	require.NoError(t, db.Create(&redemption).Error)
	category.PriceCents = 9900
	require.NoError(t, UpdateRedemptionCategory(&category))
	var unchanged Redemption
	require.NoError(t, db.First(&unchanged, redemption.Id).Error)
	assert.EqualValues(t, 2500, unchanged.CategoryPriceCents)

	legacy := Redemption{Key: "44444444444444444444444444444444", Status: common.RedemptionCodeStatusUsed, UsedUserId: 9, RedeemedTime: common.GetTimestamp()}
	require.NoError(t, db.Create(&legacy).Error)
	assigned, err := AssignRedemptionCategory([]int{legacy.Id}, category.ID, 100)
	require.NoError(t, err)
	assert.EqualValues(t, 1, assigned)
	_, err = AssignRedemptionCategory([]int{legacy.Id}, category.ID, 101)
	assert.Error(t, err)
	var audits int64
	require.NoError(t, db.Model(&RedemptionPricingAudit{}).Where("redemption_id = ?", legacy.Id).Count(&audits).Error)
	assert.EqualValues(t, 1, audits)
}

func TestSystemMonthlyStatementIsIdempotentFinalVersion(t *testing.T) {
	db := setupStatementTestDB(t)
	location := time.FixedZone("Asia/Shanghai", 8*60*60)
	now := time.Date(2026, time.August, 5, 0, 0, 0, 0, location)
	user := User{Username: "monthly-user", Password: "unused"}
	require.NoError(t, db.Create(&user).Error)
	require.NoError(t, db.Create(&StatementUsageMonthly{UserID: user.Id, MonthStart: time.Date(2026, time.July, 1, 0, 0, 0, 0, location).Unix(), ModelName: "model", InputTokens: 1, BillingCount: 1}).Error)
	first, err := GenerateConsumptionStatement(StatementGenerateInput{UserID: user.Id, Month: "2026-07", Source: StatementSourceSystemMonthly, Now: now})
	require.NoError(t, err)
	second, err := GenerateConsumptionStatement(StatementGenerateInput{UserID: user.Id, Month: "2026-07", Source: StatementSourceSystemMonthly, Now: now.Add(time.Hour)})
	require.NoError(t, err)
	assert.Equal(t, first.ID, second.ID)
	var count int64
	require.NoError(t, db.Model(&ConsumptionStatement{}).Count(&count).Error)
	assert.EqualValues(t, 1, count)
}

func TestClosedMonthCreatesFinalForEveryUserAndPreviousMonthAccessIsLimited(t *testing.T) {
	db := setupStatementTestDB(t)
	location := time.FixedZone("Asia/Shanghai", 8*60*60)
	now := time.Date(2026, time.September, 1, 0, 0, 10, 0, location)
	users := []User{
		{Username: "active-monthly-user", Password: "unused", Status: common.UserStatusEnabled, AffCode: "monthly-active"},
		{Username: "zero-activity-user", Password: "unused", Status: common.UserStatusEnabled, AffCode: "monthly-zero"},
	}
	require.NoError(t, db.Create(&users).Error)
	require.NoError(t, db.Create(&StatementUsageMonthly{
		UserID: users[0].Id, MonthStart: time.Date(2026, time.August, 1, 0, 0, 0, 0, location).Unix(),
		ModelName: "model", InputTokens: 10, BillingCount: 1,
	}).Error)

	generated, skipped, err := GenerateClosedMonthStatements("2026-08", now)
	require.NoError(t, err)
	assert.Equal(t, 2, generated)
	assert.Zero(t, skipped)

	generated, skipped, err = GenerateClosedMonthStatements("2026-08", now.Add(time.Minute))
	require.NoError(t, err)
	assert.Zero(t, generated)
	assert.Equal(t, 2, skipped)

	previous, err := GetPreviousMonthSystemStatement(users[1].Id, now)
	require.NoError(t, err)
	assert.Equal(t, StatementSourceSystemMonthly, previous.Source)
	assert.True(t, previous.IsFinal)
	assert.Equal(t, time.Date(2026, time.September, 1, 0, 0, 0, 0, location).Unix(), previous.PeriodEnd)
	assert.True(t, CanAccessConsumptionStatement(previous, users[1].Id, common.RoleCommonUser, now))
	assert.False(t, CanAccessConsumptionStatement(previous, users[0].Id, common.RoleCommonUser, now))
	assert.True(t, CanAccessConsumptionStatement(previous, users[0].Id, common.RoleAdminUser, now))
	assert.False(t, CanAccessConsumptionStatement(previous, users[1].Id, common.RoleCommonUser, now.AddDate(0, 1, 0)))
}
