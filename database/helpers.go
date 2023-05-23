package database

import (
	"application/environment"
	"fmt"
	"reflect"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

func Find[T any](s []T, f func(T) bool) *T {
	for _, item := range s {
		if f(item) {
			return &item
		}
	}
	return nil
}

func IsOrderedMap(arg interface{}) (OrderedMap, error) {
	switch args := arg.(type) {
	case OrderedMap:
		return args, nil
	case *OrderedMap:
		return *args, nil
	default:
		return OrderedMap{}, fmt.Errorf("type of interface is not map[string]interface{}")
	}
}

func IsMapToInterface(arg interface{}) (map[string]interface{}, error) {
	switch args := arg.(type) {
	case map[string]interface{}:
		return args, nil
	case OrderedMap:
		return args.values, nil
	case *OrderedMap:
		return args.values, nil
	default:
		return nil, fmt.Errorf("type of interface is not map[string]interface{}")
	}
}

func IsMapToArray(arg interface{}) (map[string][]interface{}, error) {
	switch args := arg.(type) {
	case map[string][]interface{}:
		return args, nil
	default:
		return nil, fmt.Errorf("type of interface is not map[string][]interface{}")
	}
}

func IsJwtMapClaims(arg interface{}) (map[string]interface{}, error) {
	switch args := arg.(type) {
	case jwt.MapClaims:
		return args, nil
	default:
		return nil, fmt.Errorf("type of interface is not map[string]interface{}")
	}
}

func IsArray(arg interface{}) ([]interface{}, error) {

	x, ok := arg.([]interface{})
	if !ok {
		return nil, fmt.Errorf("type of interface is not []interface{}")
	}

	return x, nil
}

func isArrayOfStrings(arg interface{}) ([]string, error) {
	v := reflect.ValueOf(arg)
	if v.Kind() != reflect.Slice {
		return nil, fmt.Errorf("type of interface is not []string")
	}
	items := make([]string, 0)
	for i := 0; i < v.Len(); i++ {
		val, ok := v.Index(i).Interface().(string)
		if !ok {
			return nil, fmt.Errorf("type of interface is not []string")
		}
		items = append(items, val)
	}
	return items, nil

}

func IsEligibleWhereOperation(arg interface{}) bool {
	_, errorX := IsMapToInterface(arg)
	_, errorY := IsArray(arg)
	return errorX != nil || errorY != nil
}

func IsEligibleOrderByOperation(arg interface{}) bool {
	parsedInput, errorX := IsMapToInterface(arg)
	if errorX != nil {
		return false
	}
	fields, ok := parsedInput["_orderBy"]
	if !ok {
		return false
	}
	parsedOrderBy, errorY := IsMapToInterface(fields)
	if errorY != nil {
		return false
	}
	return len(parsedOrderBy) > 0
}

func GetMapValueFromStringKey(arg map[string]interface{}, key string) (interface{}, error) {
	if value, ok := arg[key]; ok {
		return value, nil
	}

	return nil, fmt.Errorf("no such value")
}

func CheckIfFloat64IsInt(value float64) interface{} {
	intVal := int64(value)
	if (value - float64(intVal)) == 0 {
		return int64(value)
	}
	return value
}

func CheckIfFloat32IsInt(value float32) interface{} {
	intVal := int64(value)
	if (value - float32(intVal)) == 0 {
		return int64(value)
	}
	return value
}

func IsEligibleModelRequestBody(arg interface{}) bool {
	switch arg.(type) {
	case bool:
		return true
	case map[string]interface{}:
		return true
	default:
		return false
	}
}

func isEligibleInsertModelRequestBody(arg interface{}) (map[string]interface{}, error) {
	return IsMapToInterface(arg)
}

func GetRelationalKeys(payload map[string]interface{}) []string {
	var relationalKeys []string

	for key := range payload {
		if _, ok := SELECT_BODY_KEYS[key]; !ok {
			relationalKeys = append(relationalKeys, key)
		}
	}

	return relationalKeys
}

func GetModelColumnsWithAlias(model *Model, alias string) string {
	columns := []string{}
	prefix := ""
	if len(alias) > 0 {
		prefix = fmt.Sprintf("%s.", alias)
	}
	for _, column := range model.Columns {
		columns = append(columns, fmt.Sprintf("%s%s", prefix, column.Name))
	}

	return strings.Join(columns, ",")
}

func GetRelationalCoalesceSymbols(model *Model, relationInfo *DatabaseRelationSchema, depth int, parentAlias string) RelationCoalesceBuilder {

	builder := RelationCoalesceBuilder{
		RelationExtractSymbol:  "",
		RelationCoalesceSymbol: "[]",
		RelationAlias:          model.Table,
		RelationWhereJoin:      "",
	}
	if relationInfo == nil {
		return builder
	}
	if relationInfo.RelationType == "OBJECT" {
		builder.RelationExtractSymbol = "->0"
		builder.RelationCoalesceSymbol = "null"

	}
	currentAlias := fmt.Sprintf("_%d_%s", depth, relationInfo.ToTable)
	builder.RelationAlias = relationInfo.Alias
	builder.RelationWhereJoin = fmt.Sprintf(" WHERE %s.%s = %s.%s",
		parentAlias,
		relationInfo.FromColumn,
		currentAlias,
		relationInfo.ToColumn,
	)

	return builder
}

func (e *Engine) DatabaseExists(database string) bool {
	_, ok := e.DatabaseToTableToModelMap[database]

	return ok
}

func (e *Engine) GetModelByKey(database, key string) (*Model, error) {

	tablesMap, ok := e.DatabaseToTableToModelMap[database]
	if !ok {
		return nil, fmt.Errorf("no such model %s for database %s", key, database)
	}
	tableName := ClearAliasForAggregate(key)
	model, ok := tablesMap[tableName]
	if !ok {
		return nil, fmt.Errorf("no such model %s", key)
	}
	return model, nil
}
func IsAggregation(alias string) bool {
	return strings.HasSuffix(alias, "_aggregate")
}

func ClearAliasForAggregate(alias string) string {
	return strings.Split(alias, "_aggregate")[0]
}

func ProcessEntryIsSelect(entry map[string]interface{}) bool {
	_, ok := entry["select"]
	return ok
}

func ProcessEntryIsInsert(entry map[string]interface{}) bool {
	_, ok := entry["insert"]

	return ok
}

func ProcessEntryIsUpdate(entry map[string]interface{}) bool {
	_, ok := entry["update"]
	return ok
}

func ProcessEntryIsDelete(entry map[string]interface{}) bool {
	_, ok := entry["delete"]
	return ok
}

func GetFirstKeyFromMap(args interface{}) (string, map[string]interface{}, error) {
	parsed, err := IsMapToInterface(args)
	if err != nil {
		return "", nil, nil
	}

	for key := range parsed {
		return key, parsed, nil
	}

	return "", nil, fmt.Errorf("empty map")
}

func LogSql(query string) {
	if environment.GetEnvValue("SQL_LOGGER") == "ON" {
		fmt.Println(query)
	}
}
