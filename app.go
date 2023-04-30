package main

import (
	database "application/database"
	engine "application/engine"
	environment "application/environment"
	"database/sql"
	"fmt"
	"net/http"

	_ "github.com/lib/pq"
)

func main() {
	environment.LoadEnv()
	connStr := environment.GetEnvValue("CONNECTION_STRING")
	maxConnections := environment.GetEnvValueToIntWithDefault("MAX_CONNECTIONS", 50)
	maxIdleConnections := environment.GetEnvValueToIntWithDefault("MAX_IDLE_CONNECTIONS", 50)

	db, err := sql.Open("postgres", connStr)
	db.SetMaxOpenConns(maxConnections)
	db.SetMaxIdleConns(maxIdleConnections)
	if err != nil {
		panic(err)
	}
	defer db.Close()
	app := engine.NewApp(db)
	defer app.Logger.Sync()

	AuthMainMiddleware := func(res http.ResponseWriter, req *http.Request, next func(req *http.Request)) {
		enhancedReq, err := app.Engine.Authenticate(req)
		if err != nil {
			app.ErrorResponse(res, http.StatusUnauthorized, err.Error())
			return
		}
		req = enhancedReq
		next(req)
	}

	AuthDBMiddleware := func(res http.ResponseWriter, req *http.Request, next func(req *http.Request)) {
		params := engine.GetParams(req)
		database := params["database"]
		enhancedReq, err := app.Engine.AuthenticateForDatabase(req, database)
		if err != nil {
			app.ErrorResponse(res, http.StatusUnauthorized, err.Error())
			return
		}
		req = enhancedReq
		next(req)
	}

	app.Get("/", func(res http.ResponseWriter, req *http.Request) {
		responsePayload := map[string]string{
			"version":    fmt.Sprintf("GoJila Version %s", app.Engine.Version),
			"base":       app.BaseUrl,
			"aliveSince": app.AliveSince,
		}
		app.Json(res, http.StatusOK, responsePayload)

	})

	app.Get("/alive", func(res http.ResponseWriter, req *http.Request) {
		responsePayload := map[string]string{
			"message":    "Api is alive",
			"aliveSince": app.AliveSince,
		}
		app.Json(res, http.StatusOK, responsePayload)

	})

	// ENGINE ROUTES
	app.Use("/<str:database>", AuthDBMiddleware)
	app.Post("/<str:database>", func(res http.ResponseWriter, req *http.Request) {
		body := engine.GetBody(req)
		params := engine.GetParams(req)
		database := params["database"]

		x, err := app.Engine.SelectExec("", db, database, body)
		if err != nil {
			app.ErrorResponse(res, http.StatusInternalServerError, err.Error())
			return
		}
		app.Json(res, http.StatusOK, x)

	})

	app.Use("/<str:database>/process", AuthDBMiddleware)
	app.Post("/<str:database>/process", func(res http.ResponseWriter, req *http.Request) {
		body := engine.GetBody(req)
		params := engine.GetParams(req)
		database := params["database"]

		result, err := app.Engine.Process("", db, database, body)

		if err != nil {
			app.ErrorResponse(res, http.StatusInternalServerError, err.Error())
			return
		}
		app.Json(res, http.StatusCreated, result)

	})

	app.Use("/<str:database>/action", AuthDBMiddleware)
	app.Post("/<str:database>/action", func(res http.ResponseWriter, req *http.Request) {
		body := engine.GetBody(req)
		params := engine.GetParams(req)
		database := params["database"]

		result, err := app.Engine.InsertExec("", db, database, body)

		if err != nil {
			app.ErrorResponse(res, http.StatusInternalServerError, err.Error())
			return
		}
		app.Json(res, http.StatusCreated, result)

	})

	app.Put("/<str:database>/action", func(res http.ResponseWriter, req *http.Request) {
		body := engine.GetBody(req)
		params := engine.GetParams(req)
		database := params["database"]
		result, err := app.Engine.UpdateExec("", db, database, body)
		if err != nil {
			app.ErrorResponse(res, http.StatusInternalServerError, err.Error())
			return
		}
		app.Json(res, http.StatusOK, result)
	})

	app.Delete("/<str:database>/action", func(res http.ResponseWriter, req *http.Request) {
		body := engine.GetBody(req)
		params := engine.GetParams(req)
		database := params["database"]

		result, err := app.Engine.DeleteExec("", db, database, body)

		if err != nil {
			app.ErrorResponse(res, http.StatusInternalServerError, err.Error())
			return
		}
		app.Json(res, http.StatusOK, result)

	})

	//AUTH ROUTES
	app.Use("/auth", AuthMainMiddleware)
	app.Get("/auth", func(res http.ResponseWriter, req *http.Request) {
		auth := engine.GetAuth(req)
		token, err := app.Engine.RefreshToken(auth)
		if err != nil {
			app.ErrorResponse(res, http.StatusUnauthorized, "Unauthorized")
			return
		}

		app.Json(res, http.StatusOK, map[string]string{"token": token})

	})

	app.Post("/auth/login", func(res http.ResponseWriter, req *http.Request) {
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

	})

	app.Post("/auth/register", func(res http.ResponseWriter, req *http.Request) {
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

	})

	app.Listen()
}
