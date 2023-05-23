package main

import (
	engine "application/engine"
	"application/environment"

	"net/http"
)

func AuthEngineMiddleware(app *engine.Router) func(res http.ResponseWriter, req *http.Request, next func(req *http.Request)) {
	return func(res http.ResponseWriter, req *http.Request, next func(req *http.Request)) {
		disableAuth := environment.GetEnvValue("DISABLE_AUTH") == "ON"
		if disableAuth {
			next(req)
			return
		}
		enhancedReq, err := app.Engine.Authenticate(req)
		if err != nil {
			app.ErrorResponse(res, http.StatusUnauthorized, err.Error())
			return
		}

		auth := engine.GetAuth(enhancedReq)
		if len(auth) == 0 {
			app.ErrorResponse(res, http.StatusUnauthorized, "Unauthorized")
			return
		}

		bypass_all, ok := auth["bypass_all"]
		if !ok {
			app.ErrorResponse(res, http.StatusUnauthorized, "Unauthorized")
			return
		}

		bypassValue, ok := bypass_all.(bool)
		if !ok || !bypassValue {
			app.ErrorResponse(res, http.StatusUnauthorized, "Unauthorized")
			return
		}

		next(enhancedReq)
	}
}

func AuthMainMiddleware(app *engine.Router) func(res http.ResponseWriter, req *http.Request, next func(req *http.Request)) {
	return func(res http.ResponseWriter, req *http.Request, next func(req *http.Request)) {
		disableAuth := environment.GetEnvValue("DISABLE_AUTH") == "ON"
		if disableAuth {
			next(req)
			return
		}
		enhancedReq, err := app.Engine.Authenticate(req)
		if err != nil {
			app.ErrorResponse(res, http.StatusUnauthorized, err.Error())
			return
		}
		next(enhancedReq)
	}
}

func AuthDBMiddleware(app *engine.Router) func(res http.ResponseWriter, req *http.Request, next func(req *http.Request)) {
	return func(res http.ResponseWriter, req *http.Request, next func(req *http.Request)) {
		disableAuth := environment.GetEnvValue("DISABLE_AUTH") == "ON"
		if disableAuth {
			next(req)
			return
		}
		params := engine.GetParams(req)
		database := params["database"]
		enhancedReq, err := app.Engine.AuthenticateForDatabase(req, database)
		if err != nil {
			app.ErrorResponse(res, http.StatusUnauthorized, err.Error())
			return
		}
		next(enhancedReq)
	}
}

func AuthWSMiddleware(app *engine.Router) func(res http.ResponseWriter, req *http.Request, next func(req *http.Request)) {
	return func(res http.ResponseWriter, req *http.Request, next func(req *http.Request)) {
		disableAuth := environment.GetEnvValue("DISABLE_AUTH") == "ON"
		if disableAuth {
			next(req)
			return
		}

		enhancedReq, err := app.Engine.AuthenticateForDatabaseDataTriggers(req)
		if err != nil {
			app.ErrorResponse(res, http.StatusUnauthorized, err.Error())
			return
		}
		next(enhancedReq)
	}
}
