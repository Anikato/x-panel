package middleware

import (
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
// 如果配置了安全入口，用户必须先访问 /{entrance} 获取 cookie 后才能访问面板
// API 路由 (/api/*) 不受影响（通过 JWT 保护）
func SecurityEntrance() gin.HandlerFunc {
	settingRepo := repo.NewISettingRepo()

	return func(c *gin.Context) {
		path := c.Request.URL.Path

		// API 和 WebSocket 请求不受安全入口限制
		if strings.HasPrefix(path, "/api/") {
			c.Next()
			return
		}

		// 查询安全入口配置
		entrance, err := settingRepo.GetValueByKey("SecurityEntrance")
		if err != nil || entrance == "" {
			// 未配置安全入口，直接放行
			c.Next()
			return
		}

		// 检查是否正在访问安全入口路径
		entrancePath := "/" + entrance
		if path == entrancePath || path == entrancePath+"/" {
			// 设置 cookie 并重定向到首页
			secure := c.Request.TLS != nil
			c.SetCookie(entranceCookieName, entrance, entranceCookieAge, "/", "", secure, true)
			c.Redirect(http.StatusTemporaryRedirect, "/")
			c.Abort()
			return
		}

		// 检查 cookie 是否有效
		cookie, err := c.Cookie(entranceCookieName)
		if err == nil && cookie == entrance {
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
