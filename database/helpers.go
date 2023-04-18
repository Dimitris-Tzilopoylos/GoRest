package database

import (
	"fmt"
	"strings"
)

func Find[T any](s []T, f func(T) bool) *T {
	for _, item := range s {
		if f(item) {
			return &item
		}
	}
	return nil
}

func IsMapToInterface(arg interface{}) (map[string]interface{}, error) {
	switch args := arg.(type) {
	case map[string]interface{}:
		return args, nil
	default:
		return nil, fmt.Errorf("type of interface is not map[string]interface{}")
	}
}

func IsArray(arg interface{}) ([]interface{}, error) {
	switch args := arg.(type) {
	case []interface{}:
		return args, nil
	default:
		return nil, fmt.Errorf("type of interface is not []interface{}")
	}
}

func IsStringArray(arg interface{}) ([]string, error) {

	switch args := arg.(type) {
	case []string:
		return args, nil
	default:
		return nil, fmt.Errorf("value is not a string array")
	}
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

func (e *Engine) GetModelByKey(key string) (*Model, error) {
	model := Find(e.Models, func(model *Model) bool {
		modelName := ClearAliasForAggregate(key)

		return model.Table == modelName
	})

	var err error
	if model == nil {
		err = fmt.Errorf("no such model %s", key)
		return nil, err
	}

	return *model, err

}
