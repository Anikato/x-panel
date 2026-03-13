package middleware

import (
	"net/http"

	"xpanel/app/repo"

	"github.com/gin-gonic/gin"
)

func AgentTokenAuth() gin.HandlerFunc {
	settingRepo := repo.NewISettingRepo()
	return func(c *gin.Context) {
		token := c.GetHeader("X-Agent-Token")
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "agent token required"})
			c.Abort()
			return
		}

		agentToken, _ := settingRepo.GetValueByKey("AgentToken")
		if agentToken == "" || agentToken != token {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "invalid agent token"})
			c.Abort()
			return
		}
		c.Next()
	}
}
