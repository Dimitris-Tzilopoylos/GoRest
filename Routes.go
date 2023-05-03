package main

//BASE ROUTES
var HomeRoute string = "/"
var AliveRoute string = "/alive"

// DATA ROUTES
var QueryRoute string = "/<str:database>"
var StatementRoute string = "/<str:database>/action"
var ProcessMultipleStatementsRoute string = "/<str:database>/process"

// AUTH ROUTES
var RefreshTokenRoute string = "/auth"
var LoginRoute string = "/auth/login"
var RegisterRoute string = "/auth/register"

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
