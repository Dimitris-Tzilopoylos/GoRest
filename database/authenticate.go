package database

import (
	environment "application/environment"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/mail"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type RequestContextKey string

type AuthActionPayload struct {
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

var RegistrationRestrictKeysMap map[string]bool = map[string]bool{
	"database": true,
	"table":    true,
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
		req = req.WithContext(context.WithValue(req.Context(), RequestContextKey("auth"), claims))
		return req, nil
	}
	return nil, fmt.Errorf("unauthorized")
}

func (e *Engine) AuthenticateForDatabase(req *http.Request, database string) (*http.Request, error) {
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
		claimsDatabase, ok := claims["database"]
		if !ok {
			return nil, fmt.Errorf("unauthorized")
		}

		parsedDatabase, ok := claimsDatabase.(string)
		if !ok {
			return nil, fmt.Errorf("unauthorized")
		}

		if parsedDatabase != database {
			return nil, fmt.Errorf("unauthorized")
		}
		req = req.WithContext(context.WithValue(req.Context(), RequestContextKey("auth"), claims))

		return req, nil
	}
	return nil, fmt.Errorf("unauthorized")
}

func (e *Engine) Login(role string, db *sql.DB, payload AuthActionPayload) (string, error) {

	identityValue, ok := payload.Body[payload.IdentityField]
	if !ok {
		return "", fmt.Errorf("no identity value was provided")
	}

	passwordValue, ok := payload.Body[payload.PasswordField]
	if !ok {
		return "", fmt.Errorf("no password value was provided")
	}

	_, err := ValidateAuthAction(identityValue, passwordValue)

	if err != nil {
		return "", err
	}

	body := map[string]map[string]interface{}{
		"_where": make(map[string]interface{}),
	}

	body["_where"][payload.IdentityField] = map[string]interface{}{
		"_eq": identityValue,
	}
	queryPayload := make(map[string]interface{})
	payload.Query["_where"] = body["_where"]
	payload.Query["_limit"] = 1
	relationalPayload, ok := payload.Query[payload.Table]
	if ok {
		relationalPayload, err := IsMapToInterface(relationalPayload)
		if err == nil {
			for key, value := range relationalPayload {
				payload.Query[key] = value
			}
		}
	}
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
	expirationMinutes := GetTokenExpirationTimeFromEnv()
	if expirationMinutes > 0 {
		claims["exp"] = GetExpirationTimeForToken(expirationMinutes)
	}

	claims["database"] = payload.Database

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString(GetSecret())
	if err != nil {
		return "", err
	}

	return tokenString, nil

}

func (e *Engine) Register(role string, db *sql.DB, payload AuthActionPayload) (interface{}, error) {

	identityValue, ok := payload.Body[payload.IdentityField]
	if !ok {
		return nil, fmt.Errorf("no identity value was provided")
	}

	passwordValue, ok := payload.Body[payload.PasswordField]
	if !ok {
		return nil, fmt.Errorf("no password value was provided")
	}

	_, err := ValidateAuthAction(identityValue, passwordValue)

	if err != nil {
		return nil, err
	}

	parsedPayload, err := IsMapToInterface(payload.Body)
	if err != nil {

		return nil, err
	}

	insertPayload := make(map[string]interface{})
	objects := make([]interface{}, 0)
	for key, value := range parsedPayload {
		if _, ok := RegistrationRestrictKeysMap[key]; !ok {
			if key == payload.PasswordField {
				passwordValueToString, ok := passwordValue.(string)
				if !ok {
					return nil, fmt.Errorf("registration failed: failed to hash password field: please provide the password as a string")
				}
				passwordBytes := []byte(passwordValueToString)
				hashBytes, err := bcrypt.GenerateFromPassword(passwordBytes, 12)
				if err != nil {
					return nil, fmt.Errorf("registration failed: failed to hash password field")
				}
				insertPayload[key] = string(hashBytes)
			} else {
				insertPayload[key] = value
			}
		}
	}

	if len(insertPayload) == 0 {
		return nil, fmt.Errorf("registration failed")
	}

	objects = append(objects, insertPayload)

	body := make(map[string]interface{})

	body[payload.Table] = map[string]interface{}{
		"objects": objects,
	}

	result, err := e.InsertExec(role, db, payload.Database, body)

	if err != nil {
		return nil, err
	}

	parsedResult, err := IsMapToArray(result)
	if err != nil {
		return nil, err
	}

	parsedUsers, ok := parsedResult[payload.Table]

	if !ok {
		return nil, fmt.Errorf("nothing was returned after registering")
	}

	users, err := IsArray(parsedUsers)

	if err != nil {
		return nil, err
	}

	if len(users) == 0 {
		return nil, fmt.Errorf("nothing was returned after registering")
	}

	user := users[0]

	parsedUser, err := IsMapToInterface(user)

	if err != nil {
		return nil, err
	}

	delete(parsedUser, payload.PasswordField)

	return parsedUser, nil

}

func (e *Engine) RefreshToken(payload jwt.MapClaims) (string, error) {
	expirationMinutes := GetTokenExpirationTimeFromEnv()
	if expirationMinutes > 0 {
		payload["exp"] = GetExpirationTimeForToken(expirationMinutes)
	}
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

func GetTokenExpirationTimeFromEnv() int64 {
	expiration := environment.GetEnvValue("JWT_EXPIRATION_IN_MINUTES")

	minutes, err := strconv.Atoi(expiration)

	if err != nil {
		return 0
	}

	if minutes <= 0 {
		return 0
	}

	return int64(minutes)

}

func ValidateIdentityFieldValue(value string) bool {

	value = strings.Trim(value, " ")

	if len(value) <= 0 {
		return false
	}
	if strings.Count(value, "@") > 0 {
		_, err := mail.ParseAddress(value)
		return err == nil
	}

	return true

}

func ValidatePasswordFieldValue(value string) bool {
	value = strings.Trim(value, " ")

	return len(value) > 7 && len(value) < 17
}

func ValidateAuthAction(identityValue, passwordValue interface{}) (bool, error) {

	username, ok := identityValue.(string)

	if !ok {
		return false, fmt.Errorf("identity field value should be provided as string")
	}

	if !ValidateIdentityFieldValue(username) {
		return false, fmt.Errorf("identity field value is invalid")
	}

	password, ok := passwordValue.(string)

	if !ok {
		return false, fmt.Errorf("password field value should be provided as string")
	}

	if !ValidatePasswordFieldValue(password) {
		return false, fmt.Errorf("password field value is invalid")
	}

	return true, nil
}
