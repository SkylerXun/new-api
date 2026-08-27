package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type hupijiaoStatusResponse struct {
	Success bool `json:"success"`
	Data    struct {
		TradeNo string `json:"trade_no"`
		Status  string `json:"status"`
		Quota   int64  `json:"quota"`
	} `json:"data"`
}

func requestHupijiaoStatus(t *testing.T, userID int, path string, handler gin.HandlerFunc) hupijiaoStatusResponse {
	t.Helper()
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, path, nil)
	c.Set("id", userID)
	handler(c)

	var response hupijiaoStatusResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	return response
}

func TestGetHupijiaoPaymentStatusChecksOwnerAndProvider(t *testing.T) {
	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.TopUp{}))
	require.NoError(t, db.Create(&model.TopUp{
		UserId:          101,
		Amount:          1234,
		TradeNo:         "HUP-STATUS-OWNED",
		PaymentProvider: model.PaymentProviderHupijiao,
		Status:          common.TopUpStatusPending,
	}).Error)
	require.NoError(t, db.Create(&model.TopUp{
		UserId:          101,
		TradeNo:         "EPAY-STATUS-OWNED",
		PaymentProvider: model.PaymentProviderEpay,
		Status:          common.TopUpStatusPending,
	}).Error)

	owned := requestHupijiaoStatus(t, 101, "/api/user/topup/status?trade_no=HUP-STATUS-OWNED", GetHupijiaoPaymentStatus)
	require.True(t, owned.Success)
	require.Equal(t, "HUP-STATUS-OWNED", owned.Data.TradeNo)
	require.Equal(t, common.TopUpStatusPending, owned.Data.Status)
	require.EqualValues(t, 1234, owned.Data.Quota)

	otherUser := requestHupijiaoStatus(t, 202, "/api/user/topup/status?trade_no=HUP-STATUS-OWNED", GetHupijiaoPaymentStatus)
	require.False(t, otherUser.Success)

	wrongProvider := requestHupijiaoStatus(t, 101, "/api/user/topup/status?trade_no=EPAY-STATUS-OWNED", GetHupijiaoPaymentStatus)
	require.False(t, wrongProvider.Success)
}

func TestGetSubscriptionHupijiaoPaymentStatusChecksOwnerAndProvider(t *testing.T) {
	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.SubscriptionOrder{}))
	require.NoError(t, db.Create(&model.SubscriptionOrder{
		UserId:          303,
		PlanId:          1,
		TradeNo:         "HUP-SUB-STATUS-OWNED",
		PaymentProvider: model.PaymentProviderHupijiao,
		Status:          common.TopUpStatusSuccess,
	}).Error)
	require.NoError(t, db.Create(&model.SubscriptionOrder{
		UserId:          303,
		PlanId:          1,
		TradeNo:         "EPAY-SUB-STATUS-OWNED",
		PaymentProvider: model.PaymentProviderEpay,
		Status:          common.TopUpStatusPending,
	}).Error)

	owned := requestHupijiaoStatus(t, 303, "/api/subscription/hupijiao/status?trade_no=HUP-SUB-STATUS-OWNED", GetSubscriptionHupijiaoPaymentStatus)
	require.True(t, owned.Success)
	require.Equal(t, "HUP-SUB-STATUS-OWNED", owned.Data.TradeNo)
	require.Equal(t, common.TopUpStatusSuccess, owned.Data.Status)

	otherUser := requestHupijiaoStatus(t, 404, "/api/subscription/hupijiao/status?trade_no=HUP-SUB-STATUS-OWNED", GetSubscriptionHupijiaoPaymentStatus)
	require.False(t, otherUser.Success)

	wrongProvider := requestHupijiaoStatus(t, 303, "/api/subscription/hupijiao/status?trade_no=EPAY-SUB-STATUS-OWNED", GetSubscriptionHupijiaoPaymentStatus)
	require.False(t, wrongProvider.Success)
}
