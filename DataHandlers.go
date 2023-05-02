package main

import (
	engine "application/engine"
	"database/sql"
	"net/http"

	_ "github.com/lib/pq"
)

func SelectHandler(app *engine.Router, db *sql.DB) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		body := engine.GetBody(req)
		params := engine.GetParams(req)
		database := params["database"]

		x, err := app.Engine.SelectExec("", db, database, body)
		if err != nil {
			app.ErrorResponse(res, http.StatusInternalServerError, err.Error())
			return
		}
		app.Json(res, http.StatusOK, x)

	}
}

func InsertHandler(app *engine.Router, db *sql.DB) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		body := engine.GetBody(req)
		params := engine.GetParams(req)
		database := params["database"]

		result, err := app.Engine.InsertExec("", db, database, body)

		if err != nil {
			app.ErrorResponse(res, http.StatusInternalServerError, err.Error())
			return
		}
		app.Json(res, http.StatusCreated, result)

	}
}

func UpdateHandler(app *engine.Router, db *sql.DB) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		body := engine.GetBody(req)
		params := engine.GetParams(req)
		database := params["database"]
		result, err := app.Engine.UpdateExec("", db, database, body)
		if err != nil {
			app.ErrorResponse(res, http.StatusInternalServerError, err.Error())
			return
		}
		app.Json(res, http.StatusOK, result)
	}
}

func DeleteHandler(app *engine.Router, db *sql.DB) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		body := engine.GetBody(req)
		params := engine.GetParams(req)
		database := params["database"]

		result, err := app.Engine.DeleteExec("", db, database, body)

		if err != nil {
			app.ErrorResponse(res, http.StatusInternalServerError, err.Error())
			return
		}
		app.Json(res, http.StatusOK, result)

	}
}

func ProcessHandler(app *engine.Router, db *sql.DB) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		body := engine.GetBody(req)
		params := engine.GetParams(req)
		database := params["database"]

		result, err := app.Engine.Process("", db, database, body)

		if err != nil {
			app.ErrorResponse(res, http.StatusInternalServerError, err.Error())
			return
		}
		app.Json(res, http.StatusCreated, result)

	}
}
