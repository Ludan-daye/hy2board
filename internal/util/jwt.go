package util

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/ludandaye/hy2board/internal/config"
)

func GenerateToken(username string) (string, error) {
	return generateToken(username, "admin")
}

func GenerateUserToken(username string) (string, error) {
	return generateToken(username, "user")
}

func generateToken(username, role string) (string, error) {
	expiry, _ := time.ParseDuration(config.C.JWT.Expiry)
	if expiry == 0 {
		expiry = 24 * time.Hour
	}

	claims := jwt.MapClaims{
		"sub":  username,
		"role": role,
		"exp":  time.Now().Add(expiry).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(config.C.JWT.Secret))
}

func ParseToken(tokenStr string) (username string, role string, err error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		return []byte(config.C.JWT.Secret), nil
	})
	if err != nil {
		return "", "", err
	}
	claims := token.Claims.(jwt.MapClaims)
	username = claims["sub"].(string)
	if r, ok := claims["role"].(string); ok {
		role = r
	} else {
		role = "admin"
	}
	return username, role, nil
}
