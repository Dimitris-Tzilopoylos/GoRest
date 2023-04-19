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
	r := engine.NewApp(db)

	r.Use("*", func(res http.ResponseWriter, req *http.Request, next func(req *http.Request)) {
		enhancedReq, err := r.Engine.Authenticate(req)
		if err != nil {
			r.ErrorResponse(res, 500, err.Error())
			return
		}
		req = enhancedReq
		next(req)
	})

	r.Post("/<str:database>", func(res http.ResponseWriter, req *http.Request) {
		body := engine.GetBody(req)
		params := engine.GetParams(req)
		database := params["database"]
		auth := engine.GetAuth(req)
		role := auth["role_name"]
		x, err := r.Engine.SelectExec(role.(string), db, database, body)
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
