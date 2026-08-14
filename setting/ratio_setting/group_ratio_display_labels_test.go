package ratio_setting

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetGroupRatioDisplayLabelFallback(t *testing.T) {
	original := GroupRatioDisplayLabels2JSONString()
	t.Cleanup(func() {
		require.NoError(t, UpdateGroupRatioDisplayLabelsByJSONString(original))
	})

	require.NoError(t, UpdateGroupRatioDisplayLabelsByJSONString(`{"default":"Standard tier","blank":"  "}`))

	require.Equal(t, "Standard tier", GetGroupRatioDisplayLabel("default", "Default description"))
	require.Equal(t, "VIP description", GetGroupRatioDisplayLabel("vip", " VIP description "))
	require.Equal(t, "missing", GetGroupRatioDisplayLabel("missing", ""))
	require.Equal(t, "blank", GetGroupRatioDisplayLabel("blank", ""))
}
