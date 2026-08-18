package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAffiliateRebatePercentOptionValidation(t *testing.T) {
	tests := []struct {
		name  string
		value string
		valid bool
	}{
		{name: "zero is allowed", value: "0", valid: true},
		{name: "hundred is allowed", value: "100", valid: true},
		{name: "fraction is allowed", value: "12.5", valid: true},
		{name: "negative is rejected", value: "-0.1"},
		{name: "over one hundred is rejected", value: "100.1"},
		{name: "nan is rejected", value: "NaN"},
		{name: "positive infinity is rejected", value: "+Inf"},
		{name: "negative infinity is rejected", value: "-Inf"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateOptionValue("affiliate_setting.redeem_rebate_percent", tt.value)
			if tt.valid {
				require.NoError(t, err)
				return
			}
			assert.Error(t, err)
		})
	}
}

func TestAffiliateRebateEnabledOptionValidation(t *testing.T) {
	assert.NoError(t, validateOptionValue("affiliate_setting.redeem_rebate_enabled", "true"))
	assert.NoError(t, validateOptionValue("affiliate_setting.redeem_rebate_enabled", "false"))
	assert.Error(t, validateOptionValue("affiliate_setting.redeem_rebate_enabled", "yes"))
}
