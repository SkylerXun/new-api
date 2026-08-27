package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"

	"github.com/stretchr/testify/require"
)

// TestFormatUserLogsStripsQuotaSaturation verifies the admin-only quota
// saturation marker (nested under other.admin_info) is removed for non-admin
// log views, since formatUserLogs strips the whole admin_info object.
func TestFormatUserLogsStripsAdminAndSensitiveBillingFields(t *testing.T) {
	otherMap := map[string]interface{}{
		"cache_tokens":             128,
		"reasoning_effort":         "high",
		"web_search_call_count":    1,
		"monthly_discount_percent": 10,
		"monthly_discount_quota":   4500,
		"admin_info": map[string]interface{}{
			"quota_saturation": map[string]interface{}{
				"op":      "QuotaFromDecimal",
				"kind":    "overflow",
				"clamped": common.MaxQuota,
			},
			"monthly_discount": map[string]interface{}{
				"normal_quota":              5000,
				"progress_before_micro_usd": 990_000_000,
			},
		},
	}
	for _, field := range userLogSensitiveBillingFields {
		otherMap[field] = "sensitive"
	}
	other := common.MapToJsonStr(otherMap)
	logs := []*Log{{Other: other}}

	formatUserLogs(logs, 0)

	parsed, err := common.StrToMap(logs[0].Other)
	require.NoError(t, err)
	_, hasAdminInfo := parsed["admin_info"]
	require.False(t, hasAdminInfo, "admin_info (and nested quota_saturation) must be stripped for non-admin views")
	for _, field := range userLogSensitiveBillingFields {
		require.NotContains(t, parsed, field)
	}
	require.Equal(t, float64(128), parsed["cache_tokens"])
	require.Equal(t, "high", parsed["reasoning_effort"])
	require.Equal(t, float64(1), parsed["web_search_call_count"])
	require.Equal(t, float64(10), parsed["monthly_discount_percent"])
	require.Equal(t, float64(4500), parsed["monthly_discount_quota"])
}

func TestFormatUserLogsUsesMappedErrorForErrorLogs(t *testing.T) {
	logs := []*Log{
		{
			Type:    LogTypeError,
			Content: "status_code=503, upstream branded message",
			Other: common.MapToJsonStr(map[string]interface{}{
				"admin_info": map[string]interface{}{"mapped_error": "Service is busy"},
			}),
		},
		{
			Type:    LogTypeError,
			Content: "status_code=500, legacy upstream message",
			Other:   common.MapToJsonStr(map[string]interface{}{"admin_info": map[string]interface{}{}}),
		},
		{
			Type:    LogTypeConsume,
			Content: "consume detail",
			Other: common.MapToJsonStr(map[string]interface{}{
				"admin_info": map[string]interface{}{"mapped_error": "must not replace non-error logs"},
			}),
		},
	}

	formatUserLogs(logs, 0)

	require.Equal(t, "Service is busy", logs[0].Content)
	require.Equal(t, "status_code=500, legacy upstream message", logs[1].Content)
	require.Equal(t, "consume detail", logs[2].Content)
	for _, log := range logs {
		other, err := common.StrToMap(log.Other)
		require.NoError(t, err)
		require.NotContains(t, other, "admin_info")
	}
}
