package main

import (
	database "application/database"
	engine "application/engine"
	"database/sql"
	"net/http"

	_ "github.com/lib/pq"
)

func LoginHandler(app *engine.Router, db *sql.DB) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		body := engine.GetBody(req)
		bodyDB, ok := body["database"]
		if !ok {
			app.ErrorResponse(res, http.StatusBadRequest, "Database was not provided!")
			return
		}
		table, ok := body["table"]
		if !ok {
			app.ErrorResponse(res, http.StatusBadRequest, "Table was not provided!")
			return
		}

		entry := database.Find(app.Engine.GlobalAuthEntities, func(entry database.GlobalAuthEntity) bool {
			return entry.Database == bodyDB && entry.Table == table
		})

		if entry == nil {
			app.ErrorResponse(res, http.StatusNotFound, "Not Found")
			return
		}

		payload := database.AuthActionPayload{
			Database:      entry.Database,
			Table:         entry.Table,
			Body:          body,
			IdentityField: entry.AuthConfig.IdentifyField,
			PasswordField: entry.AuthConfig.PasswordField,
			Query:         entry.AuthConfig.Query,
		}

		token, err := app.Engine.Login("", db, payload)

		if err != nil {
			app.ErrorResponse(res, http.StatusUnauthorized, err.Error())
			return
		}

		app.Json(res, http.StatusOK, map[string]string{"token": token})

	}
}

func RegisterHandler(app *engine.Router, db *sql.DB) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		body := engine.GetBody(req)
		bodyDB, ok := body["database"]
		if !ok {
			app.ErrorResponse(res, http.StatusBadRequest, "Database was not provided!")
			return
		}
		table, ok := body["table"]
		if !ok {
			app.ErrorResponse(res, http.StatusBadRequest, "Table was not provided!")
			return
		}

		entry := database.Find(app.Engine.GlobalAuthEntities, func(entry database.GlobalAuthEntity) bool {
			return entry.Database == bodyDB && entry.Table == table
		})

		if entry == nil {
			app.ErrorResponse(res, http.StatusNotFound, "Not Found")
			return
		}

		payload := database.AuthActionPayload{
			Database:      entry.Database,
			Table:         entry.Table,
			Body:          body,
			IdentityField: entry.AuthConfig.IdentifyField,
			PasswordField: entry.AuthConfig.PasswordField,
			Query:         entry.AuthConfig.Query,
		}

		result, err := app.Engine.Register("", db, payload)

		if err != nil {
			app.ErrorResponse(res, http.StatusUnauthorized, err.Error())
			return
		}

		app.Json(res, http.StatusOK, result)

	}
}

func AuthenticateHandler(app *engine.Router, db *sql.DB) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		auth := engine.GetAuth(req)
		token, err := app.Engine.RefreshToken(auth)
		if err != nil {
			app.ErrorResponse(res, http.StatusUnauthorized, "Unauthorized")
			return
		}

		app.Json(res, http.StatusOK, map[string]string{"token": token})

	}
}
