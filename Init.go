package main

import (
	"application/engine"
	"application/environment"
	"database/sql"

	_ "github.com/lib/pq"
)

func NewApplication() (*engine.Router, *sql.DB) {
	environment.LoadEnv()
	connStr := environment.GetEnvValue("CONNECTION_STRING")
	maxConnections := environment.GetEnvValueToIntWithDefault("MAX_CONNECTIONS", 50)
	maxIdleConnections := environment.GetEnvValueToIntWithDefault("MAX_IDLE_CONNECTIONS", 50)

	db, err := sql.Open("postgres", connStr)
	db.SetMaxOpenConns(maxConnections)
	db.SetMaxIdleConns(maxIdleConnections)
	if err != nil {
		db.Close()
		panic(err)
	}

	app := engine.NewApp(db)

	return app, db
}
