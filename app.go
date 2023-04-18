package main

import (
	engine "application/engine"
	"database/sql"
	"fmt"
	"net/http"
	"os"

	_ "github.com/lib/pq"
)

func main() {

	connStr := "postgresql://postgres:postgres@localhost:5432/postgres?sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	db.SetMaxOpenConns(70)
	db.SetMaxIdleConns(70)
	if err != nil {
		panic(err)
	}
	defer db.Close()
	r := engine.NewApp(db, "eshop")

	r.Post("/", func(res http.ResponseWriter, req *http.Request) {
		body := engine.GetBody(req)
		x, err := r.Engine.SelectExec(db, body)
		if err != nil {
			r.ErrorResponse(res, 500, err.Error())
			return
		}
		r.Json(res, 200, x)

	})
	r.Get("/<str:test>", func(res http.ResponseWriter, req *http.Request) {
		r.Json(res, 200, "containers")
	})
	r.Listen(fmt.Sprintf(":%s", os.Getenv("PORT")))
	// PARAMS are defined in routes as <type:name>
}
