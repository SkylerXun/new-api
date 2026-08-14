package common

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetTopupGroupRatioAlwaysReturnsOne(t *testing.T) {
	original := TopupGroupRatio2JSONString()
	t.Cleanup(func() {
		require.NoError(t, UpdateTopupGroupRatioByJSONString(original))
	})

	require.NoError(t, UpdateTopupGroupRatioByJSONString(`{"default":0.5,"vip":3}`))
	require.Equal(t, 1.0, GetTopupGroupRatio("default"))
	require.Equal(t, 1.0, GetTopupGroupRatio("vip"))
	require.Equal(t, 1.0, GetTopupGroupRatio("missing"))
}
