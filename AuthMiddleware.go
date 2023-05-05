package main

import (
	engine "application/engine"
	"application/environment"

	"net/http"
)

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
