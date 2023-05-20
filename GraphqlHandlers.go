package main

import (
	"application/database"
	"application/engine"
	"application/environment"
	"database/sql"
	"encoding/json"
	"net/http"
)

func GraphqlIntrospection(app *engine.Router, db *sql.DB) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		if environment.GetEnvValue("GRAPHIQL") != "ON" {
			app.ErrorResponse(res, http.StatusNotFound, "not found")
			return
		}
		renderGraphiql := app.Engine.GraphiQL()
		renderGraphiql(res)
	}
}

func GraphqlHandler(app *engine.Router, db *sql.DB) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		auth := engine.GetAuth(req)
		body, err := engine.GetBodyIntoStruct(req, &database.GraphQLRequestInput{})
		if err != nil {
			app.ErrorResponse(res, http.StatusBadRequest, err.Error())
			return
		}
		err = app.Engine.GraphQL.ValidateGraphQlRequestInput(body)
		if err != nil {
			app.ErrorResponse(res, http.StatusBadRequest, err.Error())
			return
		}
		queryErrors := app.Engine.GraphQL.Handler.Schema.ValidateWithVariables(body.Query, body.Variables)
		if len(queryErrors) > 0 {
			app.Json(res, http.StatusBadRequest, queryErrors)
			return
		}
		if body.OperationName == "IntrospectionQuery" {
			introspection, err := app.Engine.GetIntrospectionQueryResponse()
			if err != nil {
				app.ErrorResponse(res, http.StatusInternalServerError, err.Error())
				return
			}
			app.Json(res, http.StatusOK, introspection)
			return
		}

		parsedBody, err := app.Engine.GraphQL.GraphqlParser(body.Query, body.Variables)
		if err != nil {
			app.ErrorResponse(res, http.StatusInternalServerError, err.Error())
			return
		}
		queryResults, err := app.Engine.GraphqlQueryResolve(parsedBody, auth, db)
		if err != nil {
			app.ErrorResponse(res, http.StatusInternalServerError, err.Error())
			return
		}

		resolveResults := make(map[string]any)

		queryParsedResults := make(map[string]any)
		if len(queryResults) > 0 {
			err = json.Unmarshal(queryResults, &queryParsedResults)
			if err != nil {
				app.ErrorResponse(res, http.StatusInternalServerError, err.Error())
				return
			}
		}

		resolveResults["data"] = queryParsedResults

		mutationResults, err := app.Engine.GraphqlMutationResolve(parsedBody, auth, db)
		if err != nil {
			app.ErrorResponse(res, http.StatusInternalServerError, err.Error())
			return
		}

		parsedMutationResults, err := database.IsMapToInterface(mutationResults)
		if err != nil {
			app.ErrorResponse(res, http.StatusInternalServerError, err.Error())
			return
		}

		mutationResponsePayload := make(map[string]any)
		for key, value := range parsedMutationResults {
			mutationResponsePayload[key] = value
		}
		if len(mutationResponsePayload) > 0 {
			resolveResults["data"] = mutationResponsePayload
		}

		app.Json(res, http.StatusOK, resolveResults)

	}
}
