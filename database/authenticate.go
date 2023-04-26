package database

import (
	environment "application/environment"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type LoginPayload struct {
	Database      string
	Table         string
	Body          map[string]interface{}
	IdentityField string
	PasswordField string
	Query         map[string]interface{}
}

type AuthConfig struct {
	IdentifyField string                 `json:"identityField"`
	PasswordField string                 `json:"passwordField"`
	Query         map[string]interface{} `json:"query"`
}

type GlobalAuthEntity struct {
	Id         int
	CreatedAt  string
	Database   string
	Table      string
	AuthConfig AuthConfig
}

func GetSecret() []byte {
	secret := environment.GetEnvValue("JWT_SECRET")
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

func (e *Engine) Login(role string, db *sql.DB, payload LoginPayload) (string, error) {

	body := map[string]map[string]interface{}{
		"_where": make(map[string]interface{}),
	}

	identityValue, ok := payload.Body[payload.IdentityField]
	if !ok {
		return "", fmt.Errorf("no identity value was provided")
	}

	passwordValue, ok := payload.Body[payload.PasswordField]
	if !ok {
		return "", fmt.Errorf("no password value was provided")
	}

	body["_where"][payload.IdentityField] = map[string]interface{}{
		"_eq": identityValue,
	}
	queryPayload := make(map[string]interface{})
	payload.Query["_where"] = body["_where"]
	payload.Query["_limit"] = 1
	queryPayload[payload.Table] = payload.Query

	result, err := e.SelectExec(role, db, payload.Database, queryPayload)

	if err != nil {
		return "", err
	}
	var users interface{}
	err = json.Unmarshal(result, &users)
	if err != nil {
		return "", err
	}

	parsedUsers, err := IsMapToInterface(users)

	if err != nil {
		return "", fmt.Errorf("Unauthorized")
	}

	usersMapResult, ok := parsedUsers["users"]
	if !ok {
		return "", fmt.Errorf("Unauthorized")
	}

	parsedUser, err := IsArray(usersMapResult)

	if err != nil {
		return "", fmt.Errorf("something went wrong")
	}

	if len(parsedUser) != 1 {
		return "", fmt.Errorf("Unauthorized")
	}

	userMap := parsedUser[0]

	userEntity, err := IsMapToInterface(userMap)
	if err != nil {
		return "", fmt.Errorf("Unauthorized")
	}
	password, ok := userEntity["password"]

	if !ok {
		return "", fmt.Errorf("something went wrong")
	}

	err = bcrypt.CompareHashAndPassword([]byte(password.(string)), []byte(passwordValue.(string)))

	if err != nil {
		return "", fmt.Errorf("Unauthorized")
	}

	delete(userEntity, payload.PasswordField)
	claims := jwt.MapClaims{}
	for key, value := range userEntity {
		claims[key] = value
	}
	claims["exp"] = GetExpirationTimeForToken(60)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString(GetSecret())
	if err != nil {
		return "", err
	}

	return tokenString, nil

}

func (e *Engine) RefreshToken(payload jwt.MapClaims) (string, error) {
	payload["exp"] = GetExpirationTimeForToken(1 * 60)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, payload)

	tokenString, err := token.SignedString(GetSecret())
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func (e *Engine) LoadGlobalAuth(db *sql.DB) []GlobalAuthEntity {
	var jsonValue interface{}
	cb := func(rows *sql.Rows) error {

		entry := new(GlobalAuthEntity)
		err := rows.Scan(&entry.Id, &entry.CreatedAt, &entry.Database, &entry.Table, &jsonValue)
		if jsonValue != nil {

			err := json.Unmarshal(jsonValue.([]byte), &entry.AuthConfig)
			if err != nil {
				panic(err)
			}
		}
		e.GlobalAuthEntities = append(e.GlobalAuthEntities, *entry)
		return err
	}

	scanner := Query(db, GET_GLOBAL_AUTH_CONFIG)

	err := scanner(cb)
	if err != nil {
		panic(err)
	}

	return e.GlobalAuthEntities
}

func (e *Engine) DeriveAuthConfigAsMap(authConfig interface{}) map[string]interface{} {
	switch x := authConfig.(type) {
	case map[string]interface{}:
		return x
	default:
		return make(map[string]interface{})
	}
}

func GetExpirationTimeForToken(minutes int64) int64 {
	if minutes <= 0 {
		return -1
	}

	return time.Now().Add(time.Minute * time.Duration(minutes)).Unix()
}
