package main

import (
	"application/database"
	"application/engine"
	"database/sql"
	"fmt"
	"net/http"
)

func GetRelations(app *engine.Router, db *sql.DB) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		relations := app.Engine.Relations
		app.Json(res, http.StatusOK, relations)
	}
}

func CreateRelation(app *engine.Router, db *sql.DB) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		relationInput, err := engine.GetBodyIntoStruct(req, database.DatabaseRelationSchema{})

		if err != nil {
			app.ErrorResponse(res, http.StatusBadRequest, err.Error())
			return
		}

		_, err = database.ValidateRelationParts(app.Engine.DatabaseToTableToModelMap, relationInput)
		if err != nil {
			app.ErrorResponse(res, http.StatusBadRequest, err.Error())
			return
		}

		err = database.CreateRelation(db, relationInput)
		if err != nil {
			app.ErrorResponse(res, http.StatusInternalServerError, err.Error())
			return
		}

		app.Engine.Reload(db)

		app.Json(res, http.StatusCreated, map[string]string{"message": fmt.Sprintf("Relation %s created", relationInput.Alias)})
	}
}

func UpdateRelation(app *engine.Router, db *sql.DB) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {

		relationInput, err := engine.GetBodyIntoStruct(req, database.DatabaseRelationSchema{})

		if relationInput.Id <= 0 {
			app.ErrorResponse(res, http.StatusBadRequest, "Please provide relation id for this operation")
			return
		}

		if err != nil {
			app.ErrorResponse(res, http.StatusBadRequest, err.Error())
			return
		}

		_, err = database.ValidateRelationParts(app.Engine.DatabaseToTableToModelMap, relationInput)

		if err != nil {
			app.ErrorResponse(res, http.StatusBadRequest, err.Error())
			return
		}

		err = database.UpdateRelationByID(db, relationInput)
		if err != nil {
			app.ErrorResponse(res, http.StatusInternalServerError, err.Error())
			return
		}

		app.Engine.Reload(db)

		app.Json(res, http.StatusOK, map[string]string{"message": fmt.Sprintf("Relation %s created", relationInput.Alias)})
	}
}

func DeleteRelation(app *engine.Router, db *sql.DB) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {

		relationInput, err := engine.GetBodyIntoStruct(req, database.DatabaseRelationSchema{})

		if relationInput.Id <= 0 {
			app.ErrorResponse(res, http.StatusBadRequest, "Please provide relation id for this operation")
			return
		}

		if err != nil {
			app.ErrorResponse(res, http.StatusBadRequest, err.Error())
			return
		}

		err = database.DeleteRelationByID(db, relationInput)

		if err != nil {
			app.ErrorResponse(res, http.StatusInternalServerError, err.Error())
			return
		}

		app.Engine.Reload(db)

		app.Json(res, http.StatusOK, map[string]string{"message": fmt.Sprintf("Relation %s deleted", relationInput.Alias)})
	}
}
