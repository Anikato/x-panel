package middleware

import (
	"net/http"

	"xpanel/app/dto"
	"xpanel/app/service"
	"xpanel/constant"
	"xpanel/i18n"
	"xpanel/utils/accessticket"
	jwtUtil "xpanel/utils/jwt"

	"github.com/gin-gonic/gin"
)

func JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		if ticketID := c.Query("ticket"); ticketID != "" {
			ticket, err := accessticket.Default.Authorize(ticketID, c.Request.Method, c.Request.URL.Path, c.Query("path"))
			if err != nil {
				unauthorized(c, constant.ErrTokenInvalid)
				return
			}
			c.Set("userName", ticket.User)
			c.Set("sessionID", ticket.SessionID)
			c.Set("authSource", "ticket")
			c.Next()
			return
		}

		token := c.GetHeader(constant.JWTHeaderKey)
		if token == "" {
			unauthorized(c, constant.ErrNotLogin)
			return
		}
		if len(token) > len(constant.JWTTokenPrefix) && token[:len(constant.JWTTokenPrefix)] == constant.JWTTokenPrefix {
			token = token[len(constant.JWTTokenPrefix):]
		}

		claims, err := jwtUtil.ParseToken(token)
		if err != nil || claims.SessionID == "" || !service.SessionValid(claims.SessionID) {
			unauthorized(c, constant.ErrTokenInvalid)
			return
		}

		c.Set("userName", claims.UserName)
		c.Set("sessionID", claims.SessionID)
		c.Set("authSource", "jwt")
		c.Next()
	}
}

func unauthorized(c *gin.Context, key string) {
	c.JSON(http.StatusUnauthorized, dto.Response{
		Code:    http.StatusUnauthorized,
		Message: i18n.GetMsgByKey(key),
	})
	c.Abort()
}
