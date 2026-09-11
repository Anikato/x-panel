package jwt

import (
	"time"

	"xpanel/constant"
	"xpanel/global"

	"github.com/golang-jwt/jwt/v5"
)

// Claims 自定义 JWT Claims
type Claims struct {
	UserName  string `json:"userName"`
	SessionID string `json:"sessionId"`
	jwt.RegisteredClaims
}

// GenerateToken 生成 JWT Token
func GenerateToken(userName string) (string, error) {
	timeout := global.CONF.System.SessionTimeout
	if timeout <= 0 {
		timeout = constant.DefaultSessionTimeout
	}
	return GenerateTokenWithTimeout(userName, timeout)
}

// GenerateTokenWithTimeout 按指定秒数生成 JWT Token
func GenerateTokenWithTimeout(userName string, timeout int) (string, error) {
	return GenerateSessionToken(userName, "", timeout)
}

func GenerateSessionToken(userName, sessionID string, timeout int) (string, error) {
	if timeout <= 0 {
		timeout = constant.DefaultSessionTimeout
	}
	claims := Claims{
		UserName:  userName,
		SessionID: sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(timeout) * time.Second)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    constant.JWTIssuer,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(global.CONF.System.JwtSecret))
}

func ParseToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrTokenSignatureInvalid
		}
		return []byte(global.CONF.System.JwtSecret), nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, jwt.ErrTokenNotValidYet
	}
	return claims, nil
}
