package main

import (
	"application/database"
	"application/engine"
	"database/sql"
	"fmt"
	"net/http"
)

func CreateIndex(app *engine.Router, db *sql.DB) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		indexInput, err := engine.GetBodyIntoStruct(req, database.IndexInput{})
		if err != nil {
			app.ErrorResponse(res, http.StatusBadRequest, err.Error())
			return
		}

		tableInput := database.TableInput{
			Database: indexInput.Database,
			Name:     indexInput.Table,
			Indexes:  []database.IndexInput{indexInput},
		}

		err = database.CreateIndex(db, tableInput, indexInput)

		if err != nil {
			app.ErrorResponse(res, http.StatusBadRequest, err.Error())
			return
		}

		app.Engine.Reload(db)

		app.Json(res, http.StatusCreated, map[string]string{"message": fmt.Sprintf("Index for table %s of database %s successfully created", tableInput.Name, tableInput.Database)})
	}
}

func DropIndex(app *engine.Router, db *sql.DB) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		indexInput, err := engine.GetBodyIntoStruct(req, database.IndexInput{})

		if err != nil {
			app.ErrorResponse(res, http.StatusBadRequest, err.Error())
			return
		}

		if len(indexInput.Name) == 0 {
			app.ErrorResponse(res, http.StatusBadRequest, "no index name was provided")
			return
		}

		err = database.DropIndex(db, indexInput.Name)

		if err != nil {
			app.ErrorResponse(res, http.StatusBadRequest, err.Error())
			return
		}

		app.Engine.Reload(db)

		app.Json(res, http.StatusCreated, map[string]string{"message": fmt.Sprintf("Index %s successfully deleted successfully created", indexInput.Name)})
	}
}
