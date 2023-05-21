package main

import (
	database "application/database"
	engine "application/engine"
	"database/sql"
	"fmt"
	"net/http"
)

func EnableRlsForDatabase(app *engine.Router, db *sql.DB) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		input, err := engine.GetBodyIntoStruct(req, database.EnableRLSForDatabaseInput{})
		if err != nil {
			app.ErrorResponse(res, http.StatusBadRequest, err.Error())
			return
		}
		err = app.Engine.EnableRLSForDatabase(db, input)
		if err != nil {
			app.ErrorResponse(res, http.StatusInternalServerError, err.Error())
			return
		}

		app.Json(res, http.StatusOK, map[string]string{"message": fmt.Sprintf("Row level security for database %s: Enabled", input.Database)})
	}
}

func DisableRlsForDatabase(app *engine.Router, db *sql.DB) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		input, err := engine.GetBodyIntoStruct(req, database.EnableRLSForDatabaseInput{})
		if err != nil {
			app.ErrorResponse(res, http.StatusBadRequest, err.Error())
			return
		}
		err = app.Engine.DisableRLSForDatabase(db, input)
		if err != nil {
			app.ErrorResponse(res, http.StatusInternalServerError, err.Error())
			return
		}

		app.Json(res, http.StatusOK, map[string]string{"message": fmt.Sprintf("Row level security for database %s: Disabled", input.Database)})
	}
}

func EnableRlsForTable(app *engine.Router, db *sql.DB) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		input, err := engine.GetBodyIntoStruct(req, database.EnableRlsForDatabaseTableInput{})
		if err != nil {
			app.ErrorResponse(res, http.StatusBadRequest, err.Error())
			return
		}
		input.Force = true
		err = app.Engine.EnableRLSForTable(db, input)
		if err != nil {
			app.ErrorResponse(res, http.StatusInternalServerError, err.Error())
			return
		}

		app.Json(res, http.StatusOK, map[string]string{"message": fmt.Sprintf("Row level security for table %s of database %s: Enabled", input.Table, input.Database)})
	}
}

func DisableRlsForTable(app *engine.Router, db *sql.DB) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		input, err := engine.GetBodyIntoStruct(req, database.EnableRlsForDatabaseTableInput{})
		if err != nil {
			app.ErrorResponse(res, http.StatusBadRequest, err.Error())
			return
		}
		input.Force = true
		err = app.Engine.DisableRLSForTable(db, input)
		if err != nil {
			app.ErrorResponse(res, http.StatusInternalServerError, err.Error())
			return
		}

		app.Json(res, http.StatusOK, map[string]string{"message": fmt.Sprintf("Row level security for table %s of database %s: Disabled", input.Table, input.Database)})
	}
}

func CreateRLSPolicy(app *engine.Router, db *sql.DB) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		input, err := engine.GetBodyIntoStruct(req, database.RLSInput{})
		if err != nil {
			app.ErrorResponse(res, http.StatusBadRequest, err.Error())
			return
		}

		err = app.Engine.CreateEngineRLS(db, input)
		if err != nil {
			app.ErrorResponse(res, http.StatusInternalServerError, err.Error())
			return
		}

		app.Json(res, http.StatusCreated, map[string]string{"message": fmt.Sprintf("Policy %s for table %s of database %s: Created", input.PolicyName, input.Table, input.Database)})
	}
}

func DeletePolicy(app *engine.Router, db *sql.DB) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		input, err := engine.GetBodyIntoStruct(req, database.RLSInput{})
		if err != nil {
			app.ErrorResponse(res, http.StatusBadRequest, err.Error())
			return
		}

		err = app.Engine.DropEngineRLS(db, input)
		if err != nil {
			app.ErrorResponse(res, http.StatusInternalServerError, err.Error())
			return
		}

		app.Json(res, http.StatusOK, map[string]string{"message": fmt.Sprintf("Policy %s for table %s of database %s: Deleted", input.PolicyName, input.Table, input.Database)})
	}
}
