package jwtx

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

func TestGetTokenIncludesDeviceType(t *testing.T) {
	const secret = "test-secret"
	tokenString, err := GetToken(secret, "user-id", "user-name", "windows", time.Now().Unix(), 60)
	if err != nil {
		t.Fatal(err)
	}
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil || !token.Valid {
		t.Fatalf("parse token: %v", err)
	}
	claims := token.Claims.(jwt.MapClaims)
	if claims["device_type"] != "windows" {
		t.Fatalf("device_type = %#v", claims["device_type"])
	}
}
