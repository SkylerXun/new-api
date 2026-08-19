/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
package middleware

import (
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
)

// isSidebarModuleEnabled reads the administrator-owned sidebar configuration.
// Missing or malformed configuration keeps new modules enabled for backwards
// compatibility with installations created before the module was introduced.
func isSidebarModuleEnabled(section, module string) bool {
	common.OptionMapRWMutex.RLock()
	raw := common.OptionMap["SidebarModulesAdmin"]
	common.OptionMapRWMutex.RUnlock()
	if strings.TrimSpace(raw) == "" {
		return true
	}

	var parsed map[string]map[string]any
	if err := common.Unmarshal([]byte(raw), &parsed); err != nil {
		return true
	}
	sectionConfig, ok := parsed[section]
	if !ok {
		return true
	}
	if enabled, ok := sectionConfig["enabled"]; ok && !parseHeaderNavBool(enabled, true) {
		return false
	}
	if enabled, ok := sectionConfig[module]; ok {
		return parseHeaderNavBool(enabled, true)
	}
	return true
}

// SidebarModuleAuth protects an authenticated endpoint with the same
// administrator sidebar-module switch used by the console navigation.
func SidebarModuleAuth(section, module string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !isSidebarModuleEnabled(section, module) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"success": false,
				"message": "sidebar module is disabled",
			})
			return
		}
		UserAuth()(c)
	}
}
