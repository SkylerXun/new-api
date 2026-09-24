package model

import (
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
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

func TestRechargeHupijiaoCreditsNewUserBonusExactlyOnce(t *testing.T) {
	truncateTables(t)
	require.NoError(t, DB.AutoMigrate(&ActivityGrant{}))
	require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&ActivityGrant{}).Error)
	activitySetting := operation_setting.GetActivitySetting()
	previous := *activitySetting
	activitySetting.NewUserRedeemBonusEnabled = true
	activitySetting.NewUserRedeemBonusPercent = 25
	activitySetting.NewUserRedeemBonusWindowDays = 1
	t.Cleanup(func() {
		*activitySetting = previous
		DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&ActivityGrant{})
	})
	user := insertUserForPaymentGuardTest(t, 603, 0)
	firstOrder := &TopUp{UserId: user.Id, Amount: 1000, Money: 8, TradeNo: "HUPIJAO-BONUS-FIRST", PaymentMethod: "alipay", PaymentProvider: PaymentProviderHupijiao, Status: common.TopUpStatusPending, CreateTime: time.Now().Unix()}
	require.NoError(t, firstOrder.Insert())
	_, err := RechargeHupijiao(firstOrder.TradeNo, "alipay", "127.0.0.1")
	require.NoError(t, err)
	assert.Equal(t, 1250, getUserQuotaForPaymentGuardTest(t, user.Id))
	grant, err := GetActivityGrantForUserSource(user.Id, ActivityKeyNewUserRedeemBonus, "topup:"+firstOrder.TradeNo)
	require.NoError(t, err)
	require.NotNil(t, grant)
	assert.Equal(t, 250, grant.Quota)

	secondOrder := &TopUp{UserId: user.Id, Amount: 500, Money: 4, TradeNo: "HUPIJAO-BONUS-SECOND", PaymentMethod: "alipay", PaymentProvider: PaymentProviderHupijiao, Status: common.TopUpStatusPending, CreateTime: time.Now().Unix()}
	require.NoError(t, secondOrder.Insert())
	_, err = RechargeHupijiao(secondOrder.TradeNo, "alipay", "127.0.0.1")
	require.NoError(t, err)
	assert.Equal(t, 1750, getUserQuotaForPaymentGuardTest(t, user.Id))
	secondGrant, err := GetActivityGrantForUserSource(user.Id, ActivityKeyNewUserRedeemBonus, "topup:"+secondOrder.TradeNo)
	require.NoError(t, err)
	assert.Nil(t, secondGrant)

	_, err = RechargeHupijiao(secondOrder.TradeNo, "alipay", "127.0.0.1")
	require.NoError(t, err)
	assert.Equal(t, 1750, getUserQuotaForPaymentGuardTest(t, user.Id))
}

func TestRechargeHupijiaoCreditsInviterForEveryNewOrder(t *testing.T) {
	truncateTables(t)
	affiliateSetting := operation_setting.GetAffiliateSetting()
	previousAffiliate := *affiliateSetting
	activitySetting := operation_setting.GetActivitySetting()
	previousActivity := *activitySetting
	affiliateSetting.RedeemRebateEnabled = true
	affiliateSetting.RedeemRebatePercent = 10
	activitySetting.NewUserRedeemBonusEnabled = false
	t.Cleanup(func() {
		*affiliateSetting = previousAffiliate
		*activitySetting = previousActivity
	})

	inviter := &User{Username: "hupijiao-affiliate-inviter", Password: "password", Status: common.UserStatusEnabled}
	require.NoError(t, DB.Create(inviter).Error)
	invitee := &User{Username: "hupijiao-affiliate-invitee", Password: "password", Status: common.UserStatusEnabled, InviterId: inviter.Id}
	require.NoError(t, DB.Create(invitee).Error)

	firstOrder := &TopUp{UserId: invitee.Id, Amount: 1000, Money: 8, TradeNo: "HUPIJAO-REBATE-FIRST", PaymentMethod: "alipay", PaymentProvider: PaymentProviderHupijiao, Status: common.TopUpStatusPending, CreateTime: time.Now().Unix()}
	secondOrder := &TopUp{UserId: invitee.Id, Amount: 500, Money: 4, TradeNo: "HUPIJAO-REBATE-SECOND", PaymentMethod: "alipay", PaymentProvider: PaymentProviderHupijiao, Status: common.TopUpStatusPending, CreateTime: time.Now().Unix()}
	require.NoError(t, firstOrder.Insert())
	require.NoError(t, secondOrder.Insert())

	_, err := RechargeHupijiao(firstOrder.TradeNo, "alipay", "127.0.0.1")
	require.NoError(t, err)
	_, err = RechargeHupijiao(secondOrder.TradeNo, "alipay", "127.0.0.1")
	require.NoError(t, err)
	_, err = RechargeHupijiao(secondOrder.TradeNo, "alipay", "127.0.0.1")
	require.NoError(t, err)

	assert.Equal(t, 1500, getUserQuotaForPaymentGuardTest(t, invitee.Id))
	storedInviter := loadAffiliateTestUser(t, inviter.Id)
	assert.Equal(t, 150, storedInviter.AffQuota)
	assert.Equal(t, 150, storedInviter.AffHistoryQuota)
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
