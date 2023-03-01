package main

import (
	"application/database"
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
	db.SetMaxOpenConns(50)
	db.SetMaxIdleConns(50)
	if err != nil {
		panic(err)
	}
	defer db.Close()
	r := engine.NewApp()
	r.Get("/", func(res http.ResponseWriter, req *http.Request) {
		var rr int
		var todos []int = []int{}
		scanner := database.Query(db, "SELECT id FROM eshop.products")
		cb := func(rows *sql.Rows) error {
			err := rows.Scan(&rr)
			todos = append(todos, rr)
			return err
		}
		scanner(cb)
		r.Json(res, 200, todos)
	})
	r.Get("/<str:test>", func(res http.ResponseWriter, req *http.Request) {
		fmt.Println("fff")
		r.Json(res, 200, "containers")
	})
	r.Listen(fmt.Sprintf(":%s", os.Getenv("PORT")))
	// PARAMS are defined in routes as <type:name>
}
