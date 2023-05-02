package main

import (
	engine "application/engine"
	environment "application/environment"
	"database/sql"

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

	app.Get("/", InfoHandler(app))

	app.Get("/alive", AliveHandler(app))

	// ENGINE ROUTES
	app.Use("/<str:database>", AuthDBMiddleware(app))
	app.Post("/<str:database>", SelectHandler(app, db))

	app.Use("/<str:database>/process", AuthDBMiddleware(app))
	app.Post("/<str:database>/process", ProcessHandler(app, db))

	app.Use("/<str:database>/action", AuthDBMiddleware(app))
	app.Post("/<str:database>/action", InsertHandler(app, db))

	app.Put("/<str:database>/action", UpdateHandler(app, db))

	app.Delete("/<str:database>/action", DeleteHandler(app, db))

	//AUTH ROUTES
	app.Use("/auth", AuthMainMiddleware(app))
	app.Get("/auth", AuthenticateHandler(app, db))

	app.Post("/auth/login", LoginHandler(app, db))

	app.Post("/auth/register", RegisterHandler(app, db))

	app.Listen()
}
