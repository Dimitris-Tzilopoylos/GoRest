package database

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

func GetSecret() []byte {
	secret := "flsdjhgfsjdkhgskjdhfjksdhgds"
	return []byte(secret)
}

func (e *Engine) Authenticate(req *http.Request) (*http.Request, error) {
	auth := req.Header.Get("Authorization")
	if auth == "" {
		return nil, fmt.Errorf("no token was provided")
	}

	tokenString := strings.Split(auth, " ")[1]

	if len(tokenString) < 1 {
		return nil, fmt.Errorf("no token was provided")
	}
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return GetSecret(), nil
	})

	if err != nil {
		return nil, fmt.Errorf("unauthorized")
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		req = req.WithContext(context.WithValue(req.Context(), "auth", claims))
		return req, nil
	}
	return nil, fmt.Errorf("unauthorized")
}
