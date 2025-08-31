package utils

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func GenerateJWT(secret string, userID int, username string, ttl time.Duration) (string, error) {
	claims := jwt.MapClaims{
		"uid":    userID,
		"uname":  username,
		"exp":    time.Now().Add(ttl).Unix(),
		"iat":    time.Now().Unix(),
		"iss":    "chat-backend",
		"sub":    "user-auth",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func ParseJWT(secret, tokenStr string) (int, string, error) {
	if tokenStr == "" {
		return 0, "", errors.New("token is empty")
	}
	
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})
	
	if err != nil {
		return 0, "", errors.New("invalid token")
	}
	
	if !token.Valid {
		return 0, "", errors.New("token is invalid")
	}
	
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return 0, "", errors.New("invalid claims")
	}
	
	uidF, ok1 := claims["uid"].(float64)
	uname, ok2 := claims["uname"].(string)
	if !ok1 || !ok2 {
		return 0, "", errors.New("bad claims")
	}
	
	return int(uidF), uname, nil
}
