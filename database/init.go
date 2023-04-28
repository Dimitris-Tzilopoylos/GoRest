package database

import (
	"application/environment"
	"database/sql"

	"github.com/graphql-go/graphql"
)

type Engine struct {
	Databases                 []string                     `json:"databases"`
	Database                  string                       `json:"database"`
	Models                    []*Model                     `json:"models"`
	DatabaseToTableToModelMap map[string]map[string]*Model `json:"schema"`
	GlobalAuthEntities        []GlobalAuthEntity
	GraphQL                   *graphql.Schema
	InternalSchemaName        string
	Version                   string
}

func Init(db *sql.DB) *Engine {
	models, err := InitializeModels(db)
	if err != nil {
		panic(err)
	}

	schema := make(map[string]map[string]*Model)
	for _, model := range models {
		database := model.Database
		table := model.Table
		_, ok := schema[database]
		if !ok {
			schema[database] = make(map[string]*Model)
		}
		schema[database][table] = model
	}
	engine := &Engine{
		Models:                    models,
		DatabaseToTableToModelMap: schema,
		GlobalAuthEntities:        make([]GlobalAuthEntity, 0),
		InternalSchemaName:        environment.GetEnvValue("INTERNAL_SCHEMA_NAME"),
		Version:                   environment.GetEnvValue("VERSION"),
	}

	engine.LoadGlobalAuth(db)
	engine.BuildGraphQLSchema()
	return engine
}

func (engine *Engine) Reload(db *sql.DB) {
	models, err := InitializeModels(db)
	if err != nil {
		panic(err)
	}
	schema := make(map[string]map[string]*Model)
	for _, model := range models {
		database := model.Database
		table := model.Table
		_, ok := schema[database]
		if !ok {
			schema[database] = make(map[string]*Model)
		}
		schema[database][table] = model
	}
	engine.Models = models
	engine.DatabaseToTableToModelMap = schema
	engine.BuildGraphQLSchema()
}

func InitializeModels(db *sql.DB) ([]*Model, error) {
	relations, _ := GetEngineRelations(db)
	var models []*Model = make([]*Model, 0)
	databases, err := GetDatabases(db)
	if err != nil {
		panic(err)
	}
	for _, database := range databases {
		tables, err := GetTableNames(db, database)
		if err != nil {
			return models, err
		}

		for _, tableName := range tables {
			columns, err := GetTableColumns(db, database, tableName)
			if err != nil {
				return models, err
			}
			indexes, err := GetTableIndexes(db, database, tableName)
			if err != nil {
				return models, err
			}

			model := NewModel(database, tableName)
			model.Columns = columns
			model.Indexes = indexes
			// model.RLS["ADMIN"] = ColumnsMap{"id": "int8"}
			for _, column := range columns {
				model.ColumnsMap[column.Name] = column.Type
			}

			models = append(models, model)
		}
		for _, relation := range relations {
			for _, model := range models {
				if model.Table == relation.FromTable {
					relatedModel := Find(models, func(model *Model) bool {
						return model.Table == relation.ToTable && model.Database == relation.Database
					})
					if relatedModel != nil {
						model.Relations[relation.Alias] = *relatedModel
						model.RelationsInfoMap[relation.Alias] = relation
					}
				}
			}

		}

	}
	return models, nil
}

func GetDatabases(db *sql.DB) ([]string, error) {
	var database string
	databases := []string{}
	scanner := Query(db, GET_DATABASES)
	cb := func(rows *sql.Rows) error {
		err := rows.Scan(&database)
		databases = append(databases, database)
		return err
	}
	err := scanner(cb)
	return databases, err
}

func GetTableNames(db *sql.DB, database string) ([]string, error) {
	var table string
	tables := []string{}
	scanner := Query(db, GET_DATABASE_TABLES, database)
	cb := func(rows *sql.Rows) error {
		err := rows.Scan(&table)
		tables = append(tables, table)
		return err
	}
	err := scanner(cb)

	return tables, err
}

func GetTableColumns(db *sql.DB, database string, table string) ([]Column, error) {
	var column Column
	columns := []Column{}
	var maxLength sql.NullInt64
	var defaultValue sql.NullString
	scanner := Query(db, GET_DATABASE_TABLE_COLUMN, database, table)
	cb := func(rows *sql.Rows) error {
		err := rows.Scan(&column.Name, &column.Type, &maxLength, &column.Nullable, &defaultValue)
		if err != nil {
			panic(err)
		}
		if maxLength.Valid {
			column.MaxLength = maxLength.Int64
		}
		if defaultValue.Valid {
			column.DefaultValue = defaultValue.String
		}
		columns = append(columns, column)
		return err
	}
	err := scanner(cb)

	return columns, err
}

func GetTableIndexes(db *sql.DB, database string, table string) ([]Index, error) {
	var index Index
	indexes := []Index{}

	scanner := Query(db, GET_DATABASE_TABLE_INDEXES, database, table)
	cb := func(rows *sql.Rows) error {
		err := rows.Scan(&index.Name, &index.Table, &index.Column, &index.ReferenceTable, &index.ReferenceColumn, &index.Type)
		if err != nil {
			panic(err)
		}

		indexes = append(indexes, index)
		return err
	}
	err := scanner(cb)
	return indexes, err
}

func GetEngineRelations(db *sql.DB) ([]DatabaseRelationSchema, error) {
	var databaseRelation DatabaseRelationSchema
	relations := []DatabaseRelationSchema{}
	scanner := Query(db, GET_ENGINE_RELATIONS)
	cb := func(rows *sql.Rows) error {
		err := rows.Scan(&databaseRelation.Id, &databaseRelation.Alias, &databaseRelation.Database, &databaseRelation.FromTable, &databaseRelation.FromColumn, &databaseRelation.ToTable, &databaseRelation.ToColumn, &databaseRelation.RelationType)
		if err != nil {
			panic(err)
		}
		relations = append(relations, databaseRelation)
		return err
	}
	err := scanner(cb)

	return relations, err
}
