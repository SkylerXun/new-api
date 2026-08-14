package controller

import (
	"net/http"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"

	"github.com/gin-gonic/gin"
)

func GetGroups(c *gin.Context) {
	groupNames := make([]string, 0)
	for groupName := range ratio_setting.GetGroupRatioCopy() {
		groupNames = append(groupNames, groupName)
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    groupNames,
	})
}

func buildUserGroupInfo(
	groupName, description string,
	ratio interface{},
) map[string]interface{} {
	return map[string]interface{}{
		"ratio":       ratio,
		"ratio_label": ratio_setting.GetGroupRatioDisplayLabel(groupName, description),
		"desc":        description,
	}
}

func GetUserGroups(c *gin.Context) {
	usableGroups := make(map[string]map[string]interface{})
	userGroup := ""
	userId := c.GetInt("id")
	userGroup, _ = model.GetUserGroup(userId, false)
	userUsableGroups := service.GetUserUsableGroups(userGroup)
	for groupName, _ := range ratio_setting.GetGroupRatioCopy() {
		// UserUsableGroups contains the groups that the user can use
		if desc, ok := userUsableGroups[groupName]; ok {
			usableGroups[groupName] = buildUserGroupInfo(
				groupName,
				desc,
				service.GetUserGroupRatio(userGroup, groupName),
			)
		}
	}
	if _, ok := userUsableGroups["auto"]; ok {
		description := setting.GetUsableGroupDescription("auto")
		usableGroups["auto"] = map[string]interface{}{
			"ratio":       "自动",
			"ratio_label": ratio_setting.GetGroupRatioDisplayLabel("auto", description),
			"desc":        description,
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    usableGroups,
	})
}
