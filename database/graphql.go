package database

import (
	"application/environment"
	"database/sql"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/relay"
	"github.com/graphql-go/graphql/language/ast"
	gqlParser "github.com/graphql-go/graphql/language/parser"
)

var SqlToGqlTypeMap map[string]string = map[string]string{
	"int":               "Int",
	"integer":           "Int",
	"bigint":            "Int",
	"bigserial":         " Int",
	"character varying": "String",
	"character":         "String",
	"varchar":           "String",
	"char":              "String",
	"float":             "Float",
	"double":            "Float",
	"tinyint":           "Boolean",
	"json":              "Object",
	"jsonb":             "Object",
	"default":           "String",
	"boolean":           "Boolean",
	"bool":              "Boolean",
}

var SqlAggValueToGqlTypeMap map[string]string = map[string]string{
	"int":               "Int",
	"integer":           "Int",
	"bigint":            "Int",
	"bigserial":         " Int",
	"character varying": "String",
	"character":         "String",
	"varchar":           "String",
	"char":              "String",
	"float":             "Float",
	"double":            "Float",
}

type GraphQLRequestInput struct {
	Query         string                 `json:"query"`
	OperationName string                 `json:"operationName"`
	Variables     map[string]interface{} `json:"variables"`
}

type EngineGraphQlDatabaseTableConfig struct {
	Database   string
	Table      string
	ActionType string
}

type GraphQLEntity struct {
	Schema                                     string
	ParsedSchema                               *graphql.Schema
	Handler                                    *relay.Handler
	EngineResolverNameToDatabaseTableConfigMap map[string]*EngineGraphQlDatabaseTableConfig
}

func (g *GraphQLEntity) ValidateGraphQlRequestInput(input *GraphQLRequestInput) error {
	formattedQuery := strings.Trim(input.Query, " ")
	if len(formattedQuery) == 0 {
		return fmt.Errorf("please provide at least a query or a mutation")
	}

	return nil
}

func GetGraphqlAggregationTypeByColumn(column Column) (string, error) {

	if cType, ok := SqlAggValueToGqlTypeMap[column.Type]; ok {
		return cType, nil
	}

	return "", fmt.Errorf("not supported column type [%s] for aggregation", column.Type)
}

func GetGraphqlArrayFieldTypeByColumn(column Column) string {
	columnType := column.Type
	suffix := ""
	if !column.Nullable {
		suffix = "!"
	}
	items := strings.Count(columnType, "[]")
	i := 0
	placeholder := "%s"
	for {
		if i >= items {
			return placeholder
		}
		placeholder = fmt.Sprintf(placeholder, fmt.Sprintf("[%s]", suffix))
	}

}

func GetGraphqlQueryFieldTypeByColumn(column Column) (string, error) {
	columnType := column.Type
	isArray := strings.HasSuffix(columnType, "[]")

	formattedStringType := strings.Split(strings.Trim(columnType, " "), "[]")

	if len(formattedStringType) == 0 {
		return "", fmt.Errorf("no type provided for column %s", column.Name)
	}

	colType := formattedStringType[0]

	if len(colType) == 0 {
		return "", fmt.Errorf("no type provided for column %s", column.Name)
	}

	typeFromMap, ok := SqlToGqlTypeMap[colType]

	if !ok {
		typeFromMap = "String"
	}

	suffix := ""
	if !column.Nullable {
		suffix = "!"
	}

	if isArray {
		placeholder := GetGraphqlArrayFieldTypeByColumn(column)
		return fmt.Sprintf(placeholder, typeFromMap+suffix), nil
	}

	return typeFromMap + suffix, nil

}

func GetOrderByEnum() string {
	return "enum order_by_direction_enum {\nDESC\nASC\nASC_NULLS_FIRST\nASC_NULLS_LAST\nDSC_NULLS_FIRST\nDESC_NULLS_LAST\n}"
}

