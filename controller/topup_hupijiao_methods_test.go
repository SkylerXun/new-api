package controller

import (
	"testing"

	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/stretchr/testify/assert"
)

func TestHupijiaoPayMethodsRequireCompleteCredentials(t *testing.T) {
	originalAlipayAppID := operation_setting.HupijiaoAlipayAppID
	originalAlipaySecret := operation_setting.HupijiaoAlipaySecret
	originalWechatAppID := operation_setting.HupijiaoWechatAppID
	originalWechatSecret := operation_setting.HupijiaoWechatSecret
	t.Cleanup(func() {
		operation_setting.HupijiaoAlipayAppID = originalAlipayAppID
		operation_setting.HupijiaoAlipaySecret = originalAlipaySecret
		operation_setting.HupijiaoWechatAppID = originalWechatAppID
		operation_setting.HupijiaoWechatSecret = originalWechatSecret
	})

	operation_setting.HupijiaoAlipayAppID = "alipay-app"
	operation_setting.HupijiaoAlipaySecret = "alipay-secret"
	operation_setting.HupijiaoWechatAppID = ""
	operation_setting.HupijiaoWechatSecret = "stale-secret"

	methods := hupijiaoPayMethods()
	assert.Equal(t, []map[string]string{{
		"name": "支付宝",
		"type": "alipay",
		"icon": "SiAlipay",
	}}, methods)
}

func TestHupijiaoPayMethodsPreferAlipay(t *testing.T) {
	originalAlipayAppID := operation_setting.HupijiaoAlipayAppID
	originalAlipaySecret := operation_setting.HupijiaoAlipaySecret
	originalWechatAppID := operation_setting.HupijiaoWechatAppID
	originalWechatSecret := operation_setting.HupijiaoWechatSecret
	t.Cleanup(func() {
		operation_setting.HupijiaoAlipayAppID = originalAlipayAppID
		operation_setting.HupijiaoAlipaySecret = originalAlipaySecret
		operation_setting.HupijiaoWechatAppID = originalWechatAppID
		operation_setting.HupijiaoWechatSecret = originalWechatSecret
	})

	operation_setting.HupijiaoAlipayAppID = "alipay-app"
	operation_setting.HupijiaoAlipaySecret = "alipay-secret"
	operation_setting.HupijiaoWechatAppID = "wechat-app"
	operation_setting.HupijiaoWechatSecret = "wechat-secret"

	methods := hupijiaoPayMethods()
	assert.Len(t, methods, 2)
	assert.Equal(t, "alipay", methods[0]["type"])
	assert.Equal(t, "wxpay", methods[1]["type"])
}
