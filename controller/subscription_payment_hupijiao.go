package controller

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

func SubscriptionRequestHupijiao(c *gin.Context) {
	if !requirePaymentCompliance(c) {
		return
	}
	if !hupijiaoEnabled() {
		common.ApiErrorMsg(c, "虎皮椒支付未启用")
		return
	}
	var req SubscriptionEpayPayRequest
	if c.ShouldBindJSON(&req) != nil || req.PlanId <= 0 {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	plan, err := model.GetSubscriptionPlanById(req.PlanId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if !plan.Enabled || plan.PriceAmount < 0.01 {
		common.ApiErrorMsg(c, "套餐不可用")
		return
	}
	appid, secret, _ := hupijiaoCredential(req.PaymentMethod)
	if appid == "" || secret == "" {
		common.ApiErrorMsg(c, "支付凭证未配置")
		return
	}
	rate := plan.HupijiaoDiscountRate
	if rate < 0.01 || rate > 1 {
		rate = 1
	}
	actual := decimal.NewFromFloat(plan.PriceAmount).Mul(decimal.NewFromFloat(rate)).Round(2).InexactFloat64()
	userId := c.GetInt("id")
	if rejectSubscriptionPurchaseWhenActive(c, userId) {
		return
	}
	if plan.MaxPurchasePerUser > 0 {
		count, countErr := model.CountUserSubscriptionsByPlan(userId, plan.Id)
		if countErr != nil {
			common.ApiError(c, countErr)
			return
		}
		if count >= int64(plan.MaxPurchasePerUser) {
			common.ApiErrorMsg(c, "已达到该套餐购买上限")
			return
		}
	}
	tradeNo := fmt.Sprintf("SUBUSR%dNO%s%d", userId, common.GetRandomString(6), time.Now().Unix())
	callback := service.GetCallbackAddress()
	values := map[string]string{"version": "1.1", "appid": appid, "trade_order_id": tradeNo, "total_fee": strconv.FormatFloat(actual, 'f', 2, 64), "title": plan.Title, "time": strconv.FormatInt(time.Now().Unix(), 10), "notify_url": callback + "/api/subscription/hupijiao/notify", "return_url": callback + "/api/subscription/hupijiao/return", "callback_url": callback + "/api/subscription/hupijiao/return", "plugins": "new-api", "attach": strconv.Itoa(plan.Id), "nonce_str": common.GetRandomString(16)}
	order := &model.SubscriptionOrder{UserId: userId, PlanId: plan.Id, Money: actual, OriginalMoney: plan.PriceAmount, TradeNo: tradeNo, PaymentMethod: req.PaymentMethod, PaymentProvider: model.PaymentProviderHupijiao, CreateTime: time.Now().Unix(), Status: common.TopUpStatusPending}
	if err := order.Insert(); err != nil {
		common.ApiErrorMsg(c, "创建订单失败")
		return
	}
	result, err := hupijiaoRequest(values, secret)
	if err != nil {
		_ = model.ExpireSubscriptionOrder(tradeNo, model.PaymentProviderHupijiao)
		logger.LogError(c.Request.Context(), fmt.Sprintf("虎皮椒订阅下单失败 trade_no=%s error=%q", tradeNo, err.Error()))
		common.ApiErrorMsg(c, "虎皮椒下单失败："+err.Error())
		return
	}
	common.ApiSuccess(c, gin.H{"redirect_url": result["url"], "qrcode_url": result["url_qrcode"], "trade_no": tradeNo, "original_amount": plan.PriceAmount, "actual_amount": actual})
}

func SubscriptionHupijiaoNotify(c *gin.Context) { subscriptionHupijiaoCallback(c, false) }
func SubscriptionHupijiaoReturn(c *gin.Context) { subscriptionHupijiaoCallback(c, true) }

func GetSubscriptionHupijiaoPaymentStatus(c *gin.Context) {
	tradeNo := strings.TrimSpace(c.Query("trade_no"))
	if tradeNo == "" {
		common.ApiErrorMsg(c, "缺少订单号")
		return
	}
	order := model.GetSubscriptionOrderByTradeNo(tradeNo)
	if order == nil || order.UserId != c.GetInt("id") || order.PaymentProvider != model.PaymentProviderHupijiao {
		common.ApiErrorMsg(c, "订单不存在")
		return
	}
	common.ApiSuccess(c, gin.H{
		"trade_no": tradeNo,
		"status":   order.Status,
	})
}

func subscriptionHupijiaoCallback(c *gin.Context, redirect bool) {
	_ = c.Request.ParseForm()
	params := map[string]string{}
	for k, v := range c.Request.Form {
		if len(v) > 0 {
			params[k] = v[0]
		}
	}
	appid := params["appid"]
	method := ""
	if appid == operation_setting.HupijiaoWechatAppID {
		method = "wxpay"
	}
	if appid == operation_setting.HupijiaoAlipayAppID {
		method = "alipay"
	}
	_, secret, _ := hupijiaoCredential(method)
	authenticated := method != "" && secret != "" && params["hash"] == hupijiaoSign(params, secret)
	tradeNo := params["trade_order_id"]
	orderValid := false
	if authenticated {
		order := model.GetSubscriptionOrderByTradeNo(tradeNo)
		callbackAmount, parseErr := decimal.NewFromString(params["total_fee"])
		orderValid = order != nil && parseErr == nil && order.PaymentProvider == model.PaymentProviderHupijiao && order.PaymentMethod == method && callbackAmount.Round(2).Equal(decimal.NewFromFloat(order.Money).Round(2))
	}
	accepted := authenticated && orderValid
	if accepted && params["status"] == "OD" {
		accepted = model.CompleteSubscriptionOrder(tradeNo, common.GetJsonString(params), model.PaymentProviderHupijiao, method) == nil
	}
	if redirect {
		result := "fail"
		if accepted && params["status"] != "OD" {
			result = "pending"
		}
		if accepted && params["status"] == "OD" {
			result = "success"
		}
		target := "/my-subscriptions?pay=" + result
		c.Redirect(http.StatusFound, paymentReturnPath(target))
		return
	}
	if accepted {
		_, _ = c.Writer.Write([]byte("success"))
	} else {
		_, _ = c.Writer.Write([]byte("fail"))
	}
}
