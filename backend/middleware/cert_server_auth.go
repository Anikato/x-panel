package middleware

import (
	"net/http"
	"strings"

	"xpanel/app/model"
	"xpanel/app/repo"
	"xpanel/global"

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
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "cert server token required"})
			c.Abort()
			return
		}

		serverToken, _ := settingRepo.GetValueByKey("CertServerToken")
		if !secretEqual(token, serverToken) {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "invalid cert server token"})
			recordCertServerAccess(c, http.StatusUnauthorized, "令牌无效")
			c.Abort()
			return
		}
		c.Next()
		recordCertServerAccess(c, c.Writer.Status(), "")
	}
}

func recordCertServerAccess(c *gin.Context, status int, message string) {
	if global.DB == nil || c == nil || c.Request == nil {
		return
	}
	name := strings.TrimSpace(c.GetHeader("X-Cert-Client"))
	if len(name) > 128 {
		name = name[:128]
	}
	if len(message) > 256 {
		message = message[:256]
	}
	action := c.Request.Method + " " + c.Request.URL.Path
	if err := global.DB.Create(&model.CertServerAccessLog{
		ClientName: name,
		RemoteIP:   c.ClientIP(),
		Action:     action,
		Status:     status,
		Message:    message,
	}).Error; err != nil {
		return
	}
	var cutoff model.CertServerAccessLog
	if err := global.DB.Order("id desc").Offset(499).Limit(1).First(&cutoff).Error; err != nil {
		return
	}
	_ = global.DB.Where("id < ?", cutoff.ID).Delete(&model.CertServerAccessLog{}).Error
}
