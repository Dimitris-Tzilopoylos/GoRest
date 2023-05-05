package main

import (
	"application/database"
	"application/engine"
	"database/sql"
	"fmt"
	"net/http"
)

func GetRestHandlers(app *engine.Router, db *sql.DB) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {

		app.Json(res, http.StatusCreated, app.Engine.RestHandlers)

	}
}

func CreateRestHandler(app *engine.Router, db *sql.DB) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {

		restHandlerInput, err := engine.GetBodyIntoStruct(req, database.CustomRestHandlerInput{})
		if err != nil {
			app.ErrorResponse(res, http.StatusBadRequest, err.Error())
			return
		}

		err = app.Engine.CreateRestHandler(db, restHandlerInput)
		if err != nil {
			app.ErrorResponse(res, http.StatusBadRequest, err.Error())
			return
		}

		app.Engine.Reload(db)
		app.Json(res, http.StatusCreated, map[string]string{"message": fmt.Sprintf("rest handler [%s]: %s created", restHandlerInput.Method, restHandlerInput.Endpoint)})

	}
}

func UpdateRestHandler(app *engine.Router, db *sql.DB) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {

		restHandlerInput, err := engine.GetBodyIntoStruct(req, database.CustomRestHandlerInput{})
		if err != nil {
			app.ErrorResponse(res, http.StatusBadRequest, err.Error())
			return
		}

		err = app.Engine.UpdateRestHandler(db, restHandlerInput)
		if err != nil {
			app.ErrorResponse(res, http.StatusBadRequest, err.Error())
			return
		}

		app.Engine.Reload(db)
		app.Json(res, http.StatusCreated, map[string]string{"message": fmt.Sprintf("rest handler [%s]: %s updated", restHandlerInput.Method, restHandlerInput.Endpoint)})

	}
}

func DeleteRestHandler(app *engine.Router, db *sql.DB) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {

		restHandlerInput, err := engine.GetBodyIntoStruct(req, database.CustomRestHandlerInput{})
		if err != nil {
			app.ErrorResponse(res, http.StatusBadRequest, err.Error())
			return
		}

		err = app.Engine.DeleteRestHandler(db, restHandlerInput)
		if err != nil {
			app.ErrorResponse(res, http.StatusBadRequest, err.Error())
			return
		}

		app.Engine.Reload(db)
		app.Json(res, http.StatusCreated, map[string]string{"message": "rest handler deleted"})

	}
}
