package jwtx

import (
	"fmt"
	"github.com/golang-jwt/jwt/v4"
)

func GetToken(secretKey, uid string, iat, seconds int64) (string, error) {
	claims := make(jwt.MapClaims)
	claims["exp"] = iat + seconds
	claims["iat"] = iat
	claims["uid"] = uid
	token := jwt.New(jwt.SigningMethodHS256)
	token.Claims = claims
	return token.SignedString([]byte(secretKey))
}

func TokenKey(accessToken string) string {
	return fmt.Sprintf("%s", accessToken)
}
