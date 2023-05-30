package main

import "application/environment"

//BASE ROUTES
var HomeRoute string = "/"
var AliveRoute string = "/alive"

//ENGINE ROUTES
var EngineRoute string = "/engine"
var EngineReloadRoute string = "/engine/reload"

// DATA ROUTES
var QueryRoute string = "/<str:database>"
var StatementRoute string = "/<str:database>/actions"
var ProcessMultipleStatementsRoute string = "/<str:database>/process"

// AUTH ROUTES
var RefreshTokenRoute string = "/auth"
var LoginRoute string = "/auth/login"
var RegisterRoute string = "/auth/register"

// ENGINE AUTH ROUTES
var EngineLoginRoute string = "/engine/auth/login"
var EngineRegisterRoute string = "/engine/auth/register"

// RELATIONS ROUTES
var RelationsRoutes string = "/engine/relations"

// DATABASE ROUTES
var DatabasesRoutes string = "/engine/databases"
var DatabaseRoutes string = "/engine/databases/<str:database>"

// TABLE ROUTES
var TableRoutes string = "/engine/databases/<str:database>/tables"
var TableRoute string = "/engine/databases/<str:database>/tables/<str:table>"

// COLUMNS ROUTES
var ColumnsRoute string = "/engine/databases/<str:database>/tables/<str:table>/columns"

// INDEXES ROUTES
var IndexesRoute string = "/engine/databases/<str:database>/tables/<str:table>/indexes"

// CUSTOM REST HANDLERS ROUTES
var CustomRestHandlersRoute string = "/engine/rest-handlers"

// GRAPHQL ROUTES
var GraphQLRoute string = environment.GetEnvValueToStringWithDefault("GRAPHQL_ENDPOINT", "/graphql")
var GraphiQLRoute string = environment.GetEnvValueToStringWithDefault("GRAPHIQL_ENDPOINT", GraphQLRoute)

// RLS ROUTES
var RLS_DB string = "/engine/rls/database"
var RLS_TABLE string = "/engine/rls/table"
var RLS_TABLE_POLICY string = "/engine/rls/policy"

//WS ROUTES
var WS string = "/ws"
