package middleware

import (
	"net/http"

	"xpanel/app/dto"
	"xpanel/constant"
	"xpanel/i18n"
	jwtUtil "xpanel/utils/jwt"

	"github.com/gin-gonic/gin"
)

// JWTAuth JWT 认证中间件
func JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader(constant.JWTHeaderKey)

		// 兼容 WebSocket：从 query 参数获取 token
		if token == "" {
			token = c.Query("token")
		}

		if token == "" {
			c.JSON(http.StatusUnauthorized, dto.Response{
				Code:    http.StatusUnauthorized,
				Message: i18n.GetMsgByKey(constant.ErrNotLogin),
			})
			c.Abort()
			return
		}

		// 去掉 Bearer 前缀（仅当 token 以 Bearer 开头时）
		if len(token) > len(constant.JWTTokenPrefix) && token[:len(constant.JWTTokenPrefix)] == constant.JWTTokenPrefix {
			token = token[len(constant.JWTTokenPrefix):]
		}

		claims, err := jwtUtil.ParseToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, dto.Response{
				Code:    http.StatusUnauthorized,
				Message: i18n.GetMsgByKey(constant.ErrTokenInvalid),
			})
			c.Abort()
			return
		}

		// 将用户信息放入上下文
		c.Set("userName", claims.UserName)
		c.Next()
	}
}
