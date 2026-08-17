package model

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdvanceUserBillingCurveProgressIgnoresBalanceChanges(t *testing.T) {
	truncateTables(t)
	require.NoError(t, DB.AutoMigrate(&UserBillingCurveProgress{}))
	DB.Exec("DELETE FROM user_billing_curve_progress")

	user := &User{
		Username: fmt.Sprintf("curve-progress-%d", time.Now().UnixNano()),
		Password: "test-password",
	}
	require.NoError(t, DB.Create(user).Error)
	t.Cleanup(func() {
		DB.Where("user_id = ?", user.Id).Delete(&UserBillingCurveProgress{})
	})

	before, after, err := AdvanceUserBillingCurveProgress(user.Id, 125)
	require.NoError(t, err)
	assert.Equal(t, int64(0), before)
	assert.Equal(t, int64(125), after)

	require.NoError(t, IncreaseUserQuota(user.Id, 5_000, true))
	require.NoError(t, DecreaseUserQuota(user.Id, 2_000, true))

	before, after, err = AdvanceUserBillingCurveProgress(user.Id, 75)
	require.NoError(t, err)
	assert.Equal(t, int64(125), before)
	assert.Equal(t, int64(200), after)
}
