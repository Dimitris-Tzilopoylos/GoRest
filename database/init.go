package database

import (
	"application/environment"
	"database/sql"
	"log"
	"strings"

	"github.com/graphql-go/graphql"
)

type Engine struct {
	Databases                 []string                     `json:"databases"`
	Database                  string                       `json:"database"`
	Models                    []*Model                     `json:"models"`
	DatabaseToTableToModelMap map[string]map[string]*Model `json:"schema"`
	GlobalAuthEntities        []GlobalAuthEntity
	GraphQL                   *graphql.Schema
	EventEmitter              *EventEmitter
	InternalSchemaName        string
	Version                   string
	Webhooks                  map[string]map[string]map[string]map[string][]Webhook
	DataTriggers              map[string]map[string]DataTrigger
}

func Init(db *sql.DB) *Engine {
	InitializeEngineDatabase(db)

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
		EventEmitter:              NewEventEmitter(),
	}

	engine.LoadGlobalAuth(db)
	engine.LoadWebhooks(db)
	engine.LoadDataTriggers(db)
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
	engine.LoadGlobalAuth(db)
	engine.LoadWebhooks(db)
	engine.LoadDataTriggers(db)
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

	indexes := []Index{}

	scanner := Query(db, GET_DATABASE_TABLE_INDEXES, database, table)
	cb := func(rows *sql.Rows) error {
		var index Index
		err := rows.Scan(&index.Name, &index.Table, &index.Column, &index.ReferenceTable, &index.ReferenceColumn, &index.Type)
		if err != nil {
			panic(err)
		}

		indexes = append(indexes, index)
		return err
	}
	err := scanner(cb)

	scannerUnique := Query(db, GET_UNIQUE_INDXES, database, table)
	unqCB := func(rows *sql.Rows) error {
		var index Index
		var unq bool = false
		err = rows.Scan(&index.Database, &index.Table, &index.Name, &unq, &index.Column)
		if err != nil {
			panic(err)
		}

		if unq {
			columns := strings.Split(index.Column, ",")
			for _, col := range columns {
				colName := strings.Trim(col, " ")
				newIndex := Index{
					Name:   index.Name,
					Type:   "UNIQUE KEY",
					Table:  index.Table,
					Column: colName,
				}
				indexes = append(indexes, newIndex)
			}
		}

		return err
	}

	err = scannerUnique(unqCB)

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

func InitializeEngineDatabase(db *sql.DB) {
	err := CreateDataBase(db, environment.GetEnvValue("INTERNAL_SCHEMA_NAME"))
	if err != nil {
		log.Fatal(err)
	}
	CreateEngineLogsTable(db)
	CreateEngineWebhooksTable(db)
	CreateEngineAuthProviderTable(db)
	CreateEngineDataTriggersTable(db)
	CreateEngineRelationsTable(db)
	CreateEngineApiKeysTable(db)
}

func CreateEngineLogsTable(db *sql.DB) {
	columns := []ColumnInput{}
	columns = append(columns, ColumnInput{
		Name:          "id",
		Type:          "int",
		Nullable:      false,
		AutoIncrement: true,
	})
	columns = append(columns, ColumnInput{
		Name:      "log_type",
		Type:      "varchar",
		Nullable:  false,
		MaxLength: 255,
	})
	columns = append(columns, ColumnInput{
		Name:         "created_at",
		Type:         "timestamp",
		Nullable:     false,
		DefaultValue: "CURRENT_TIMESTAMP",
	})
	columns = append(columns, ColumnInput{
		Name:     "log_data",
		Type:     "json",
		Nullable: false,
	})

	indexes := []IndexInput{}

	primaryIndexColumn := ColumnInput{
		Name:          "id",
		Type:          "int",
		Nullable:      false,
		AutoIncrement: true,
	}

	primaryIndex := IndexInput{
		Columns: []ColumnInput{
			primaryIndexColumn,
		},
		Type: PRIMARY,
	}

	indexes = append(indexes, primaryIndex)

	table := TableInput{
		Database: environment.GetEnvValue("INTERNAL_SCHEMA_NAME"),
		Name:     "engine_logs",
		Columns:  columns,
		Indexes:  indexes,
	}

	CreateTable(db, table)

	CreateIndexes(db, table)

}

