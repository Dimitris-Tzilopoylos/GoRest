package database

import (
	"application/environment"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/graphql-go/graphql"
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

type GQLQuery struct{}

type Object map[string]interface{}

func (q *GQLQuery) Resolve(p graphql.ResolveParams) (interface{}, error) {
	// logic to handle any query
	return nil, nil
}

func (Object) ImplementsGraphQLType(name string) bool {
	return name == "JSONObject"
}

func (j *Object) UnmarshalGraphQL(input interface{}) error {
	switch input := input.(type) {
	case string:
		return json.Unmarshal([]byte(input), &j)
	default:
		return fmt.Errorf("invalid JSONObject value")
	}
}

func (j Object) MarshalJSON() ([]byte, error) {
	return json.Marshal(j)
}

type GraphQLEntity struct {
	Schema string
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

// func BuildRelationalAggregationQueryFields(model *Model) ([]string, error) {

// }

// func BuildRelationalQueryFields(model *Model) ([]string, error) {

// }

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

func BuildQueryType(model *Model) (string, error) {
	typeName := fmt.Sprintf("type %s_%s", model.Database, model.Table)

	modelColumnFields, err := BuildQueryTypeFields(model)

	if err != nil {
		return typeName, err
	}

	modelColumnFieldsString := strings.Join(modelColumnFields, "\n")

	return fmt.Sprintf("%s {\n%s\n}", typeName, modelColumnFieldsString), nil

}

func WriteGraphQLSchemaToFile(schema string) {
	shouldWrite := environment.GetEnvValue("WRITE_GRAPHQL_SCHEMA_FILE") == "ON"
	if !shouldWrite {
		return
	}

	err := ioutil.WriteFile("engine_graphql_schema.gql", []byte(schema), 0644)
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

func (e *Engine) BuildGraphQLSchema() {

	orderBy := []string{GetOrderByEnum()}
	scalarsAndDefaultInputs := []string{GetScalarsAndInputs()}
	queryTypes, err := e.BuildQueryTypes()

	if err != nil {
		return
	}

	parts := make([]string, 0)
	parts = append(parts, scalarsAndDefaultInputs...)
	parts = append(parts, orderBy...)
	parts = append(parts, queryTypes...)

	if err != nil {
		fmt.Println(err)
	}

}