func GetScalarsAndInputs() string {
	return `scalar Object
scalar SingleValue
input limit_input_exp {
  _limit: Int
}
input offset_input_exp {
  _offset: Int
}
input _in {
 _in: [SingleValue!]
} 
input _nin {
 _nin: [SingleValue!]
} 
input _lt {
 _lt: SingleValue
} 
input _lte {
 _lte: SingleValue
} 
input _gt {
 _gt: SingleValue
} 
input _gte {
 _gte: SingleValue
} 
input _is {
 _is: SingleValue
} 
input _is_not {
 _is_not: SingleValue
} 
input _like {
 _like: String
} 
input _ilike {
 _ilike: String
} 
input _eq {
 _eq: SingleValue
} 
input _neq {
 _neq: SingleValue
} 
input _any {
 _any: [SingleValue!]
} 
input _nany {
 _nany: SingleValue
} 
input _all {
 _all: [SingleValue!]
} 
input _contains {
 _contains: Object
} 
input _contained_in {
 _contained_in: Object
} 
input _key_exists {
 _key_exists: String
} 
input _key_exists_any {
 _key_exists_any: [String]
} 
input _key_exists_all {
 _key_exists_all: [String]
} 
input _text_search {
 _text_search: SingleValue
} 
input column_input {
_in: [SingleValue!]
_nin: [SingleValue!]
_lt: SingleValue
_lte: SingleValue
_gt: SingleValue
_gte: SingleValue
_is: SingleValue
_is_not: SingleValue
_like: String
_ilike: String
_eq: SingleValue
_neq: SingleValue
_any: [SingleValue!]
_nany: SingleValue
_all: [SingleValue!]
_contains: Object
_contained_in: Object
_key_exists: String
_key_exists_any: [String]
_key_exists_all: [String]
_text_search: SingleValue
}`
}

func RemoveRequiredSuffixFromGQLType(fields []string) []string {
	newFields := []string{}
	for _, field := range fields {
		newField := strings.TrimRight(field, "!")
		newFields = append(newFields, newField)
	}

	return newFields
}

func BuildModelColumnsEnum(model *Model) (string, error) {
	typeName := fmt.Sprintf("enum %s_%s_enum", model.Database, model.Table)
	fields := []string{}
	for key := range model.ColumnsMap {
		fields = append(fields, key)
	}

	return fmt.Sprintf("%s {\n%s\n}", typeName, strings.Join(fields, "\n")), nil
}

func BuildModelOrderByExp(model *Model) (string, error) {
	typeName := fmt.Sprintf("input %s_%s_order_by_exp", model.Database, model.Table)

	fields := []string{}
	for key := range model.ColumnsMap {
		fields = append(fields, fmt.Sprintf("%s: order_by_direction_enum", key))
	}

	return fmt.Sprintf("%s {\n%s\n}", typeName, strings.Join(fields, "\n")), nil
}

func BuildModelBoolExp(model *Model) (string, error) {
	typeName := fmt.Sprintf("input %s_%s_bool_exp", model.Database, model.Table)

	fields := []string{}
	for key := range model.ColumnsMap {
		fields = append(fields, fmt.Sprintf("%s: column_input", key))
	}

	for key, value := range model.Relations {
		fields = append(fields, fmt.Sprintf("%s: %s_%s_bool_exp", key, value.Database, value.Table))
	}

	fields = append(fields, fmt.Sprintf("_and: [%s_%s_bool_exp!]", model.Database, model.Table))
	fields = append(fields, fmt.Sprintf("_or: [%s_%s_bool_exp!]", model.Database, model.Table))

	return fmt.Sprintf("%s {\n%s\n}", typeName, strings.Join(fields, "\n")), nil
}

func BuildQueryTypeFields(model *Model) ([]string, error) {
	fields := make([]string, 0)

	for _, col := range model.Columns {
		fieldType, err := GetGraphqlQueryFieldTypeByColumn(col)
		if err != nil {
			return fields, err
		}
		fields = append(fields, fmt.Sprintf("%s: %s", col.Name, fieldType))
	}

	return fields, nil

}