func CreateEngineWebhooksTable(db *sql.DB) {
	columns := []ColumnInput{}
	columns = append(columns, ColumnInput{
		Name:          "id",
		Type:          "int",
		Nullable:      false,
		AutoIncrement: true,
	})
	columns = append(columns, ColumnInput{
		Name:      "endpoint",
		Type:      "varchar",
		Nullable:  false,
		MaxLength: 255,
	})
	columns = append(columns, ColumnInput{
		Name:      "db",
		Type:      "varchar",
		Nullable:  false,
		MaxLength: 255,
	})
	columns = append(columns, ColumnInput{
		Name:      "db_table",
		Type:      "varchar",
		Nullable:  false,
		MaxLength: 255,
	})
	columns = append(columns, ColumnInput{
		Name:      "operation",
		Type:      "varchar",
		Nullable:  false,
		MaxLength: 255,
	})
	columns = append(columns, ColumnInput{
		Name:         "enabled",
		Type:         "boolean",
		Nullable:     false,
		DefaultValue: false,
	})
	columns = append(columns, ColumnInput{
		Name:         "rest",
		Type:         "boolean",
		Nullable:     false,
		DefaultValue: false,
	})
	columns = append(columns, ColumnInput{
		Name:         "graphql",
		Type:         "boolean",
		Nullable:     false,
		DefaultValue: false,
	})
	columns = append(columns, ColumnInput{
		Name:         "forward_auth_headers",
		Type:         "boolean",
		Nullable:     false,
		DefaultValue: false,
	})
	columns = append(columns, ColumnInput{
		Name:         "type",
		Type:         "varchar",
		Nullable:     false,
		MaxLength:    255,
		DefaultValue: "'POST_EXEC'",
	})
	columns = append(columns, ColumnInput{
		Name:         "created_at",
		Type:         "timestamp",
		Nullable:     false,
		DefaultValue: "CURRENT_TIMESTAMP",
	})

	indexes := []IndexInput{}

	primaryIndexColumn := ColumnInput{
		Name:          "id",
		Type:          "int",
		Nullable:      false,
		AutoIncrement: true,
	}

	primaryIndex := IndexInput{
		Columns: []ColumnInput{
			primaryIndexColumn,
		},
		Type: PRIMARY,
	}

	uniqueEndpoint := ColumnInput{
		Name:      "endpoint",
		Type:      "varchar",
		Nullable:  false,
		MaxLength: 255,
	}

	uniqueDB := ColumnInput{
		Name:      "db",
		Type:      "varchar",
		Nullable:  false,
		MaxLength: 255,
	}

	uniqueDBTable := ColumnInput{
		Name:      "db_table",
		Type:      "varchar",
		Nullable:  false,
		MaxLength: 255,
	}

	uniqueOperation := ColumnInput{
		Name:      "operation",
		Type:      "varchar",
		Nullable:  false,
		MaxLength: 255,
	}

	uniqueColumns := []ColumnInput{}

	uniqueColumns = append(uniqueColumns, uniqueEndpoint)
	uniqueColumns = append(uniqueColumns, uniqueDB)
	uniqueColumns = append(uniqueColumns, uniqueDBTable)
	uniqueColumns = append(uniqueColumns, uniqueOperation)

	uniqueIndex := IndexInput{
		Columns: uniqueColumns,
		Type:    UNIQUE,
	}

	indexes = append(indexes, primaryIndex)
	indexes = append(indexes, uniqueIndex)

	table := TableInput{
		Database: environment.GetEnvValue("INTERNAL_SCHEMA_NAME"),
		Name:     "engine_webhooks",
		Columns:  columns,
		Indexes:  indexes,
	}
	CreateTable(db, table)
	CreateIndexes(db, table)

}

func CreateEngineAuthProviderTable(db *sql.DB) {
	columns := []ColumnInput{}
	columns = append(columns, ColumnInput{
		Name:          "id",
		Type:          "bigint",
		Nullable:      false,
		AutoIncrement: true,
	})
	columns = append(columns, ColumnInput{
		Name:     "auth_config",
		Type:     "jsonb",
		Nullable: false,
	})
	columns = append(columns, ColumnInput{
		Name:      "db",
		Type:      "varchar",
		Nullable:  false,
		MaxLength: 255,
	})
	columns = append(columns, ColumnInput{
		Name:      "tbl",
		Type:      "varchar",
		Nullable:  false,
		MaxLength: 255,
	})
	columns = append(columns, ColumnInput{
		Name:         "created_at",
		Type:         "timestamp",
		Nullable:     false,
		DefaultValue: "CURRENT_TIMESTAMP",
	})

	table := TableInput{
		Database: environment.GetEnvValue("INTERNAL_SCHEMA_NAME"),
		Name:     "engine_auth_provider",
		Columns:  columns,
	}

	CreateTable(db, table)

}

