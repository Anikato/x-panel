package middleware

import (
	"net/http"

	"xpanel/app/repo"

	"github.com/gin-gonic/gin"
)

func CertServerAuth() gin.HandlerFunc {
	settingRepo := repo.NewISettingRepo()
	return func(c *gin.Context) {
		enabled, _ := settingRepo.GetValueByKey("CertServerEnabled")
		if enabled != "enable" {
			c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "certificate server is not enabled"})
			c.Abort()
			return
		}

		token := c.GetHeader("X-Cert-Token")
		if token == "" {
			token = c.Query("token")
		}
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "cert server token required"})
			c.Abort()
			return
		}

		serverToken, _ := settingRepo.GetValueByKey("CertServerToken")
		if serverToken == "" || serverToken != token {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "invalid cert server token"})
			c.Abort()
			return
		}
		c.Next()
	}
}
