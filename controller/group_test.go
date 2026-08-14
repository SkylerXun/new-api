package controller

import (
	"testing"

	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/stretchr/testify/require"
)

func TestBuildUserGroupInfoIncludesDisplayLabel(t *testing.T) {
	original := ratio_setting.GroupRatioDisplayLabels2JSONString()
	t.Cleanup(func() {
		require.NoError(t, ratio_setting.UpdateGroupRatioDisplayLabelsByJSONString(original))
	})

	require.NoError(t, ratio_setting.UpdateGroupRatioDisplayLabelsByJSONString(`{"vip":"Premium tier"}`))
	info := buildUserGroupInfo("vip", "Priority access", 2.5)

	require.Equal(t, 2.5, info["ratio"])
	require.Equal(t, "Priority access", info["desc"])
	require.Equal(t, "Premium tier", info["ratio_label"])
}