func BuildSelectAggregateTypeArgs(model *Model) string {
	_where := fmt.Sprintf("_where: %s_%s_bool_exp", model.Database, model.Table)
	_groupBy := fmt.Sprintf("_groupBy: [%s_%s_enum]", model.Database, model.Table)
	_orderBy := fmt.Sprintf("_orderBy: %s_%s_order_by_exp", model.Database, model.Table)
	_distinct := fmt.Sprintf("_distinct: [%s_%s_enum]", model.Database, model.Table)

	arr := []string{
		_where,
		_orderBy,
		_groupBy,
		_distinct,
	}
	return fmt.Sprintf("(%s)", strings.Join(arr, ", "))
}

func BuildSelectTypeArgs(model *Model) string {
	_where := fmt.Sprintf("_where: %s_%s_bool_exp", model.Database, model.Table)
	_groupBy := fmt.Sprintf("_groupBy: [%s_%s_enum]", model.Database, model.Table)
	_orderBy := fmt.Sprintf("_orderBy: %s_%s_order_by_exp", model.Database, model.Table)
	_distinct := fmt.Sprintf("_distinct: [%s_%s_enum]", model.Database, model.Table)
	_limit := "_limit: Int"
	_offset := "_offset: Int"
	arr := []string{
		_where,
		_groupBy,
		_orderBy,
		_distinct,
		_limit,
		_offset,
	}
	return fmt.Sprintf("(%s)", strings.Join(arr, ", "))
}

func BuildModelRelationalFields(model *Model) ([]string, error) {
	fields := []string{}
	for key, value := range model.Relations {

		field := fmt.Sprintf(`%s%s:%s_%s`, key, BuildSelectTypeArgs(value), value.Database, value.Table)
		fields = append(fields, field)
	}

	return fields, nil
}

func BuildModelRelationalAggregateColumns(model *Model) ([]string, error) {
	fields := []string{}
	for key, value := range model.Relations {

		field := fmt.Sprintf(`%s_aggregate%s: %s_%s_aggregate`, key, BuildSelectAggregateTypeArgs(value), value.Database, value.Table)
		fields = append(fields, field)
	}

	return fields, nil
}

func BuildModelAggregateType(model *Model) ([]string, error) {
	fields := []string{}
	for _, column := range model.Columns {
		colType, err := GetGraphqlAggregationTypeByColumn(column)
		if err != nil {
			fields = append(fields, fmt.Sprintf("%s: %s", column.Name, "Float"))
			continue
		}
		fields = append(fields, fmt.Sprintf("%s: %s", column.Name, colType))
	}

	if len(fields) == 0 {
		return nil, fmt.Errorf("no fields for aggregation")
	}

	return fields, nil
}

func BuildModelAggregateTypes(model *Model) ([]string, error) {
	typeName := fmt.Sprintf("type %s_%s_aggregate", model.Database, model.Table)
	types := []string{"min", "max", "sum", "avg"}

	fields := []string{}
	for _, aggType := range types {
		aggFields, err := BuildModelAggregateType(model)
		if err != nil {
			continue
		}

		fields = append(fields, fmt.Sprintf("%s_%s {\n%s\n}", typeName, aggType, strings.Join(aggFields, "\n")))
	}

	return fields, nil
}

func BuildModelUpdateInput(model *Model) (string, error) {
	typeName := fmt.Sprintf("input %s_%s_update_input", model.Database, model.Table)
	fields, _ := BuildQueryTypeFields(model)
	formattedFields := RemoveRequiredSuffixFromGQLType(fields)
	return fmt.Sprintf("%s {\n%s\n}", typeName, strings.Join(formattedFields, "\n")), nil
}

