package main

import (
	"application/engine"
	"database/sql"
	"net/http"
)

func GetEngineConfigHandler(app *engine.Router) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		engine := app.Engine
		payload := map[string]any{
			"databases": engine.Databases,
			"relations": engine.Relations,
			"models":    engine.EngineModelsToNotCycledValue(),
		}
		app.Json(res, http.StatusOK, payload)
	}
}

func ReloadEngine(app *engine.Router, db *sql.DB) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		app.Engine.Reload(db)
		app.Json(res, http.StatusOK, map[string]string{"message": "Engine has been reloaded!"})
	}
}
