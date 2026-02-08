package jwt

import (
	"time"

	"xpanel/constant"
	"xpanel/global"

	"github.com/golang-jwt/jwt/v5"
)

// Claims 自定义 JWT Claims
type Claims struct {
	UserName string `json:"userName"`
	jwt.RegisteredClaims
}

// GenerateToken 生成 JWT Token
func GenerateToken(userName string) (string, error) {
	timeout := global.CONF.System.SessionTimeout
	if timeout <= 0 {
		timeout = constant.DefaultSessionTimeout
	}

	claims := Claims{
		UserName: userName,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(timeout) * time.Second)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    constant.JWTIssuer,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(global.CONF.System.JwtSecret))
}

// ParseToken 解析 JWT Token
func ParseToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(global.CONF.System.JwtSecret), nil
	})
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}
	return nil, jwt.ErrTokenNotValidYet
}
