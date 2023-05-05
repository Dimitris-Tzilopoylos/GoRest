package database

import (
	"application/environment"
	"database/sql"
	"fmt"
	"regexp"
	"strings"
)

type CustomRestHandlerInput struct {
	Database  string `json:"database"`
	Auth      bool   `json:"auth"`
	Enabled   bool   `json:"enabled"`
	Method    string `json:"method"`
	Endpoint  string `json:"endpoint"`
	Query     string `json:"query"`
	Id        any    `json:"id"`
	CreatedAt string `json:"created_at"`
}

var GET string = "GET"
var POST string = "POST"
var PUT string = "PUT"
var PATCH string = "PATCH"
var DELETE string = "DELETE"

func FormatDBName(database string) string {
	return strings.ToLower(strings.Trim(database, " "))
}

func FormatQuery(query string) string {
	return strings.Trim(query, " ")
}

func (engine *Engine) ValidateDatabase(input CustomRestHandlerInput) error {

	dbname := FormatDBName(input.Database)
	if len(dbname) == 0 {
		return fmt.Errorf("database name was not provided")
	}

	if input.Database == environment.GetEnvValue("INTERNAL_SCHEMA_NAME") {
		return fmt.Errorf("cannot use  %s database for this action", environment.GetEnvValue("INTERNAL_SCHEMA_NAME"))
	}

	_, ok := engine.DatabaseToTableToModelMap[dbname]

	if !ok {
		return fmt.Errorf("database %s doesn't exist", dbname)
	}

	return nil

}

func (engine *Engine) ValidateMethod(input CustomRestHandlerInput) error {
	switch input.Method {
	case GET:
	case POST:
	case PUT:
	case PATCH:
	case DELETE:
		return nil
	default:
		return fmt.Errorf("not supported method")
	}
	return nil
}

func (engine *Engine) ValidateEndpoint(input CustomRestHandlerInput) error {
	pattern := "^/rest(/[a-zA-Z0-9_]+)+$"
	re := regexp.MustCompile(pattern)

	if !re.MatchString(input.Endpoint) {
		return fmt.Errorf("endpoint should start with /rest/ and should contain only letters numbers and underscores")
	}
	return nil
}

func (engine *Engine) LoadRestHandlers(db *sql.DB) {
	query := fmt.Sprintf("SELECT id,method,endpoint,db,query,enabled,auth,created_at FROM %s.engine_rest_actions", environment.GetEnvValue("INTERNAL_SCHEMA_NAME"))
	var restHandlers []CustomRestHandlerInput = make([]CustomRestHandlerInput, 0)
	scanner := Query(db, query)
	cb := func(rows *sql.Rows) error {
		var restHandler CustomRestHandlerInput
		err := rows.Scan(&restHandler.Id, &restHandler.Method, &restHandler.Endpoint, &restHandler.Database, &restHandler.Query, &restHandler.Enabled, &restHandler.Auth, &restHandler.CreatedAt)
		if err != nil {
			panic(err)
		}
		restHandlers = append(restHandlers, restHandler)
		return err
	}
	err := scanner(cb)
	if err != nil {
		panic(err)
	}

	engine.RestHandlers = restHandlers
	engine.RestHandlersMap = map[string]map[string]CustomRestHandlerInput{}
	for _, handler := range restHandlers {
		if _, ok := engine.RestHandlersMap[handler.Method]; !ok {
			engine.RestHandlersMap[handler.Method] = make(map[string]CustomRestHandlerInput)
		}

		if _, ok := engine.RestHandlersMap[handler.Method][handler.Endpoint]; !ok {
			engine.RestHandlersMap[handler.Method][handler.Endpoint] = handler
		}
	}

}

func (engine *Engine) CreateRestHandler(db *sql.DB, customHandlerInput CustomRestHandlerInput) error {
	customHandlerInput.Database = FormatDBName(customHandlerInput.Database)
	customHandlerInput.Query = FormatQuery(customHandlerInput.Query)
	err := engine.ValidateDatabase(customHandlerInput)
	if err != nil {
		return err
	}

	err = engine.ValidateMethod(customHandlerInput)
	if err != nil {
		return err
	}

	err = engine.ValidateEndpoint(customHandlerInput)
	if err != nil {
		return err
	}

	err = CheckSQLStringValidity(db, customHandlerInput.Query)
	if err != nil {
		return err
	}

	query := fmt.Sprintf("INSERT INTO %s.engine_rest_actions(endpoint,method,db,query,enabled,auth) VALUES($1,$2,$3,$4,$5,$6)", environment.GetEnvValue("INTERNAL_SCHEMA_NAME"))
	_, err = db.Exec(query, customHandlerInput.Endpoint, customHandlerInput.Method, customHandlerInput.Database, customHandlerInput.Query, customHandlerInput.Enabled, customHandlerInput.Auth)

	if err != nil {
		return err
	}

	return nil
}

func (engine *Engine) UpdateRestHandler(db *sql.DB, customHandlerInput CustomRestHandlerInput) error {
	if customHandlerInput.Id == nil {
		return fmt.Errorf("rest handler id  was not provided")
	}
	customHandlerInput.Database = FormatDBName(customHandlerInput.Database)
	customHandlerInput.Query = FormatQuery(customHandlerInput.Query)
	err := engine.ValidateDatabase(customHandlerInput)
	if err != nil {
		return err
	}

	err = engine.ValidateMethod(customHandlerInput)
	if err != nil {
		return err
	}

	err = engine.ValidateEndpoint(customHandlerInput)
	if err != nil {
		return err
	}

	err = CheckSQLStringValidity(db, customHandlerInput.Query)
	if err != nil {
		return err
	}

	query := fmt.Sprintf("UPDATE %s.engine_rest_actions SET endpoint = $1, method = $2, db = $3, query = $4, enabled = $5, auth = $6 WHERE id = $7", environment.GetEnvValue("INTERNAL_SCHEMA_NAME"))
	_, err = db.Exec(query, customHandlerInput.Endpoint, customHandlerInput.Method, customHandlerInput.Database, customHandlerInput.Query, customHandlerInput.Enabled, customHandlerInput.Auth, customHandlerInput.Id)

	if err != nil {
		return err
	}

	return nil
}

func (engine *Engine) DeleteRestHandler(db *sql.DB, customHandlerInput CustomRestHandlerInput) error {
	query := fmt.Sprintf("DELETE FROM %s.engine_rest_actions WHERE id = $1", environment.GetEnvValue("INTERNAL_SCHEMA_NAME"))
	_, err := db.Exec(query, customHandlerInput.Id)
	return err
}

func (engine *Engine) DeleteRestHandlerByDatabase(db *sql.DB, customHandlerInput CustomRestHandlerInput) error {
	query := fmt.Sprintf("DELETE FROM %s.engine_rest_actions WHERE db = $1", environment.GetEnvValue("INTERNAL_SCHEMA_NAME"))
	_, err := db.Exec(query, customHandlerInput.Database)
	return err
}
