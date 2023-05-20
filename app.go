package main

func main() {

	app, db := NewApplication()
	defer db.Close()
	defer app.Logger.Sync()

	// BASE ROUTES
	app.Get(HomeRoute, InfoHandler(app))
	app.Get(AliveRoute, AliveHandler(app))

	// DATA ROUTES
	app.Use(QueryRoute, AuthDBMiddleware(app))
	app.Post(QueryRoute, SelectHandler(app, db))
	app.Use(ProcessMultipleStatementsRoute, AuthDBMiddleware(app))
	app.Post(ProcessMultipleStatementsRoute, ProcessHandler(app, db))
	app.Use(StatementRoute, AuthDBMiddleware(app))
	app.Post(StatementRoute, InsertHandler(app, db))
	app.Put(StatementRoute, UpdateHandler(app, db))
	app.Delete(StatementRoute, DeleteHandler(app, db))

	// AUTH ROUTES
	app.Use(RefreshTokenRoute, AuthMainMiddleware(app))
	app.Get(RefreshTokenRoute, AuthenticateHandler(app, db))
	app.Post(LoginRoute, LoginHandler(app, db))
	app.Post(RegisterRoute, RegisterHandler(app, db))

	// RELATIONS ROUTES
	app.Use(RelationsRoutes, AuthMainMiddleware(app))
	app.Get(RelationsRoutes, GetRelations(app, db))
	app.Post(RelationsRoutes, CreateRelation(app, db))
	app.Put(RelationsRoutes, UpdateRelation(app, db))
	app.Delete(RelationsRoutes, DeleteRelation(app, db))

	// DATABASE ROUTES
	app.Use(DatabasesRoutes, AuthMainMiddleware(app))
	app.Get(DatabasesRoutes, GetDatabases(app, db))
	app.Post(DatabasesRoutes, CreateDatabase(app, db))
	app.Use(DatabaseRoutes, AuthMainMiddleware(app))
	app.Get(DatabaseRoutes, GetDatabaseTables(app, db))
	app.Delete(DatabaseRoutes, DropDatabase(app, db))

	// TABLE ROUTES
	app.Use(TableRoutes, AuthMainMiddleware(app))
	app.Post(TableRoutes, CreateTable(app, db))
	app.Use(TableRoute, AuthMainMiddleware(app))
	app.Get(TableRoute, GetTable(app, db))
	app.Delete(TableRoute, DropTable(app, db))

	// COLUMN ROUTES
	app.Use(ColumnsRoute, AuthMainMiddleware(app))
	app.Post(ColumnsRoute, CreateColumn(app, db))
	app.Delete(ColumnsRoute, DropColumn(app, db))

	// INDEX ROUTES
	app.Use(IndexesRoute, AuthMainMiddleware(app))
	app.Post(IndexesRoute, CreateIndex(app, db))
	app.Delete(IndexesRoute, DropIndex(app, db))

	// GRAPHQL ROUTES
	app.Use(GraphiQLRoute, AuthMainMiddleware(app))
	app.Get(GraphiQLRoute, GraphqlIntrospection(app, db))
	app.Use(GraphQLRoute, AuthMainMiddleware(app))
	app.Post(GraphQLRoute, GraphqlHandler(app, db))

	app.Listen()
}
