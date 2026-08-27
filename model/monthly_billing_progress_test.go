package model

import (
	"fmt"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMonthlyBillingProgressSeparatesMonthsAndSupportsRefund(t *testing.T) {
	truncateTables(t)
	require.NoError(t, DB.AutoMigrate(&UserMonthlyBillingProgress{}, &UserMonthlyBillingSettlement{}))
	require.NoError(t, DB.Exec("DELETE FROM user_monthly_billing_settlements").Error)
	require.NoError(t, DB.Exec("DELETE FROM user_monthly_billing_progress").Error)
	user := &User{Username: fmt.Sprintf("monthly-progress-%d", time.Now().UnixNano()), Password: "test-password"}
	require.NoError(t, DB.Create(user).Error)

	monthOne := int64(1_780_275_600)
	monthTwo := int64(1_782_867_600)
	before, after, err := UpdateUserMonthlyBillingProgress(user.Id, monthOne, func(current int64) (int64, error) { return current + 100, nil })
	require.NoError(t, err)
	assert.Equal(t, int64(0), before)
	assert.Equal(t, int64(100), after)
	_, after, err = UpdateUserMonthlyBillingProgress(user.Id, monthTwo, func(current int64) (int64, error) { return current + 40, nil })
	require.NoError(t, err)
	assert.Equal(t, int64(40), after)

	require.NoError(t, RevertUserMonthlyBillingProgress(user.Id, monthOne, 30))
	monthOneSpent, err := GetUserMonthlyBillingProgress(user.Id, monthOne)
	require.NoError(t, err)
	monthTwoSpent, err := GetUserMonthlyBillingProgress(user.Id, monthTwo)
	require.NoError(t, err)
	assert.Equal(t, int64(70), monthOneSpent)
	assert.Equal(t, int64(40), monthTwoSpent)
}

func TestMonthlyBillingSettlementIsIdempotentAndRefundsOnce(t *testing.T) {
	truncateTables(t)
	require.NoError(t, DB.AutoMigrate(&UserMonthlyBillingProgress{}, &UserMonthlyBillingSettlement{}))
	require.NoError(t, DB.Exec("DELETE FROM user_monthly_billing_settlements").Error)
	require.NoError(t, DB.Exec("DELETE FROM user_monthly_billing_progress").Error)
	user := &User{Username: fmt.Sprintf("monthly-settlement-%d", time.Now().UnixNano()), Password: "test-password"}
	require.NoError(t, DB.Create(user).Error)
	monthStart := int64(1_780_275_600)
	calls := 0
	settle := func(current int64) (int64, int, error) { calls++; return current + 90_000_000, 45_000_000, nil }
	before, after, charged, settledMonth, err := SettleUserMonthlyBillingProgress(user.Id, monthStart, "task:one", 50_000_000, settle)
	require.NoError(t, err)
	assert.Equal(t, int64(0), before)
	assert.Equal(t, int64(90_000_000), after)
	assert.Equal(t, 45_000_000, charged)
	assert.Equal(t, monthStart, settledMonth)
	_, _, charged, settledMonth, err = SettleUserMonthlyBillingProgress(user.Id, monthStart+1, "task:one", 50_000_000, settle)
	require.NoError(t, err)
	assert.Equal(t, 45_000_000, charged)
	assert.Equal(t, monthStart, settledMonth)
	assert.Equal(t, 1, calls)

	require.NoError(t, RefundUserMonthlyBillingSettlement("task:one"))
	require.NoError(t, RefundUserMonthlyBillingSettlement("task:one"))
	spent, err := GetUserMonthlyBillingProgress(user.Id, monthStart)
	require.NoError(t, err)
	assert.Equal(t, int64(0), spent)
}

func TestEnsureUserMonthlyBillingBackfillUsesNetHistoricalChargeOnce(t *testing.T) {
	truncateTables(t)
	require.NoError(t, DB.AutoMigrate(&UserMonthlyBillingProgress{}, &UserMonthlyBillingSettlement{}))
	require.NoError(t, DB.Exec("DELETE FROM user_monthly_billing_settlements").Error)
	require.NoError(t, DB.Exec("DELETE FROM user_monthly_billing_progress").Error)
	user := &User{Username: fmt.Sprintf("monthly-backfill-%d", time.Now().UnixNano()), Password: "test-password", Quota: 12_345}
	require.NoError(t, DB.Create(user).Error)

	originalQuotaPerUnit := common.QuotaPerUnit
	common.QuotaPerUnit = 500_000
	t.Cleanup(func() { common.QuotaPerUnit = originalQuotaPerUnit })
	monthStart := int64(1_780_275_600)
	cutoff := monthStart + 10_000
	require.NoError(t, DB.Create(&[]Log{
		{UserId: user.Id, CreatedAt: monthStart + 1, Type: LogTypeConsume, Quota: 500_000},
		{UserId: user.Id, CreatedAt: monthStart + 2, Type: LogTypeConsume, Quota: 250_000},
		{UserId: user.Id, CreatedAt: monthStart + 3, Type: LogTypeRefund, Quota: 100_000},
		{UserId: user.Id, CreatedAt: cutoff, Type: LogTypeConsume, Quota: 999_999},
	}).Error)

	require.NoError(t, EnsureUserMonthlyBillingBackfill([]int{user.Id}, monthStart, cutoff))
	require.NoError(t, EnsureUserMonthlyBillingBackfill([]int{user.Id}, monthStart, cutoff))
	spent, err := GetUserMonthlyBillingProgress(user.Id, monthStart)
	require.NoError(t, err)
	assert.Equal(t, int64(1_300_000), spent)

	var progress UserMonthlyBillingProgress
	require.NoError(t, DB.Where("user_id = ? AND month_start = ?", user.Id, monthStart).First(&progress).Error)
	assert.Equal(t, cutoff, progress.BackfillCutoff)
	quota, err := GetUserQuota(user.Id, false)
	require.NoError(t, err)
	assert.Equal(t, 12_345, quota)
	var logCount int64
	require.NoError(t, DB.Model(&Log{}).Where("user_id = ?", user.Id).Count(&logCount).Error)
	assert.Equal(t, int64(4), logCount)
}
