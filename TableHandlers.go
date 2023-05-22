package main

import (
	"application/database"
	"application/engine"
	"database/sql"
	"fmt"
	"net/http"
)

type TableInput struct {
	Database string                 `json:"database"`
	Name     string                 `json:"name"`
	Columns  []database.ColumnInput `json:"columns"`
}

func GetTable(app *engine.Router, db *sql.DB) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		params := engine.GetParams(req)
		db := params["database"]
		tbl := params["table"]
		tablesMap, ok := app.Engine.DatabaseToTableToModelMap[db]

		if !ok {
			app.NotFound(res, req)
			return
		}
		table, ok := tablesMap[tbl]
		if !ok {
			app.NotFound(res, req)
			return
		}

		response := *table

		response.Relations = nil

		app.Json(res, http.StatusOK, map[string]any{"table": response})
	}
}

func CreateTable(app *engine.Router, db *sql.DB) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {

		payload, err := engine.GetBodyIntoStruct(req, TableInput{})

		if err != nil {
			app.ErrorResponse(res, http.StatusBadRequest, "invalid payload")
			return
		}

		dbname, err := database.FormatDatabaseName(payload.Database)
		if err != nil {
			app.ErrorResponse(res, http.StatusBadRequest, "invalid payload")
			return
		}

		tblname, err := database.FormatTableName(payload.Name)
		if err != nil {
			app.ErrorResponse(res, http.StatusBadRequest, "invalid payload")
			return
		}

		tableInput := database.TableInput{
			Database: dbname,
			Name:     tblname,
			Columns:  payload.Columns,
		}

		err = database.CreateTable(db, tableInput)
		if err != nil {
			app.ErrorResponse(res, http.StatusInternalServerError, fmt.Sprintf("Table %s for Database %s was not created", tblname, dbname))
			return
		}

		app.Engine.Reload(db)
		app.Json(res, http.StatusCreated, map[string]string{"message": fmt.Sprintf("Table %s for Database %s was successfully created", tblname, dbname)})
	}
}

func DropTable(app *engine.Router, db *sql.DB) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		params := engine.GetParams(req)
		dbFromParams := params["database"]
		tbl := params["table"]

		dbname, err := database.FormatDatabaseName(dbFromParams)
		if err != nil {
			app.ErrorResponse(res, http.StatusBadRequest, "invalid payload")
			return
		}

		tblname, err := database.FormatTableName(tbl)
		if err != nil {
			app.ErrorResponse(res, http.StatusBadRequest, "invalid payload")
			return
		}

		tableInput := database.TableInput{
			Database: dbname,
			Name:     tblname,
		}

		err = database.DropTable(db, tableInput)

		if err != nil {
			app.ErrorResponse(res, http.StatusInternalServerError, fmt.Sprintf("Table %s for Database %s was not deleted", tblname, dbname))
			return
		}

		// HANDLE DELETE RELEVANT ENTITIES: e.g relations
		relationInput := database.DatabaseRelationSchema{
			Database: dbname,
			Table:    tblname,
		}
		database.DeleteRelationsByDatabaseTable(db, relationInput)

		// RELOAD ENGINE IN MEMORY
		app.Engine.Reload(db)
		app.Json(res, http.StatusOK, map[string]string{"message": fmt.Sprintf("Table %s for Database %s was successfully deleted", tblname, dbname)})
	}
}