func BuildModelInsertInput(model *Model) (string, error) {
	typeName := fmt.Sprintf("input %s_%s_insert_input", model.Database, model.Table)
	fields := []string{}
	fields = append(fields, fmt.Sprintf("objects: [%s_%s_insert_input_objects!]!", model.Database, model.Table))
	fields = append(fields, fmt.Sprintf("onConflict: %s_%s_insert_input_conflict", model.Database, model.Table))
	return fmt.Sprintf("%s {\n%s\n}", typeName, strings.Join(fields, "\n")), nil
}

func BuildModelInsertInputObjects(model *Model) (string, error) {
	typeName := fmt.Sprintf("input %s_%s_insert_input_objects", model.Database, model.Table)
	fields, _ := BuildQueryTypeFields(model)
	formattedFields := RemoveRequiredSuffixFromGQLType(fields)
	for key, value := range model.Relations {
		formattedFields = append(formattedFields, fmt.Sprintf("%s: %s_%s_insert_input", key, value.Database, value.Table))
	}
	return fmt.Sprintf("%s {\n%s\n}", typeName, strings.Join(formattedFields, "\n")), nil
}

func BuildModelInsertInputOnConflict(model *Model) (string, error) {
	typeName := fmt.Sprintf("input %s_%s_insert_input_conflict", model.Database, model.Table)

	fields := []string{
		"ignore: Boolean",
		fmt.Sprintf("update: [%s_%s_enum!]", model.Database, model.Table),
		"constraint: String",
	}

	return fmt.Sprintf("%s {\n%s\n}", typeName, strings.Join(fields, "\n")), nil
}

func BuildQueryType(model *Model) (string, error) {
	typeName := fmt.Sprintf("type %s_%s", model.Database, model.Table)

	modelColumnFields, err := BuildQueryTypeFields(model)

	if err != nil {
		return typeName, err
	}

	relationalColumns, _ := BuildModelRelationalFields(model)
	modelColumnFields = append(modelColumnFields, relationalColumns...)

	relationalAggregateColumns, _ := BuildModelRelationalAggregateColumns(model)
	modelColumnFields = append(modelColumnFields, relationalAggregateColumns...)

	modelColumnFieldsString := strings.Join(modelColumnFields, "\n")

	return fmt.Sprintf("%s {\n%s\n}", typeName, modelColumnFieldsString), nil

}

func BuildQueryAggregateType(model *Model) (string, error) {
	typeName := fmt.Sprintf("type %s_%s_aggregate", model.Database, model.Table)
	arr := []string{
		fmt.Sprintf("min: %s_%s_aggregate_min", model.Database, model.Table),
		fmt.Sprintf("max: %s_%s_aggregate_max", model.Database, model.Table),
		fmt.Sprintf("sum: %s_%s_aggregate_sum", model.Database, model.Table),
		fmt.Sprintf("avg: %s_%s_aggregate_avg", model.Database, model.Table),
		"count: Int",
	}
	return fmt.Sprintf("%s {\n%s\n}", typeName, strings.Join(arr, "\n")), nil
}

func WriteGraphQLSchemaToFile(schema string) {
	shouldWrite := environment.GetEnvValue("WRITE_GRAPHQL_SCHEMA_FILE") == "ON"
	if !shouldWrite {
		return
	}
	schemaFileName := environment.GetEnvValueToStringWithDefault("GRAPHQL_SCHEMA_FILE_NAME", "engine_graphql_schema.gql")
	err := ioutil.WriteFile(schemaFileName, []byte(schema), 0644)
	if err != nil {
		fmt.Println("Error writing schema file:", err)
		return
	}

	fmt.Println("GraphQL Schema was successfully written to file")
}

func (e *Engine) BuildQueryTypes() ([]string, error) {
	queryTypes := make([]string, 0)
	for _, model := range e.Models {
		if model.Database == e.InternalSchemaName {
			continue
		}
		queryType, err := BuildQueryType(model)
		if err != nil {
			return queryTypes, err
		}

		queryTypes = append(queryTypes, queryType)
	}
	return queryTypes, nil
}

