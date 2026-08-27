package controller

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

type HupijiaoPackage struct {
	operation_setting.HupijiaoPackageConfig
	ActualAmount float64 `json:"actual_amount,omitempty"`
}

func hupijiaoSign(values map[string]string, secret string) string {
	keys := make([]string, 0, len(values))
	for k, v := range values {
		if k != "hash" && strings.TrimSpace(v) != "" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	var b strings.Builder
	for i, k := range keys {
		if i > 0 {
			b.WriteByte('&')
		}
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(values[k])
	}
	b.WriteString(secret)
	sum := md5.Sum([]byte(b.String()))
	return hex.EncodeToString(sum[:])
}

func parseHupijiaoPackages() ([]HupijiaoPackage, error) {
	configured, err := operation_setting.ParseHupijiaoPackages(operation_setting.HupijiaoPackages)
	if err != nil {
		return nil, err
	}
	packages := make([]HupijiaoPackage, len(configured))
	for i, config := range configured {
		packages[i].HupijiaoPackageConfig = config
		p := &packages[i]
		p.ActualAmount = decimal.NewFromFloat(p.OriginalAmount).Mul(decimal.NewFromFloat(p.DiscountRate)).Round(2).InexactFloat64()
		if p.ActualAmount < 0.01 {
			return nil, errors.New("虎皮椒套餐支付金额无效")
		}
	}
	return packages, nil
}

func hupijiaoCredential(method string) (string, string, error) {
	if method == "wxpay" {
		return operation_setting.HupijiaoWechatAppID, operation_setting.HupijiaoWechatSecret, nil
	}
	if method == "alipay" {
		return operation_setting.HupijiaoAlipayAppID, operation_setting.HupijiaoAlipaySecret, nil
	}
	return "", "", errors.New("支付方式不存在")
}

func hupijiaoRequest(values map[string]string, secret string) (map[string]string, error) {
	values["hash"] = hupijiaoSign(values, secret)
	body := url.Values{}
	for k, v := range values {
		body.Set(k, v)
	}
	addresses := []string{strings.TrimRight(operation_setting.HupijiaoAPIAddress, "/")}
	if addresses[0] == "" {
		addresses[0] = "https://api.xunhupay.com"
	}
	if backup := strings.TrimRight(operation_setting.HupijiaoBackupAPIAddress, "/"); backup != "" && backup != addresses[0] {
		addresses = append(addresses, backup)
	}
	var last error
	for _, base := range addresses {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		req, reqErr := http.NewRequestWithContext(ctx, http.MethodPost, base+"/payment/do.html", strings.NewReader(body.Encode()))
		if reqErr != nil {
			cancel()
			last = reqErr
			continue
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			last = err
			cancel()
			continue
		}
		data, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		cancel()
		if readErr != nil || resp.StatusCode >= 500 {
			last = readErr
			if last == nil {
				last = fmt.Errorf("http %d", resp.StatusCode)
			}
			continue
		}
		var result map[string]interface{}
		if err := common.Unmarshal(data, &result); err != nil {
			last = err
			continue
		}
		if code, exists := result["errcode"]; exists && fmt.Sprint(code) != "0" {
			return nil, fmt.Errorf("虎皮椒: %v", result["errmsg"])
		}
		out := map[string]string{}
		for k, v := range result {
			if s, ok := v.(string); ok {
				out[k] = s
			}
		}
		return out, nil
	}
	return nil, last
}

type HupijiaoPayRequest struct {
	PackageID     string `json:"package_id"`
	PaymentMethod string `json:"payment_method"`
	Device        string `json:"device"`
}

func hupijiaoEnabled() bool {
	return isPaymentComplianceConfirmed() && operation_setting.OnlinePaymentProvider == model.PaymentProviderHupijiao
}

func RequestHupijiao(c *gin.Context) {
	if !hupijiaoEnabled() {
		common.ApiErrorMsg(c, "虎皮椒支付未启用")
		return
	}
	var req HupijiaoPayRequest
	if c.ShouldBindJSON(&req) != nil || req.PackageID == "" {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	packages, err := parseHupijiaoPackages()
	if err != nil {
		common.ApiErrorMsg(c, "套餐配置无效")
		return
	}
	var selected *HupijiaoPackage
	for i := range packages {
		if packages[i].ID == req.PackageID {
			selected = &packages[i]
			break
		}
	}
	if selected == nil || !selected.Enabled {
		common.ApiErrorMsg(c, "套餐不存在或已停用")
		return
	}
	appid, secret, _ := hupijiaoCredential(req.PaymentMethod)
	if appid == "" || secret == "" {
		common.ApiErrorMsg(c, "支付凭证未配置")
		return
	}
	actual := decimal.NewFromFloat(selected.OriginalAmount).Mul(decimal.NewFromFloat(selected.DiscountRate)).Round(2).InexactFloat64()
	if actual < 0.01 {
		common.ApiErrorMsg(c, "支付金额过低")
		return
	}
	id := c.GetInt("id")
	tradeNo := fmt.Sprintf("USR%dNO%s%d", id, common.GetRandomString(6), time.Now().Unix())
	callback := service.GetCallbackAddress()
	values := map[string]string{"version": "1.1", "appid": appid, "trade_order_id": tradeNo, "total_fee": strconv.FormatFloat(actual, 'f', 2, 64), "title": selected.Title, "time": strconv.FormatInt(time.Now().Unix(), 10), "notify_url": callback + "/api/user/hupijiao/notify", "return_url": callback + "/api/user/hupijiao/return", "callback_url": callback + "/api/user/hupijiao/return", "plugins": "new-api", "attach": selected.ID, "nonce_str": common.GetRandomString(16)}
	topup := &model.TopUp{UserId: id, Amount: selected.Quota, Money: actual, OriginalAmount: selected.OriginalAmount, DiscountRate: selected.DiscountRate, ActualAmount: actual, PackageID: selected.ID, TradeNo: tradeNo, PaymentMethod: req.PaymentMethod, PaymentProvider: model.PaymentProviderHupijiao, CreateTime: time.Now().Unix(), Status: common.TopUpStatusPending}
	if err := topup.Insert(); err != nil {
		common.ApiErrorMsg(c, "创建订单失败")
		return
	}
	result, err := hupijiaoRequest(values, secret)
	if err != nil {
		_ = model.UpdatePendingTopUpStatus(tradeNo, model.PaymentProviderHupijiao, common.TopUpStatusExpired)
		common.ApiErrorMsg(c, "拉起支付失败")
		return
	}
	common.ApiSuccess(c, gin.H{"redirect_url": result["url"], "qrcode_url": result["url_qrcode"], "trade_no": tradeNo, "package_id": selected.ID, "original_amount": selected.OriginalAmount, "actual_amount": actual, "quota": selected.Quota})
}

func HupijiaoNotify(c *gin.Context) { hupijiaoCallback(c, false) }
func HupijiaoReturn(c *gin.Context) { hupijiaoCallback(c, true) }

func hupijiaoCallback(c *gin.Context, redirect bool) {
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
	} else if appid == operation_setting.HupijiaoAlipayAppID {
		method = "alipay"
	}
	_, secret, _ := hupijiaoCredential(method)
	authenticated := method != "" && secret != "" && params["hash"] == hupijiaoSign(params, secret)
	tradeNo := params["trade_order_id"]
	orderValid := false
	if authenticated {
		if topup := model.GetTopUpByTradeNo(tradeNo); topup != nil {
			callbackAmount, parseErr := decimal.NewFromString(params["total_fee"])
			expected := decimal.NewFromFloat(topup.ActualAmount).Round(2)
			orderValid = parseErr == nil && topup.PaymentProvider == model.PaymentProviderHupijiao && topup.PaymentMethod == method && callbackAmount.Round(2).Equal(expected)
		}
	}
	accepted := authenticated && orderValid
	if accepted && params["status"] == "OD" {
		_, err := model.RechargeHupijiao(tradeNo, method, c.ClientIP())
		accepted = err == nil
	}
	if redirect {
		result := "fail"
		if accepted && params["status"] != "OD" {
			result = "pending"
		}
		if accepted && params["status"] == "OD" {
			result = "success"
		}
		target := "/wallet?pay=" + result
		c.Redirect(http.StatusFound, paymentReturnPath(target))
		return
	}
	if accepted {
		_, _ = c.Writer.Write([]byte("success"))
	} else {
		_, _ = c.Writer.Write([]byte("fail"))
	}
}
