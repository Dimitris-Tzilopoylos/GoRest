package main

import (
	engine "application/engine"
	"fmt"

	"net/http"
)

func InfoHandler(app *engine.Router) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		responsePayload := map[string]string{
			"version":    fmt.Sprintf("GoJila Version %s", app.Engine.Version),
			"base":       app.BaseUrl,
			"aliveSince": app.AliveSince,
		}
		app.Json(res, http.StatusOK, responsePayload)

	}
}

func AliveHandler(app *engine.Router) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		responsePayload := map[string]string{
			"message":    "Api is alive",
			"aliveSince": app.AliveSince,
		}
		app.Json(res, http.StatusOK, responsePayload)

	}
}
