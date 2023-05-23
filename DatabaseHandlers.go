package main

import (
	"application/database"
	"application/engine"
	"database/sql"
	"fmt"
	"net/http"
)

func GetDatabases(app *engine.Router, db *sql.DB) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		databases := app.Engine.Databases
		app.Json(res, http.StatusOK, map[string][]string{"databases": databases})
	}
}

func GetDatabaseTables(app *engine.Router, db *sql.DB) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		params := engine.GetParams(req)
		db := params["database"]
		tablesMap, ok := app.Engine.DatabaseToTableToModelMap[db]

		tables := make([]database.Model, 0)
		if ok {
			for _, value := range tablesMap {
				table := *value
				table.Relations = nil
				tables = append(tables, table)
			}
		}

		app.Json(res, http.StatusOK, map[string]any{"database": db, "tables": tables})
	}
}

func CreateDatabase(app *engine.Router, db *sql.DB) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		body := engine.GetBody(req)
		dbInput, ok := body["database"]
		if !ok {
			app.ErrorResponse(res, http.StatusBadRequest, "invalid payload")
			return
		}

		databaseName, ok := dbInput.(string)
		if !ok {
			app.ErrorResponse(res, http.StatusBadRequest, "invalid payload")
			return
		}

		dbname, err := database.FormatDatabaseName(databaseName)

		if err != nil {
			app.ErrorResponse(res, http.StatusBadRequest, "invalid database name")
			return
		}

		err = database.CreateDataBase(db, dbname)
		if err != nil {
			app.ErrorResponse(res, http.StatusBadRequest, err.Error())
			return
		}

		app.Engine.Reload(db)

		app.Json(res, http.StatusCreated, map[string]string{"message": fmt.Sprintf("Database %s successfully created", dbname)})
	}

}

func DropDatabase(app *engine.Router, db *sql.DB) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		params := engine.GetParams(req)
		dbInput, ok := params["database"]
		if !ok {
			app.ErrorResponse(res, http.StatusBadRequest, "invalid payload")
			return
		}

		dbname, err := database.FormatDatabaseName(dbInput)

		if err != nil {
			app.ErrorResponse(res, http.StatusBadRequest, "invalid database name")
			return
		}

		err = database.DropDataBase(db, dbname)
		if err != nil {
			app.ErrorResponse(res, http.StatusBadRequest, err.Error())
			return
		}

		// OVER HERE DELETE ANY OTHER RELATED STUFF FROM ENGINE
		relationInput := database.DatabaseRelationSchema{
			Database: dbname,
		}
		database.DeleteRelationsByDatabase(db, relationInput)

		restHandlerInput := database.CustomRestHandlerInput{
			Database: dbname,
		}
		app.Engine.DeleteRestHandlerByDatabase(db, restHandlerInput)

		// RELOAD ENGINE IN MEMORY
		app.Engine.Reload(db)

		app.Json(res, http.StatusCreated, map[string]string{"message": fmt.Sprintf("Database %s successfully deleted", dbname)})
	}
}
