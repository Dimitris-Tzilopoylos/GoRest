package main

import (
	"application/database"
	"application/engine"
	"database/sql"
	"fmt"
	"net/http"
)

func CreateColumn(app *engine.Router, db *sql.DB) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {

		tableInput, err := engine.GetBodyIntoStruct(req, database.TableInput{})
		if err != nil {
			app.ErrorResponse(res, http.StatusBadRequest, err.Error())
			return
		}

		_, err = database.ValidateCreateColumn(app.Engine.DatabaseToTableToModelMap, tableInput)
		if err != nil {
			app.ErrorResponse(res, http.StatusBadRequest, err.Error())
			return
		}

		err = database.CreateColumn(db, tableInput)
		if err != nil {
			app.ErrorResponse(res, http.StatusBadRequest, err.Error())
			return
		}

		app.Engine.Reload(db)
		app.Json(res, http.StatusCreated, map[string]string{"message": fmt.Sprintf("Column %s for table %s of database %s successfully created", tableInput.Columns[0].Name, tableInput.Name, tableInput.Database)})

	}
}

func DropColumn(app *engine.Router, db *sql.DB) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {

		tableInput, err := engine.GetBodyIntoStruct(req, database.TableInput{})
		if err != nil {
			app.ErrorResponse(res, http.StatusBadRequest, err.Error())
			return
		}

		_, err = database.ValidateDropColumn(app.Engine.DatabaseToTableToModelMap, tableInput)
		if err != nil {
			app.ErrorResponse(res, http.StatusBadRequest, err.Error())
			return
		}

		err = database.DropColumn(db, tableInput)
		if err != nil {
			app.ErrorResponse(res, http.StatusBadRequest, err.Error())
			return
		}

		relationInput := database.DatabaseRelationSchema{
			Table:      tableInput.Name,
			Database:   tableInput.Database,
			FromColumn: tableInput.Columns[0].Name,
			ToColumn:   tableInput.Columns[0].Name,
		}
		database.DeleteRelationsByDatabaseTableColumn(db, relationInput)

		app.Engine.Reload(db)
		app.Json(res, http.StatusCreated, map[string]string{"message": fmt.Sprintf("Column %s for table %s of database %s successfully created", tableInput.Columns[0].Name, tableInput.Name, tableInput.Database)})

	}
}
