package middleware

import (
	"net/http"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

// AccountLinkageGuard prevents a stale session from using reward and payment
// endpoints after the account has been classified as a linked subaccount.
// Profile endpoints intentionally do not use this guard so an enabled user
// can inspect the relationship before a session is revoked.
func AccountLinkageGuard() gin.HandlerFunc {
	return func(c *gin.Context) {
		linkage, err := model.GetAccountLinkage(c.GetInt("id"))
		if err != nil {
			common.ApiError(c, err)
			c.Abort()
			return
		}
		if linkage.Relation == "subaccount" {
			c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "关联小号不可使用此功能，请使用主号。"})
			c.Abort()
			return
		}
		c.Next()
	}
}
