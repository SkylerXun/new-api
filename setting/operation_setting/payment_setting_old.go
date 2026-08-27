/**
此文件为旧版支付设置文件，如需增加新的参数、变量等，请在 payment_setting.go 中添加
This file is the old version of the payment settings file. If you need to add new parameters, variables, etc., please add them in payment_setting.go
*/

package operation_setting

import (
	"errors"
	"math"
	"strings"

	"github.com/QuantumNous/new-api/common"
)

var PayAddress = ""
var CustomCallbackAddress = ""
var EpayId = ""
var EpayKey = ""
var Price = 7.3
var MinTopUp = 1
var USDExchangeRate = 7.3

// Built-in Hupijiao payment configuration.
var OnlinePaymentProvider = "epay"
var HupijiaoAPIAddress = "https://api.xunhupay.com"
var HupijiaoBackupAPIAddress = "https://api.dpweixin.com"
var HupijiaoWechatAppID = ""
var HupijiaoWechatSecret = ""
var HupijiaoAlipayAppID = ""
var HupijiaoAlipaySecret = ""
var HupijiaoPackages = "[]"

type HupijiaoPackageConfig struct {
	ID             string  `json:"id"`
	Title          string  `json:"title"`
	OriginalAmount float64 `json:"original_amount"`
	Quota          int64   `json:"quota"`
	DiscountRate   float64 `json:"discount_rate"`
	Enabled        bool    `json:"enabled"`
}

func ParseHupijiaoPackages(value string) ([]HupijiaoPackageConfig, error) {
	if !strings.HasPrefix(strings.TrimSpace(value), "[") {
		return nil, errors.New("虎皮椒套餐必须是JSON数组")
	}
	var packages []HupijiaoPackageConfig
	if err := common.Unmarshal([]byte(value), &packages); err != nil {
		return nil, err
	}
	seen := make(map[string]struct{}, len(packages))
	for _, p := range packages {
		if p.ID == "" || p.Title == "" || p.OriginalAmount < 0.01 || p.OriginalAmount > 999999 || p.Quota <= 0 || p.Quota > math.MaxInt32 || p.DiscountRate < 0.01 || p.DiscountRate > 1 {
			return nil, errors.New("虎皮椒套餐配置无效")
		}
		if _, exists := seen[p.ID]; exists {
			return nil, errors.New("虎皮椒套餐ID重复")
		}
		seen[p.ID] = struct{}{}
	}
	return packages, nil
}

var PayMethods = []map[string]string{
	{
		"name": "支付宝",
		"icon": "SiAlipay",
		"type": "alipay",
	},
	{
		"name": "微信",
		"icon": "SiWechat",
		"type": "wxpay",
	},
	{
		"name":      "自定义1",
		"icon":      "LuCreditCard",
		"type":      "custom1",
		"min_topup": "50",
	},
}

func UpdatePayMethodsByJsonString(jsonString string) error {
	PayMethods = make([]map[string]string, 0)
	return common.Unmarshal([]byte(jsonString), &PayMethods)
}

func PayMethods2JsonString() string {
	jsonBytes, err := common.Marshal(PayMethods)
	if err != nil {
		return "[]"
	}
	return string(jsonBytes)
}

func ContainsPayMethod(method string) bool {
	for _, payMethod := range PayMethods {
		if payMethod["type"] == method {
			return true
		}
	}
	return false
}
