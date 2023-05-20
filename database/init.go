package database

import (
	"application/environment"
	"database/sql"
	"log"
	"sync"
)

var mutex sync.Mutex

type Engine struct {
	Databases                 []string                     `json:"databases"`
	Database                  string                       `json:"database"`
	Models                    []*Model                     `json:"models"`
	DatabaseToTableToModelMap map[string]map[string]*Model `json:"schema"`
	Relations                 []DatabaseRelationSchema     `json:"relations"`
	GlobalAuthEntities        []GlobalAuthEntity
	GraphQL                   *GraphQLEntity
	EventEmitter              *EventEmitter
	InternalSchemaName        string
	Version                   string
	Webhooks                  map[string]map[string]map[string]map[string][]Webhook
	DataTriggers              map[string]map[string]DataTrigger
	RestHandlers              []CustomRestHandlerInput
	RestHandlersMap           map[string]map[string]CustomRestHandlerInput
}

func Init(db *sql.DB) *Engine {
	InitializeEngineDatabase(db)
	relations, _ := GetEngineRelations(db)
	databases, _ := GetDatabases(db)
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
		Databases:                 databases,
		Models:                    models,
		DatabaseToTableToModelMap: schema,
		GlobalAuthEntities:        make([]GlobalAuthEntity, 0),
		InternalSchemaName:        environment.GetEnvValue("INTERNAL_SCHEMA_NAME"),
		Version:                   environment.GetEnvValue("VERSION"),
		EventEmitter:              NewEventEmitter(),
		Relations:                 relations,
	}

	engine.LoadGlobalAuth(db)
	engine.LoadWebhooks(db)
	engine.LoadDataTriggers(db)
	engine.LoadRestHandlers(db)
	engine.LoadGraphql()
	return engine
}

func (engine *Engine) Reload(db *sql.DB) {
	mutex.Lock()
	defer mutex.Unlock()
	databases, _ := GetDatabases(db)
	relations, _ := GetEngineRelations(db)

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

	engine.Databases = databases
	engine.Models = models
	engine.DatabaseToTableToModelMap = schema
	engine.Relations = relations
	engine.LoadGlobalAuth(db)
	engine.LoadWebhooks(db)
	engine.LoadDataTriggers(db)
	engine.LoadRestHandlers(db)
	engine.LoadGraphql()
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
	CreateEngineCustomEndopointsTable(db)
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

	primaryIndexColumn := ColumnInput{
		Name:          "id",
		Type:          "bigint",
		Nullable:      false,
		AutoIncrement: true,
	}

	primaryIndex := IndexInput{
		Columns: []ColumnInput{
			primaryIndexColumn,
		},
		Type: PRIMARY,
	}

	indexes := []IndexInput{}

	indexes = append(indexes, primaryIndex)

	table := TableInput{
		Database: environment.GetEnvValue("INTERNAL_SCHEMA_NAME"),
		Name:     "engine_auth_provider",
		Columns:  columns,
		Indexes:  indexes,
	}

	CreateTable(db, table)
	CreateIndexes(db, table)

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

	primaryIndexColumn := ColumnInput{
		Name:          "id",
		Type:          "bigint",
		Nullable:      false,
		AutoIncrement: true,
	}

	primaryIndex := IndexInput{
		Columns: []ColumnInput{
			primaryIndexColumn,
		},
		Type: PRIMARY,
	}

	indexes := []IndexInput{}

	indexes = append(indexes, primaryIndex)

	table := TableInput{
		Database: environment.GetEnvValue("INTERNAL_SCHEMA_NAME"),
		Name:     "engine_data_triggers",
		Columns:  columns,
		Indexes:  indexes,
	}

	CreateTable(db, table)
	CreateIndexes(db, table)

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
	primaryIndexColumn := ColumnInput{
		Name:          "id",
		Type:          "bigint",
		Nullable:      false,
		AutoIncrement: true,
	}

	primaryIndex := IndexInput{
		Columns: []ColumnInput{
			primaryIndexColumn,
		},
		Type: PRIMARY,
	}

	indexes = append(indexes, primaryIndex, uniqueIndex)

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

	primaryIndexColumn := ColumnInput{
		Name:          "id",
		Type:          "bigint",
		Nullable:      false,
		AutoIncrement: true,
	}

	primaryIndex := IndexInput{
		Columns: []ColumnInput{
			primaryIndexColumn,
		},
		Type: PRIMARY,
	}

	indexes = append(indexes, uniqueIndex, primaryIndex)

	table := TableInput{
		Database: environment.GetEnvValue("INTERNAL_SCHEMA_NAME"),
		Name:     "engine_api_keys",
		Columns:  columns,
		Indexes:  indexes,
	}

	CreateTable(db, table)

	CreateIndexes(db, table)

}

func CreateEngineCustomEndopointsTable(db *sql.DB) {
	columns := []ColumnInput{}
	columns = append(columns, ColumnInput{
		Name:          "id",
		Type:          "bigint",
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
		Name:      "method",
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
		Name:     "query",
		Type:     "text",
		Nullable: false,
	})
	columns = append(columns, ColumnInput{
		Name:         "auth",
		Type:         "boolean",
		Nullable:     false,
		DefaultValue: false,
	})
	columns = append(columns, ColumnInput{
		Name:         "enabled",
		Type:         "boolean",
		Nullable:     false,
		DefaultValue: false,
	})
	columns = append(columns, ColumnInput{
		Name:         "created_at",
		Type:         "timestamp",
		Nullable:     false,
		DefaultValue: "CURRENT_TIMESTAMP",
	})

	indexes := []IndexInput{}

	methodUniqueColumn := ColumnInput{
		Name:      "method",
		Type:      "varchar",
		Nullable:  false,
		MaxLength: 255,
	}

	endpointUniqueColumn := ColumnInput{
		Name:      "endpoint",
		Type:      "varchar",
		Nullable:  false,
		MaxLength: 255,
	}

	uniqueColumns := []ColumnInput{}

	uniqueColumns = append(uniqueColumns, endpointUniqueColumn, methodUniqueColumn)

	uniqueIndex := IndexInput{
		Type:    UNIQUE,
		Columns: uniqueColumns,
	}

	primaryIndexColumn := ColumnInput{
		Name:          "id",
		Type:          "bigint",
		Nullable:      false,
		AutoIncrement: true,
	}

	primaryIndex := IndexInput{
		Columns: []ColumnInput{
			primaryIndexColumn,
		},
		Type: PRIMARY,
	}

	indexes = append(indexes, uniqueIndex, primaryIndex)

	table := TableInput{
		Database: environment.GetEnvValue("INTERNAL_SCHEMA_NAME"),
		Name:     "engine_rest_actions",
		Columns:  columns,
		Indexes:  indexes,
	}

	CreateTable(db, table)

	CreateIndexes(db, table)
}
