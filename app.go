package main

import (
	database "application/database"
	engine "application/engine"
	environment "application/environment"
	"database/sql"
	"net/http"

	_ "github.com/lib/pq"
)

func main() {
	environment.LoadEnv()
	connStr := environment.GetEnvValue("CONNECTION_STRING")
	entryPoint := environment.GetEnvValue("ROUTER_ENTRY_POINT")
	port := environment.GetEnvValue("PORT")

	db, err := sql.Open("postgres", connStr)
	db.SetMaxOpenConns(70)
	db.SetMaxIdleConns(70)
	if err != nil {
		panic(err)
	}
	defer db.Close()
	r := engine.NewApp(db, entryPoint)
	defer r.Logger.Sync()

	AuthMainMiddleware := func(res http.ResponseWriter, req *http.Request, next func(req *http.Request)) {
		enhancedReq, err := r.Engine.Authenticate(req)
		if err != nil {
			r.ErrorResponse(res, http.StatusUnauthorized, err.Error())
			return
		}
		req = enhancedReq
		next(req)
	}

	AuthDBMiddleware := func(res http.ResponseWriter, req *http.Request, next func(req *http.Request)) {
		params := engine.GetParams(req)
		database := params["database"]
		enhancedReq, err := r.Engine.AuthenticateForDatabase(req, database)
		if err != nil {
			r.ErrorResponse(res, http.StatusUnauthorized, err.Error())
			return
		}
		req = enhancedReq
		next(req)
	}

	// ENGINE ROUTES
	r.Use("/<str:database>", AuthDBMiddleware)
	r.Post("/<str:database>", func(res http.ResponseWriter, req *http.Request) {
		body := engine.GetBody(req)
		params := engine.GetParams(req)
		database := params["database"]

		x, err := r.Engine.SelectExec("", db, database, body)
		if err != nil {
			r.ErrorResponse(res, http.StatusInternalServerError, err.Error())
			return
		}
		r.Json(res, http.StatusOK, x)

	})

	r.Use("/<str:database>/process", AuthDBMiddleware)
	r.Post("/<str:database>/process", func(res http.ResponseWriter, req *http.Request) {
		body := engine.GetBody(req)
		params := engine.GetParams(req)
		database := params["database"]

		result, err := r.Engine.Process("", db, database, body)

		if err != nil {
			r.ErrorResponse(res, http.StatusInternalServerError, err.Error())
			return
		}
		r.Json(res, http.StatusCreated, result)

	})

	r.Use("/<str:database>/action", AuthDBMiddleware)
	r.Post("/<str:database>/action", func(res http.ResponseWriter, req *http.Request) {
		body := engine.GetBody(req)
		params := engine.GetParams(req)
		database := params["database"]

		result, err := r.Engine.InsertExec("", db, database, body)

		if err != nil {
			r.ErrorResponse(res, http.StatusInternalServerError, err.Error())
			return
		}
		r.Json(res, http.StatusCreated, result)

	})

	r.Put("/<str:database>/action", func(res http.ResponseWriter, req *http.Request) {
		body := engine.GetBody(req)
		params := engine.GetParams(req)
		database := params["database"]
		result, err := r.Engine.UpdateExec("", db, database, body)
		if err != nil {
			r.ErrorResponse(res, http.StatusInternalServerError, err.Error())
			return
		}
		r.Json(res, http.StatusOK, result)
	})

	r.Delete("/<str:database>/action", func(res http.ResponseWriter, req *http.Request) {
		body := engine.GetBody(req)
		params := engine.GetParams(req)
		database := params["database"]

		result, err := r.Engine.DeleteExec("", db, database, body)

		if err != nil {
			r.ErrorResponse(res, http.StatusInternalServerError, err.Error())
			return
		}
		r.Json(res, http.StatusOK, result)

	})

	//AUTH ROUTES
	r.Use("/auth", AuthMainMiddleware)
	r.Get("/auth", func(res http.ResponseWriter, req *http.Request) {
		auth := engine.GetAuth(req)
		token, err := r.Engine.RefreshToken(auth)
		if err != nil {
			r.ErrorResponse(res, http.StatusUnauthorized, "Unauthorized")
			return
		}

		r.Json(res, http.StatusOK, map[string]string{"token": token})

	})

	r.Post("/auth/login", func(res http.ResponseWriter, req *http.Request) {
		body := engine.GetBody(req)
		bodyDB, ok := body["database"]
		if !ok {
			r.ErrorResponse(res, http.StatusBadRequest, "Database was not provided!")
			return
		}
		table, ok := body["table"]
		if !ok {
			r.ErrorResponse(res, http.StatusBadRequest, "Table was not provided!")
			return
		}

		entry := database.Find(r.Engine.GlobalAuthEntities, func(entry database.GlobalAuthEntity) bool {
			return entry.Database == bodyDB && entry.Table == table
		})

		if entry == nil {
			r.ErrorResponse(res, http.StatusNotFound, "Not Found")
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

		token, err := r.Engine.Login("", db, payload)

		if err != nil {
			r.ErrorResponse(res, http.StatusUnauthorized, err.Error())
			return
		}

		r.Json(res, http.StatusOK, map[string]string{"token": token})

	})

	r.Post("/auth/register", func(res http.ResponseWriter, req *http.Request) {
		body := engine.GetBody(req)
		bodyDB, ok := body["database"]
		if !ok {
			r.ErrorResponse(res, http.StatusBadRequest, "Database was not provided!")
			return
		}
		table, ok := body["table"]
		if !ok {
			r.ErrorResponse(res, http.StatusBadRequest, "Table was not provided!")
			return
		}

		entry := database.Find(r.Engine.GlobalAuthEntities, func(entry database.GlobalAuthEntity) bool {
			return entry.Database == bodyDB && entry.Table == table
		})

		if entry == nil {
			r.ErrorResponse(res, http.StatusNotFound, "Not Found")
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

		result, err := r.Engine.Register("", db, payload)

		if err != nil {
			r.ErrorResponse(res, http.StatusUnauthorized, err.Error())
			return
		}

		r.Json(res, http.StatusOK, result)

	})

	r.Listen(port)
}
