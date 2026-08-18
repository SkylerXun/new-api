package controller

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateOptionRejectsNonFiniteActivityBonusPercent(t *testing.T) {
	for _, value := range []string{"NaN", "+Inf", "-Inf"} {
		t.Run(value, func(t *testing.T) {
			response := httptest.NewRecorder()
			context, _ := gin.CreateTestContext(response)
			context.Request = httptest.NewRequest(
				http.MethodPut,
				"/api/option/",
				strings.NewReader(`{"key":"activity_setting.new_user_redeem_bonus_percent","value":"`+value+`"}`),
			)

			UpdateOption(context)

			assert.Equal(t, http.StatusOK, response.Code)
			var payload struct {
				Success bool   `json:"success"`
				Message string `json:"message"`
			}
			require.NoError(t, common.Unmarshal(response.Body.Bytes(), &payload))
			assert.False(t, payload.Success)
			assert.NotEmpty(t, payload.Message)
		})
	}
}

func TestUpdateOptionRejectsInvalidAffiliateRebatePercent(t *testing.T) {
	for _, value := range []string{"-1", "100.1", "NaN", "+Inf", "-Inf"} {
		t.Run(value, func(t *testing.T) {
			response := httptest.NewRecorder()
			context, _ := gin.CreateTestContext(response)
			context.Request = httptest.NewRequest(
				http.MethodPut,
				"/api/option/",
				strings.NewReader(`{"key":"affiliate_setting.redeem_rebate_percent","value":"`+value+`"}`),
			)

			UpdateOption(context)

			assert.Equal(t, http.StatusOK, response.Code)
			var payload struct {
				Success bool   `json:"success"`
				Message string `json:"message"`
			}
			require.NoError(t, common.Unmarshal(response.Body.Bytes(), &payload))
			assert.False(t, payload.Success)
			assert.NotEmpty(t, payload.Message)
		})
	}
}

func TestUpdateOptionRequiresComplianceForAffiliateRebate(t *testing.T) {
	setting := operation_setting.GetPaymentSetting()
	previous := *setting
	setting.ComplianceConfirmed = false
	setting.ComplianceTermsVersion = ""
	t.Cleanup(func() { *setting = previous })

	tests := []struct {
		name string
		body string
	}{
		{
			name: "enable",
			body: `{"key":"affiliate_setting.redeem_rebate_enabled","value":true}`,
		},
		{
			name: "positive percent",
			body: `{"key":"affiliate_setting.redeem_rebate_percent","value":25}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			context, _ := gin.CreateTestContext(response)
			context.Request = httptest.NewRequest(http.MethodPut, "/api/option/", strings.NewReader(tt.body))

			UpdateOption(context)

			assert.Equal(t, http.StatusOK, response.Code)
			var payload struct {
				Success bool   `json:"success"`
				Message string `json:"message"`
			}
			require.NoError(t, common.Unmarshal(response.Body.Bytes(), &payload))
			assert.False(t, payload.Success)
			assert.NotEmpty(t, payload.Message)
		})
	}
}
