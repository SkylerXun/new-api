package model

import (
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRechargeHupijiaoCreditsSnapshotQuotaExactlyOnce(t *testing.T) {
	truncateTables(t)
	user := insertUserForPaymentGuardTest(t, 601, 7)
	originalPrice := operation_setting.Price
	operation_setting.Price = 999
	t.Cleanup(func() { operation_setting.Price = originalPrice })

	order := &TopUp{UserId: user.Id, Amount: 1234, Money: 8, OriginalAmount: 10, DiscountRate: 0.8, ActualAmount: 8, PackageID: "cny10", TradeNo: "HUPIJAO-ONCE", PaymentMethod: "alipay", PaymentProvider: PaymentProviderHupijiao, Status: common.TopUpStatusPending, CreateTime: time.Now().Unix()}
	require.NoError(t, order.Insert())

	alreadyDone, err := RechargeHupijiao(order.TradeNo, "alipay", "127.0.0.1")
	require.NoError(t, err)
	assert.False(t, alreadyDone)
	assert.Equal(t, 1241, getUserQuotaForPaymentGuardTest(t, user.Id))

	alreadyDone, err = RechargeHupijiao(order.TradeNo, "alipay", "127.0.0.1")
	require.NoError(t, err)
	assert.True(t, alreadyDone)
	assert.Equal(t, 1241, getUserQuotaForPaymentGuardTest(t, user.Id))
}

func TestRechargeHupijiaoRejectsCredentialMethodMismatch(t *testing.T) {
	truncateTables(t)
	user := insertUserForPaymentGuardTest(t, 602, 0)
	order := &TopUp{UserId: user.Id, Amount: 500, Money: 5, ActualAmount: 5, PackageID: "cny5", TradeNo: "HUPIJAO-METHOD", PaymentMethod: "wxpay", PaymentProvider: PaymentProviderHupijiao, Status: common.TopUpStatusPending, CreateTime: time.Now().Unix()}
	require.NoError(t, order.Insert())

	_, err := RechargeHupijiao(order.TradeNo, "alipay", "127.0.0.1")
	assert.ErrorIs(t, err, ErrPaymentMethodMismatch)
	assert.Equal(t, 0, getUserQuotaForPaymentGuardTest(t, user.Id))
	assert.Equal(t, common.TopUpStatusPending, getTopUpStatusForPaymentGuardTest(t, order.TradeNo))
}
