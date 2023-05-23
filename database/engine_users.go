package database

import (
	"database/sql"
	"fmt"

	"github.com/golang-jwt/jwt/v5"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

type EngineRole struct {
	Id          int    `json:"id"`
	RoleName    string `json:"role_name"`
	CreatedAt   string `json:"created_at"`
	Permissions []byte `json:"permissions"`
}

type EngineUserInput struct {
	Id        int    `json:"id"`
	Email     string `json:"email"`
	Password  string `json:"password"`
	CreatedAt string `json:"created_at"`
	RoleName  string `json:"role_name"`
	RoleId    string `json:"role_id"`
}

func (engine *Engine) GetEngineRoleByName(db *sql.DB, roleName string) (EngineRole, error) {
	var engineRole EngineRole

	row := db.QueryRow(GET_ENGINE_ROLE, roleName)
	if row == nil {
		return engineRole, fmt.Errorf("no such role")
	}

	err := row.Scan(&engineRole.Id, &engineRole.RoleName, &engineRole.Permissions, &engineRole.CreatedAt)
	if err != nil {
		return engineRole, err
	}
	return engineRole, nil

}

func (engine *Engine) CreateEngineRole(db *sql.DB, input EngineRole) {
	db.Exec(CREATE_ENGINE_ROLE, input.RoleName)
}

func (engine *Engine) CreateEngineUser(db *sql.DB, input EngineUserInput) error {

	engineRole, err := engine.GetEngineRoleByName(db, input.RoleName)
	if err != nil {
		return err
	}

	hashBytes, err := bcrypt.GenerateFromPassword([]byte(input.Password), 12)
	if err != nil {
		return err
	}
	hashPWD := string(hashBytes)

	_, err = db.Exec(CREATE_ENGINE_SUPER_USER, input.Email, hashPWD, engineRole.Id)

	return err
}

func (e *Engine) LoginEngineUser(db *sql.DB, input EngineUserInput) (string, error) {
	row := db.QueryRow(GET_ENGINE_USER_BY_EMAIL, input.Email)
	var engineUser EngineUserInput

	err := row.Scan(&engineUser.Id, &engineUser.Email, &engineUser.Password, &engineUser.CreatedAt, &engineUser.RoleId, &engineUser.RoleName)
	if err != nil {
		return "", err
	}
	fmt.Println(engineUser)
	err = bcrypt.CompareHashAndPassword([]byte(engineUser.Password), []byte(input.Password))
	if err != nil {
		return "", err
	}
	claims := jwt.MapClaims{}
	expirationMinutes := GetTokenExpirationTimeFromEnv()
	if expirationMinutes > 0 {
		claims["exp"] = GetExpirationTimeForToken(expirationMinutes)
	}
	claims["id"] = engineUser.Id
	claims["email"] = engineUser.Email
	claims["role_id"] = engineUser.RoleId
	claims["role_name"] = engineUser.RoleName
	claims["created_at"] = engineUser.CreatedAt
	claims["bypass_all"] = true

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString(GetSecret())
	if err != nil {
		return "", err
	}

	return tokenString, nil

}
