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

// DATABASE ROUTES
var DatabasesRoutes string = "/engine/databases"
var DatabaseRoutes string = "/engine/databases/<str:database>"