func CreateEngineDataTriggersTable(db *sql.DB) {
	columns := []ColumnInput{}
	columns = append(columns, ColumnInput{
		Name:          "id",
		Type:          "bigint",
		Nullable:      false,
		AutoIncrement: true,
	})
	columns = append(columns, ColumnInput{
		Name:     "trigger_config",
		Type:     "jsonb",
		Nullable: false,
	})
	columns = append(columns, ColumnInput{
		Name:      "db",
		Type:      "varchar",
		Nullable:  false,
		MaxLength: 255,
	})
	columns = append(columns, ColumnInput{
		Name:      "tbl",
		Type:      "varchar",
		Nullable:  false,
		MaxLength: 255,
	})
	columns = append(columns, ColumnInput{
		Name:         "created_at",
		Type:         "timestamp",
		Nullable:     false,
		DefaultValue: "CURRENT_TIMESTAMP",
	})

	table := TableInput{
		Database: environment.GetEnvValue("INTERNAL_SCHEMA_NAME"),
		Name:     "engine_data_triggers",
		Columns:  columns,
	}

	CreateTable(db, table)

}

func CreateEngineRelationsTable(db *sql.DB) {
	columns := []ColumnInput{}
	columns = append(columns, ColumnInput{
		Name:          "id",
		Type:          "bigint",
		Nullable:      false,
		AutoIncrement: true,
	})
	columns = append(columns, ColumnInput{
		Name:      "alias",
		Type:      "varchar",
		Nullable:  false,
		MaxLength: 255,
	})
	columns = append(columns, ColumnInput{
		Name:      "db",
		Type:      "varchar",
		Nullable:  false,
		MaxLength: 255,
	})
	columns = append(columns, ColumnInput{
		Name:      "from_table",
		Type:      "varchar",
		Nullable:  false,
		MaxLength: 255,
	})
	columns = append(columns, ColumnInput{
		Name:      "to_table",
		Type:      "varchar",
		Nullable:  false,
		MaxLength: 255,
	})
	columns = append(columns, ColumnInput{
		Name:      "from_column",
		Type:      "varchar",
		Nullable:  false,
		MaxLength: 255,
	})
	columns = append(columns, ColumnInput{
		Name:      "to_column",
		Type:      "varchar",
		Nullable:  false,
		MaxLength: 255,
	})
	columns = append(columns, ColumnInput{
		Name:      "relation",
		Type:      "varchar",
		Nullable:  false,
		MaxLength: 255,
	})

	indexes := []IndexInput{}

	aliasUniqueColumn := ColumnInput{
		Name:      "alias",
		Type:      "varchar",
		Nullable:  false,
		MaxLength: 255,
	}

	uniqueColumns := []ColumnInput{}

	uniqueColumns = append(uniqueColumns, aliasUniqueColumn)

	uniqueIndex := IndexInput{
		Type:    UNIQUE,
		Columns: uniqueColumns,
	}

	indexes = append(indexes, uniqueIndex)

	table := TableInput{
		Database: environment.GetEnvValue("INTERNAL_SCHEMA_NAME"),
		Name:     "relations",
		Columns:  columns,
		Indexes:  indexes,
	}

	CreateTable(db, table)

	CreateIndexes(db, table)

}

func CreateEngineApiKeysTable(db *sql.DB) {
	columns := []ColumnInput{}
	columns = append(columns, ColumnInput{
		Name:          "id",
		Type:          "bigint",
		Nullable:      false,
		AutoIncrement: true,
	})
	columns = append(columns, ColumnInput{
		Name:      "api_key",
		Type:      "varchar",
		Nullable:  false,
		MaxLength: 255,
	})
	columns = append(columns, ColumnInput{
		Name:         "created_at",
		Type:         "timestamp",
		Nullable:     false,
		DefaultValue: "CURRENT_TIMESTAMP",
	})
	columns = append(columns, ColumnInput{
		Name:         "enabled",
		Type:         "boolean",
		Nullable:     false,
		DefaultValue: false,
	})

	indexes := []IndexInput{}

	apiKeyUniqueColumn := ColumnInput{
		Name:      "api_key",
		Type:      "varchar",
		Nullable:  false,
		MaxLength: 255,
	}

	uniqueColumns := []ColumnInput{}

	uniqueColumns = append(uniqueColumns, apiKeyUniqueColumn)

	uniqueIndex := IndexInput{
		Type:    UNIQUE,
		Columns: uniqueColumns,
	}

	indexes = append(indexes, uniqueIndex)

	table := TableInput{
		Database: environment.GetEnvValue("INTERNAL_SCHEMA_NAME"),
		Name:     "engine_api_keys",
		Columns:  columns,
		Indexes:  indexes,
	}

	CreateTable(db, table)

	CreateIndexes(db, table)

}