func (e *Engine) BuildQueryAggregateTypes() ([]string, error) {
	queryTypes := make([]string, 0)
	for _, model := range e.Models {
		if model.Database == e.InternalSchemaName {
			continue
		}
		queryType, err := BuildQueryAggregateType(model)
		if err != nil {
			return queryTypes, err
		}

		queryTypes = append(queryTypes, queryType)
		subAggregateTypes, err := BuildModelAggregateTypes(model)
		if err != nil {
			return queryTypes, err
		}
		queryTypes = append(queryTypes, subAggregateTypes...)
	}
	return queryTypes, nil
}

func (e *Engine) BuildEnumTypes() ([]string, error) {
	queryTypes := make([]string, 0)
	for _, model := range e.Models {
		if model.Database == e.InternalSchemaName {
			continue
		}
		queryType, err := BuildModelColumnsEnum(model)
		if err != nil {
			return queryTypes, err
		}

		queryTypes = append(queryTypes, queryType)
	}
	return queryTypes, nil
}

func (e *Engine) BuildSelectInputTypes() ([]string, error) {
	queryTypes := make([]string, 0)
	for _, model := range e.Models {
		if model.Database == e.InternalSchemaName {
			continue
		}
		queryType, err := BuildModelBoolExp(model)
		if err != nil {
			return queryTypes, err
		}
		queryTypes = append(queryTypes, queryType)

		queryType, err = BuildModelOrderByExp(model)
		if err != nil {
			return queryTypes, err
		}

		queryTypes = append(queryTypes, queryType)
	}
	return queryTypes, nil
}

func (e *Engine) BuildUpdateInputTypes() ([]string, error) {
	queryTypes := make([]string, 0)
	for _, model := range e.Models {
		if model.Database == e.InternalSchemaName {
			continue
		}
		queryType, err := BuildModelUpdateInput(model)
		if err != nil {
			return queryTypes, err
		}

		queryTypes = append(queryTypes, queryType)
	}
	return queryTypes, nil
}

func (e *Engine) BuildInsertInputTypes() ([]string, error) {
	queryTypes := make([]string, 0)
	for _, model := range e.Models {
		if model.Database == e.InternalSchemaName {
			continue
		}
		queryType, err := BuildModelInsertInput(model)
		if err != nil {
			return queryTypes, err
		}
		queryTypes = append(queryTypes, queryType)

		queryType, err = BuildModelInsertInputObjects(model)
		if err != nil {
			return queryTypes, err
		}
		queryTypes = append(queryTypes, queryType)

		queryType, err = BuildModelInsertInputOnConflict(model)
		if err != nil {
			return queryTypes, err
		}
		queryTypes = append(queryTypes, queryType)
	}
	return queryTypes, nil
}

func (e *Engine) BuildRootQueryType() ([]string, error) {
	typeName := "type Query"
	fields := []string{}
	for _, model := range e.Models {
		if model.Database == e.InternalSchemaName {
			continue
		}

		fields = append(fields, fmt.Sprintf("%s_%s%s: [%s_%s!]", model.Database, model.Table, BuildSelectTypeArgs(model), model.Database, model.Table))
		fields = append(fields, fmt.Sprintf("%s_%s_aggregate%s: %s_%s_aggregate", model.Database, model.Table, BuildSelectAggregateTypeArgs(model), model.Database, model.Table))

	}

	str := fmt.Sprintf("%s{\n%s\n}", typeName, strings.Join(fields, "\n"))

	return []string{str}, nil
}

