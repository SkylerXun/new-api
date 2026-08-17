package controller

import (
	"testing"

	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/require"
)

func TestFilterPriceListPricingExcludesNonTokenAndTieredModels(t *testing.T) {
	pricing := []model.Pricing{
		{ModelName: "token-model", QuotaType: 0},
		{ModelName: "per-request", QuotaType: 1},
		{ModelName: "tiered", QuotaType: 0, BillingMode: "tiered_expr"},
	}

	filtered := filterPriceListPricing(pricing)

	require.Len(t, filtered, 1)
	require.Equal(t, "token-model", filtered[0].ModelName)
}
