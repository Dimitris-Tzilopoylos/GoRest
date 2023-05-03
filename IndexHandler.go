package main

import (
	"application/engine"
	"database/sql"
	"net/http"
)

func CreateIndex(app *engine.Router, db *sql.DB) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {

	}
}