func (e *Engine) BuildRootMutationType() ([]string, error) {
	typeName := "type Mutation"
	fields := []string{}
	for _, model := range e.Models {
		if model.Database == e.InternalSchemaName {
			continue
		}

		fields = append(fields, fmt.Sprintf("%s_%s_insert(args:%s_%s_insert_input!): Object", model.Database, model.Table, model.Database, model.Table))
		fields = append(fields, fmt.Sprintf("%s_%s_update(set:%s_%s_update_input!,_where:%s_%s_bool_exp): Object", model.Database, model.Table, model.Database, model.Table, model.Database, model.Table))
		fields = append(fields, fmt.Sprintf("%s_%s_delete(_where:%s_%s_bool_exp): Object", model.Database, model.Table, model.Database, model.Table))
	}

	str := fmt.Sprintf("%s{\n%s\n}", typeName, strings.Join(fields, "\n"))

	return []string{str}, nil
}

func (e *Engine) BuildEngineGraphqlConfig() map[string]*EngineGraphQlDatabaseTableConfig {
	config := make(map[string]*EngineGraphQlDatabaseTableConfig)
	for _, model := range e.Models {
		resolverBaseName := fmt.Sprintf("%s_%s", model.Database, model.Table)
		config[resolverBaseName] = &EngineGraphQlDatabaseTableConfig{
			Database:   model.Database,
			Table:      model.Table,
			ActionType: "SELECT",
		}
		config[fmt.Sprintf("%s_aggregate", resolverBaseName)] = &EngineGraphQlDatabaseTableConfig{
			Database:   model.Database,
			Table:      model.Table,
			ActionType: "SELECT",
		}
		config[fmt.Sprintf("%s_insert", resolverBaseName)] = &EngineGraphQlDatabaseTableConfig{
			Database:   model.Database,
			Table:      model.Table,
			ActionType: "INSERT",
		}
		config[fmt.Sprintf("%s_update", resolverBaseName)] = &EngineGraphQlDatabaseTableConfig{
			Database:   model.Database,
			Table:      model.Table,
			ActionType: "UPDATE",
		}
		config[fmt.Sprintf("%s_delete", resolverBaseName)] = &EngineGraphQlDatabaseTableConfig{
			Database:   model.Database,
			Table:      model.Table,
			ActionType: "DELETE",
		}
	}

	return config
}

func (e *Engine) LoadGraphql() {

	orderBy := []string{GetOrderByEnum()}
	scalarsAndDefaultInputs := []string{GetScalarsAndInputs()}
	queryTypes, _ := e.BuildQueryTypes()
	queryAggregateTypes, _ := e.BuildQueryAggregateTypes()
	enumTypes, _ := e.BuildEnumTypes()
	selectInputTypes, _ := e.BuildSelectInputTypes()
	updateInputTypes, _ := e.BuildUpdateInputTypes()
	insertInputTypes, _ := e.BuildInsertInputTypes()
	rootQuery, _ := e.BuildRootQueryType()
	rootMutation, _ := e.BuildRootMutationType()

	parts := make([]string, 0)
	parts = append(parts, scalarsAndDefaultInputs...)
	parts = append(parts, orderBy...)
	parts = append(parts, queryTypes...)
	parts = append(parts, queryAggregateTypes...)
	parts = append(parts, enumTypes...)
	parts = append(parts, selectInputTypes...)
	parts = append(parts, insertInputTypes...)
	parts = append(parts, updateInputTypes...)
	parts = append(parts, rootQuery...)
	parts = append(parts, rootMutation...)

	schemaStr := strings.Join(parts, "\n")
	WriteGraphQLSchemaToFile(schemaStr)
	config := e.BuildEngineGraphqlConfig()
	e.GraphQL = &GraphQLEntity{
		Schema: schemaStr,
		EngineResolverNameToDatabaseTableConfigMap: config,
	}
	schema := graphql.MustParseSchema(schemaStr, nil)
	e.GraphQL.ParsedSchema = schema
	e.GraphQL.Handler = &relay.Handler{Schema: schema}

}

func (g *GraphQLEntity) GraphqlParser(input string, variables map[string]any) (any, error) {
	if len(strings.Trim(input, " ")) == 0 {
		return nil, fmt.Errorf("no query provided")
	}
	ast, err := gqlParser.Parse(gqlParser.ParseParams{
		Source: input,
	})

	if err != nil {
		return nil, err
	}

	if variables == nil {
		variables = make(map[string]any)
	}
	return astToMap(ast, variables, false), err
}

