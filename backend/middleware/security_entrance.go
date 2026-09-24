package middleware

import (
	"crypto/sha256"
	"crypto/subtle"
	"net/http"
	"strings"

	"xpanel/app/repo"
	"xpanel/global"

	"github.com/gin-gonic/gin"
)

const (
	entranceCookieName = "xpanel-entrance"
	entranceCookieAge  = 86400 * 7 // 7 天
)

// SecurityEntrance 安全入口中间件
// 如果配置了安全入口，用户必须先访问 /{entrance} 获取 cookie 后才能访问面板。
// 健康检查和证书服务不检查入口。数据库尚未打开时放行。
func SecurityEntrance() gin.HandlerFunc {
	settingRepo := repo.NewISettingRepo()

	return func(c *gin.Context) {
		path := c.Request.URL.Path
		if entranceExempt(path) || global.DB == nil {
			c.Next()
			return
		}

		entrance, err := settingRepo.GetValueByKey("SecurityEntrance")
		if err != nil || entrance == "" {
			c.Next()
			return
		}

		entrancePath := "/" + entrance
		if path == entrancePath || path == entrancePath+"/" {
			secure := c.Request.TLS != nil
			c.SetSameSite(http.SameSiteLaxMode)
			c.SetCookie(entranceCookieName, entrance, entranceCookieAge, "/", "", secure, true)
			c.Redirect(http.StatusTemporaryRedirect, "/")
			c.Abort()
			return
		}

		cookie, err := c.Cookie(entranceCookieName)
		if err == nil && secretEqual(cookie, entrance) {
			c.Next()
			return
		}

		// 没有有效 cookie，返回 404
		if global.LOG != nil {
			global.LOG.Debugf("Security entrance blocked: %s (no valid cookie)", path)
		}
		c.String(http.StatusNotFound, "404 page not found")
		c.Abort()
	}
}

func entranceExempt(path string) bool {
	switch {
	case strings.HasPrefix(path, "/.well-known/acme-challenge/"):
		return true
	case path == "/api/v1/version" || strings.HasPrefix(path, "/api/v1/version/"):
		return true
	case path == "/api/v1/cert-server" || strings.HasPrefix(path, "/api/v1/cert-server/"):
		return true
	default:
		return false
	}
}

func secretEqual(got, want string) bool {
	if want == "" {
		return false
	}
	sumGot := sha256.Sum256([]byte(got))
	sumWant := sha256.Sum256([]byte(want))
	return subtle.ConstantTimeCompare(sumGot[:], sumWant[:]) == 1
}
