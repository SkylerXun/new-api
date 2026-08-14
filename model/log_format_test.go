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
		"cache_tokens":          128,
		"reasoning_effort":      "high",
		"web_search_call_count": 1,
		"admin_info": map[string]interface{}{
			"quota_saturation": map[string]interface{}{
				"op":      "QuotaFromDecimal",
				"kind":    "overflow",
				"clamped": common.MaxQuota,
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
}