func (e *Engine) GraphqlQueryResolve(inputData GraphQLRequestInput, role string, db *sql.DB) ([]byte, error) {
	selectBody, err := e.GraphQL.GraphqlParser(inputData.Query, inputData.Variables)
	if err != nil {
		return []byte{}, nil
	}

	parsedSelectBody, err := IsOrderedMap(selectBody)

	if err != nil {

		return nil, err
	}

	configByDatabase := make(map[string]map[string]any)

	for key, value := range parsedSelectBody.GetMap() {
		config, ok := e.GraphQL.EngineResolverNameToDatabaseTableConfigMap[key]
		if !ok {
			return nil, fmt.Errorf("no such relation")
		}
		if config.ActionType != "SELECT" {
			continue
		}
		if _, ok := configByDatabase[config.Database]; !ok {
			configByDatabase[config.Database] = make(map[string]any)
		}
		isAggregate := strings.HasSuffix(key, "_aggregate")
		if isAggregate {
			configByDatabase[config.Database][fmt.Sprintf("%s_aggregate", config.Table)] = value
		} else {
			configByDatabase[config.Database][config.Table] = value
		}
	}

	results := []byte{}
	for dbName, payload := range configByDatabase {
		result, err := e.SelectExec(role, db, dbName, payload)
		if err != nil {
			return nil, err
		}
		results = append(results, result...)
	}

	return results, nil
}

func (e *Engine) GraphqlMutationResolve(inputData GraphQLRequestInput, role string, db *sql.DB) (any, error) {
	actionBody, err := e.GraphQL.GraphqlParser(inputData.Query, inputData.Variables)
	if err != nil {
		return map[string]any{}, nil
	}
	parsedActionBody, err := IsOrderedMap(actionBody)
	if err != nil {
		return nil, err
	}

	configByDatabase := make(map[string][]any)
	var iterErr error
	parsedActionBody.Iter(func(key string, value any) {
		config, ok := e.GraphQL.EngineResolverNameToDatabaseTableConfigMap[key]
		if !ok {
			iterErr = fmt.Errorf("no such relation")
		}
		if config.ActionType == "SELECT" {
			return
		}
		if _, ok := configByDatabase[config.Database]; !ok {
			configByDatabase[config.Database] = make([]any, 0)
		}
		switch config.ActionType {
		case "INSERT":
			payload := make(map[string]any)
			parsedValue, err := IsMapToInterface(value)
			if err != nil {
				iterErr = err
				return
			}
			insertInput, ok := parsedValue["args"]
			if !ok {
				iterErr = fmt.Errorf("no insert input provided")
				return
			}
			payload[config.Table] = insertInput
			configByDatabase[config.Database] = append(configByDatabase[config.Database], map[string]any{
				"insert": payload,
			})
		case "UPDATE":
			payload := make(map[string]any)
			payload[config.Table] = value
			configByDatabase[config.Database] = append(configByDatabase[config.Database], map[string]any{
				"update": payload,
			})
		case "DELETE":
			payload := make(map[string]any)
			payload[config.Table] = value
			configByDatabase[config.Database] = append(configByDatabase[config.Database], map[string]any{
				"delete": payload,
			})
		default:
			iterErr = fmt.Errorf("unknown database operation %s", config.ActionType)
			return
		}
	})
	if iterErr != nil {
		return nil, iterErr
	}

	results := make(map[string]any)

	for dbName, value := range configByDatabase {
		body := map[string]any{
			"transactions": value,
		}
		result, err := e.Process(role, db, dbName, body)
		if err != nil {
			return nil, err
		}
		results[dbName] = result
	}

	return results, nil
}

func (e *Engine) GraphiQL() func(http.ResponseWriter) {

	return func(res http.ResponseWriter) {
		res.Write([]byte(GetGraphiqlTemplate()))
	}
}

func (e *Engine) GetIntrospectionQueryPayload() ([]byte, error) {

	return e.GraphQL.Handler.Schema.ToJSON()
}

func (e *Engine) GetIntrospectionQueryResponse() ([]byte, error) {
	payload, err := e.GetIntrospectionQueryPayload()
	if err != nil {
		return nil, err
	}
	resp := []byte{}
	resp = append(resp, []byte("{\"data\":")...)
	resp = append(resp, payload...)
	resp = append(resp, []byte("}")...)
	return resp, nil
}

func astToMap(node ast.Node, variables map[string]interface{}, isAggregate bool) interface{} {
	switch node := node.(type) {
	case *ast.Document:
		return astToMap(node.Definitions[0], variables, isAggregate)
	case *ast.OperationDefinition:
		return astToMap(node.SelectionSet, variables, isAggregate)
	case *ast.SelectionSet:
		result := NewOrderedMap()
		for _, selection := range node.Selections {
			switch selection := selection.(type) {
			case *ast.Field:
				result.Set(selection.Name.Value, astToMap(selection, variables, isAggregate))
			case *ast.FragmentSpread:

				// Handle fragment spread if needed
				// Example: result["fragmentSpread"] = handleFragmentSpread(selection)
			default:

				// Handle other types of selections if needed
			}
		}
		return result
	case *ast.Field:
		result := make(map[string]interface{})
		isAggregation := strings.HasSuffix(node.Name.Value, "_aggregate") || isAggregate

		if len(node.Arguments) > 0 {
			arguments := make(map[string]interface{})
			for _, arg := range node.Arguments {
				argName := arg.Name.Value
				argValue := astToMap(arg.Value, variables, isAggregation)
				arguments[argName] = argValue
			}
			for key, value := range arguments {
				result[key] = value
			}
		}
		if node.SelectionSet != nil {
			selectionSet := astToMap(node.SelectionSet, variables, isAggregation)
			parsedSelectionSet, err := IsMapToInterface(selectionSet)
			if err != nil {
				return result
			}
			selectMap := make(map[string]any)
			for key, value := range parsedSelectionSet {
				parsedValue, err := IsMapToInterface(value)
				if err != nil {
					if isAggregation {
						result[fmt.Sprintf("_%s", key)] = value
					} else {
						selectMap[key] = value
					}
				} else {
					if len(parsedValue) == 0 {

						if isAggregation {
							if isAggregate {
								result[key] = true
							} else {
								result[fmt.Sprintf("_%s", key)] = true
							}
						} else {
							selectMap[key] = true
						}
					} else {
						if isAggregation {
							keys := []any{}
							for key := range parsedValue {
								keys = append(keys, key)
							}
							result[fmt.Sprintf("_%s", key)] = keys
						} else {
							result[key] = parsedValue
						}
					}
				}
			}
			if isAggregation {
				for key, value := range selectMap {
					result[key] = value
				}
			} else {
				if len(selectMap) > 0 {
					result["_select"] = selectMap
				}
			}
		}
		return result
	case *ast.Variable:
		if value, ok := variables[node.Name.Value]; ok {
			return value
		}
		return nil
	case *ast.IntValue:
		return node.Value
	case *ast.FloatValue:
		return node.Value
	case *ast.StringValue:
		return node.Value
	case *ast.BooleanValue:
		return node.Value
	case *ast.EnumValue:
		return node.Value
	case *ast.ObjectValue:
		obj := make(map[string]interface{})
		for _, field := range node.Fields {
			obj[field.Name.Value] = astToMap(field.Value, variables, isAggregate)
		}
		return obj
	case *ast.ListValue:
		list := make([]interface{}, len(node.Values))
		for i, value := range node.Values {
			list[i] = astToMap(value, variables, isAggregate)
		}
		return list
	default:
		return nil
	}
}
