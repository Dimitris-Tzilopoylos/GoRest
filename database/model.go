package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lib/pq"
	_ "github.com/lib/pq"
)

type JSONB json.RawMessage
type JSON json.RawMessage
type JSONColumn interface{}
type Column struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	MaxLength    int64  `json:"max_length"`
	Nullable     bool   `json:"nullable"`
	DefaultValue string `json:"default_value"`
}

type RelationWhereAggregate struct {
	alias    string
	modelKey string
	body     interface{}
	binder   string
}

type ModelRelation *Model
type RelationMap map[string]ModelRelation
type RelationInfoMap map[string]DatabaseRelationSchema
type ColumnsMap map[string]string
type RLSMap map[string]ColumnsMap

type InsertionResult struct {
	LastInsertId any
	RowsAffected any
	Returning    any
}
type DatabaseRelationSchema struct {
	Id           int64  `json:"id"`
	Alias        string `json:"alias"`
	Database     string `json:"database"`
	FromTable    string `json:"from_table"`
	FromColumn   string `json:"from_column"`
	ToTable      string `json:"to_table"`
	ToColumn     string `json:"to_column"`
	RelationType string `json:"relation_type"`
}

type IndexType string

type Index struct {
	Name            string    `json:"name"`
	Type            IndexType `json:"type"`
	Table           string    `json:"table"`
	Column          string    `json:"column"`
	ReferenceTable  string    `json:"reference_table"`
	ReferenceColumn string    `json:"reference_column"`
}

type Model struct {
	Database         string   `json:"database"`
	Table            string   `json:"table"`
	Columns          []Column `json:"columns"`
	Relations        RelationMap
	RelationsInfoMap RelationInfoMap
	Indexes          []Index
	ColumnsMap       ColumnsMap
	RLS              RLSMap
}

type RelationCoalesceBuilder struct {
	RelationExtractSymbol  string
	RelationCoalesceSymbol string
	RelationAlias          string
	RelationWhereJoin      string
}

func NewModel(database string, table string) *Model {
	return &Model{
		Database:         database,
		Table:            table,
		Columns:          make([]Column, 0),
		Relations:        make(RelationMap),
		Indexes:          make([]Index, 0),
		RelationsInfoMap: make(RelationInfoMap),
		ColumnsMap:       make(ColumnsMap),
		RLS:              make(RLSMap),
	}
}

func (model *Model) GetModelRelationInfo(alias string) (*DatabaseRelationSchema, error) {
	x := ClearAliasForAggregate(alias)
	if info, ok := model.RelationsInfoMap[x]; ok {
		return &info, nil
	}

	return nil, fmt.Errorf("no such relation")
}

func (model *Model) GetModelRelation(alias string) (*Model, error) {
	x := ClearAliasForAggregate(alias)
	if info, ok := model.Relations[x]; ok {
		return info, nil
	}
	return nil, fmt.Errorf("no such relation")
}

func (model *Model) Select(role string, body interface{}, depth int, idx *int, relationInfo *DatabaseRelationSchema, parentAlias string) (string, []interface{}) {
	query := ``
	args := make([]interface{}, 0)
	if !IsEligibleModelRequestBody(body) {
		return query, args
	}
	builder := GetRelationalCoalesceSymbols(model, relationInfo, depth, parentAlias)
	query = fmt.Sprintf(`SELECT coalesce(json_agg(_%d_%s)%s,'%s') as %s FROM (`,
		depth,
		model.Table,
		builder.RelationExtractSymbol,
		builder.RelationCoalesceSymbol,
		builder.RelationAlias)
	makeQuery := func(model *Model, bodyEntities interface{}, aliasPart string) {
		parsedBody, err := IsMapToInterface(bodyEntities)
		currentAlias := fmt.Sprintf("_%d_%s", depth, aliasPart)
		modelColumnsString := model.GetModelColumnsWithAlias(role, body, currentAlias)

		if _where, ok := parsedBody["_where"]; ok {
			initialQuery := " WHERE "
			binder := ""
			if len(builder.RelationWhereJoin) > 0 {
				initialQuery = builder.RelationWhereJoin
				binder = "AND"
			}
			whereQuery, whereArgs := model.BuildWhereClause(_where, currentAlias, idx, initialQuery, binder)
			args = append(args, whereArgs...)
			builder.RelationWhereJoin = whereQuery
		}

		//DISTINCT ON
		distinctOnQuery, _ := model.BuildDistinctOn(body, currentAlias)

		//GROUP BY
		groupByQuery, _ := model.BuildGroupBy(body, currentAlias)

		// LIMIT AND OFFSET
		paginationQuery, paginationArgs := model.GetPagination(body, idx)
		args = append(args, paginationArgs...)

		// ORDER BY
		orderByQuery, orderByArgs := model.BuildOrderBy(body, currentAlias, idx)
		args = append(args, orderByArgs...)

		query += fmt.Sprintf(`SELECT row_to_json((SELECT  %s FROM (SELECT %s%s ) %s )) %s FROM ( SELECT %s * FROM %s.%s %s %s %s %s %s) %s`,
			currentAlias,
			modelColumnsString,
			"%s",
			currentAlias,
			currentAlias,
			distinctOnQuery,
			model.Database,
			model.Table,
			currentAlias,
			builder.RelationWhereJoin,
			groupByQuery,
			orderByQuery,
			paginationQuery,
			currentAlias)

		if err == nil {
			relationalKeys := GetRelationalKeys(parsedBody)
			for _, key := range relationalKeys {
				relatedModelInfo, err := model.GetModelRelationInfo(key)
				if err != nil {
					continue
				}
				relatedModel, err := model.GetModelRelation(key)
				if err != nil {
					continue
				}

				if bodyRelation, err := IsMapToInterface(bodyEntities); err == nil {
					if IsAggregation(key) {
						depth = depth + 1
						queryStr, queryArgs := relatedModel.SelectAggregate(role, bodyRelation[key], depth, idx, relatedModelInfo, currentAlias, key)
						relationQueryAlias := fmt.Sprintf("_%d_%s", depth, relatedModel.Table)
						query = fmt.Sprintf(query, fmt.Sprintf(",%s.%s%s", relationQueryAlias, key, "%s"))
						query += fmt.Sprintf(` LEFT OUTER JOIN LATERAL (%s) AS %s on true `, queryStr, relationQueryAlias)
						args = append(args, queryArgs...)
					} else {
						depth = depth + 1
						queryStr, queryArgs := relatedModel.Select(role, bodyRelation[key], depth, idx, relatedModelInfo, currentAlias)
						relationQueryAlias := fmt.Sprintf("_%d_%s", depth, relatedModel.Table)
						query = fmt.Sprintf(query, fmt.Sprintf(",%s.%s%s", relationQueryAlias, relatedModelInfo.Alias, "%s"))
						query += fmt.Sprintf(` LEFT OUTER JOIN LATERAL (%s) AS %s on true `, queryStr, relationQueryAlias)
						args = append(args, queryArgs...)
					}

				}
			}
		}
	}

	makeQuery(model, body, model.Table)
	query += fmt.Sprintf(") _%d_%s", depth, model.Table)
	query = fmt.Sprintf(query, "")

	return query, args
}

func (model *Model) SelectAggregate(role string, body interface{}, depth int, idx *int, relationInfo *DatabaseRelationSchema, parentAlias string, aggregation_name string) (string, []interface{}) {
	query := ``
	args := make([]interface{}, 0)
	if !IsEligibleModelRequestBody(body) {
		return query, args
	}
	builder := GetRelationalCoalesceSymbols(model, relationInfo, depth, parentAlias)
	makeQuery := func(model *Model, bodyEntities interface{}, aliasPart string) {
		parsedBody, err := IsMapToInterface(bodyEntities)
		if err != nil {
			return
		}
		currentAlias := fmt.Sprintf("_%d_%s", depth, aliasPart)
		if _where, ok := parsedBody["_where"]; ok {
			initialQuery := " WHERE "
			binder := ""
			if len(builder.RelationWhereJoin) > 0 {
				initialQuery = builder.RelationWhereJoin
				binder = "AND"
			}
			whereQuery, whereArgs := model.BuildWhereClause(_where, currentAlias, idx, initialQuery, binder)
			args = append(args, whereArgs...)
			builder.RelationWhereJoin = whereQuery
		}

		//DISTINCT ON
		distinctOnQuery, _ := model.BuildDistinctOn(body, currentAlias)

		//GROUP BY
		groupByQuery, _ := model.BuildGroupBy(body, currentAlias)

		// LIMIT AND OFFSET
		paginationQuery, paginationArgs := model.GetPagination(body, idx)
		args = append(args, paginationArgs...)

		// ORDER BY
		orderByQuery, orderByArgs := model.BuildOrderBy(body, currentAlias, idx)
		args = append(args, orderByArgs...)

		// MAIN AGGREGATION JSON OBJECT BUILDER
		queryString := model.BuildAggregate(role, parsedBody, currentAlias)

		query += fmt.Sprintf(`SELECT %s as %s FROM ( SELECT %s * FROM %s.%s %s %s %s %s %s) %s`,
			queryString,
			aggregation_name,
			distinctOnQuery,
			model.Database,
			model.Table,
			currentAlias,
			builder.RelationWhereJoin,
			groupByQuery,
			orderByQuery,
			paginationQuery,
			currentAlias)
	}

	makeQuery(model, body, model.Table)
	return query, args
}

func (model *Model) Insert(role string, ctx context.Context, tx *sql.Tx, body interface{}) (interface{}, error) {
	query, args, err := model.InsertOneQueryBuilder(role, body, nil)
	if err != nil {
		return nil, err
	}

	cb := QueryContext(ctx, tx, query, args...)
	row, err := model.ScanOneFromReturningResult(cb)
	if err != nil {
		return nil, err
	}

	entry, err := IsMapToInterface(body)
	if err == nil {
		relations := model.GetRelationalColumnsFromPayload(entry)
		parsedRow, parsedRowErr := IsMapToInterface(row)
		if parsedRowErr != nil {
			return nil, parsedRowErr
		}
		for _, key := range relations {
			relatedModel, err := model.GetModelRelation(key)
			if err != nil {
				return nil, err
			}
			relationalInfo, err := model.GetModelRelationInfo(key)
			if err != nil {
				return nil, err
			}
			parsedBody, err := IsMapToInterface(body)
			if err != nil {
				return nil, err
			}

			relationalInput, ok := parsedBody[key]

			if !ok {
				return nil, fmt.Errorf("malformed insertion")
			}

			parsedRelationalInput, err := IsMapToInterface(relationalInput)

			if err != nil {
				return nil, err
			}

			objects, ok := parsedRelationalInput["objects"]

			if !ok {
				return nil, fmt.Errorf("malformed insertion")
			}

			parsedObjects, err := IsArray(objects)

			if err != nil {
				return nil, err
			}

			relationalResults := make([]interface{}, 0)
			for _, input := range parsedObjects {

				parsedInput, err := IsMapToInterface(input)
				if err != nil {
					return nil, err
				}
				if parsedRowErr == nil {
					value, ok := parsedRow[relationalInfo.FromColumn]
					if ok {
						parsedInput[relationalInfo.ToColumn] = value
						result, err := relatedModel.Insert(role, ctx, tx, parsedInput)
						if err != nil {
							return nil, err
						}
						relationalResults = append(relationalResults, result)
					} else {
						return nil, fmt.Errorf("could not enhance entry with relational column")
					}
				} else {
					return nil, fmt.Errorf("could not enhance entry with relational column")
				}
			}
			parsedRow[key] = relationalResults

		}
	}
	// insertionResult := InsertionResult{
	// 	LastInsertId: lastInsertId,
	// 	RowsAffected: affectedRows,
	// }
	return row, nil
}

func (model *Model) InsertOneQueryBuilder(role string, body interface{}, onConflict interface{}) (string, []interface{}, error) {
	query := "INSERT INTO %s.%s(%s) VALUES(%s) RETURNING *"
	args := make([]interface{}, 0)
	parsedBody, err := isEligibleInsertModelRequestBody(body)
	if err != nil {
		return query, args, fmt.Errorf("invalid body provided")
	}

	allowedColumns, err := model.GetAllowedColumnsMapByRole(role)
	if err != nil {
		return query, args, err
	}
	columnsParts := make([]string, 0)
	valuesParts := make([]string, 0)
	idx := 1
	flag := false
	for key := range parsedBody {
		_, exists := allowedColumns[key]
		if _, ok := parsedBody[key]; ok && exists {
			columnsParts = append(columnsParts, key)
			valuesParts = append(valuesParts, fmt.Sprintf("$%d", idx))
			value, err := model.GetArgumentValueByColumnType(parsedBody[key], key)
			if err != nil {
				return query, args, err
			}
			args = append(args, value)
			idx += 1
			flag = true
		}
	}

	if !flag {
		return query, args, fmt.Errorf("nothing to insert here")
	}
	query = fmt.Sprintf(query, model.Database, model.Table, strings.Join(columnsParts, ","), strings.Join(valuesParts, ","))
	return query, args, nil
}

func (model *Model) Update(role string, ctx context.Context, tx *sql.Tx, body interface{}) ([]interface{}, error) {
	parsedBody, err := IsMapToInterface(body)
	if err != nil {
		return nil, err
	}

	_set, ok := parsedBody["_set"]
	if !ok {
		_set = make(map[string]interface{})
	}
	parsedSet, err := IsMapToInterface(_set)
	if err != nil {
		return nil, err
	}
	query := fmt.Sprintf(`UPDATE %s.%s SET `, model.Database, model.Table)
	columnParts := make([]string, 0)
	args := make([]interface{}, 0)
	idx := 1
	for key, value := range parsedSet {
		if _, ok := model.ColumnsMap[key]; ok {
			columnParts = append(columnParts, fmt.Sprintf("%s = $%d", key, idx))
			idx += 1
			transformedValue, err := model.GetArgumentValueByColumnType(value, key)
			if err != nil {
				return nil, fmt.Errorf("invalid value provided")
			}
			args = append(args, transformedValue)
		}
	}

	for operator, symbol := range UPDATE_SELF_REFERENCING_OPERATORS {
		payload, ok := parsedBody[operator]
		if !ok {
			continue
		}
		parsedPayload, err := IsMapToInterface(payload)
		if err != nil {
			return nil, err
		}
		for key, value := range parsedPayload {
			if _, ok := model.ColumnsMap[key]; ok {
				columnParts = append(columnParts, fmt.Sprintf("%s = %s %s $%d", key, key, symbol, idx))
				idx += 1
				transformedValue, err := model.GetArgumentValueByColumnType(value, key)
				if err != nil {
					return nil, fmt.Errorf("invalid value provided")
				}
				args = append(args, transformedValue)
			}
		}

	}

	if len(columnParts) == 0 {
		return nil, fmt.Errorf("invalid update input")
	}

	query += fmt.Sprintf(" %s ", strings.Join(columnParts, ", "))

	var _where any
	if err == nil {
		where, ok := parsedBody["_where"]
		if ok {
			_where = where
		}
	}
	whereClause, whereArgs := model.BuildWhereClause(_where, model.Table, &idx, "", "")
	if len(whereClause) > 0 {
		args = append(args, whereArgs...)
		query += fmt.Sprintf(" WHERE %s ", whereClause)
	}

	query += " RETURNING * "
	cb := QueryContext(ctx, tx, query, args...)
	return model.ScanManyFromReturningResult(cb)
}

func (model *Model) Delete(role string, ctx context.Context, tx *sql.Tx, body interface{}) ([]interface{}, error) {
	query := fmt.Sprintf(`DELETE FROM %s.%s`, model.Database, model.Table)
	idx := 1
	parsedBody, err := IsMapToInterface(body)
	var _where any
	if err == nil {
		where, ok := parsedBody["_where"]
		if ok {
			_where = where
		}
	}
	whereClause, args := model.BuildWhereClause(_where, model.Table, &idx, "", "")
	if len(whereClause) > 0 {
		query += fmt.Sprintf(" WHERE %s ", whereClause)
	}
	query += " RETURNING * "
	cb := QueryContext(ctx, tx, query, args...)
	return model.ScanManyFromReturningResult(cb)
}

func (model *Model) ScanOneFromReturningResult(cb func(func(rows *sql.Rows) error) error) (any, error) {
	row := make(map[string]any)
	scanner := func(rows *sql.Rows) error {
		cols, err := rows.Columns()
		if err != nil {
			return err
		}
		values := make([]interface{}, len(cols))
		ptrs := make([]interface{}, len(cols))
		for i := range values {
			ptrs[i] = &values[i]
		}
		err = rows.Scan(ptrs...)
		if err != nil {
			return err
		}
		for i, col := range cols {
			if columnType, ok := model.ColumnsMap[col]; ok {
				if columnType == "json" || columnType == "jsonb" {
					var val any
					x, ok := values[i].([]byte)
					if ok {
						json.Unmarshal(x, &val)
						row[col] = val
						continue
					}
				}
				row[col] = values[i]
			}
		}
		return nil
	}

	err := cb(scanner)

	return row, err
}

func (model *Model) ScanManyFromReturningResult(cb func(func(rows *sql.Rows) error) error) ([]any, error) {
	results := make([]interface{}, 0)
	scanner := func(rows *sql.Rows) error {
		row := make(map[string]any)
		cols, err := rows.Columns()
		if err != nil {
			return err
		}
		values := make([]interface{}, len(cols))
		ptrs := make([]interface{}, len(cols))
		for i := range values {
			ptrs[i] = &values[i]
		}
		err = rows.Scan(ptrs...)
		if err != nil {
			return err
		}
		for i, col := range cols {
			if columnType, ok := model.ColumnsMap[col]; ok {
				if columnType == "json" || columnType == "jsonb" {
					var val any
					x, ok := values[i].([]byte)
					if ok {
						json.Unmarshal(x, &val)
						row[col] = val
						continue
					}
				}
				row[col] = values[i]
			}
		}
		results = append(results, row)
		return nil
	}

	err := cb(scanner)

	return results, err
}

func (model *Model) isModelColumn(key string) bool {
	_, ok := model.ColumnsMap[key]
	return ok
}

func (model *Model) isRelationColumnWithAggregation(key string) bool {
	return strings.HasSuffix(key, "_aggregate") && model.isRelationColumn(key)
}

func (model *Model) isRelationColumn(key string) bool {
	_, err := model.GetModelRelationInfo(key)
	return err == nil
}

func (model *Model) BuildWhereClause(body interface{}, alias string, idx *int, initialQuery string, binder string) (string, []interface{}) {
	isEligibleOperation := IsEligibleWhereOperation(body)
	args := make([]interface{}, 0)
	queryString := initialQuery
	if !isEligibleOperation {
		return queryString, args
	}

	if arr, err := IsArray(body); err == nil {
		for i, value := range arr {
			qBinder := binder
			if i == 0 {
				qBinder = ""
			}
			query, newArgs := model.BuildWhereClause(value, alias, idx, initialQuery, qBinder)
			queryString += query
			args = append(args, newArgs...)
		}
	} else if operation, err := IsMapToInterface(body); err == nil {
		for key, value := range operation {
			if model.isModelColumn(key) {
				qBinder := binder
				if len(queryString) > len(initialQuery) {
					if len(qBinder) == 0 {
						qBinder = "AND"
					}
				}
				queryString += fmt.Sprintf(" %s %s.%s ", qBinder, alias, key)
				query, newArgs := model.BuildWhereClause(value, alias, idx, "", qBinder)
				queryString += query
				args = append(args, newArgs...)
			} else if _, ok := QUERY_BINDER_KEYS[key]; ok {
				query, newArgs := model.BuildWhereClause(value, alias, idx, "", WHERE_CLAUSE_KEYS[key])
				qBinder := binder
				if len(queryString) > len(initialQuery) {
					qBinder = WHERE_CLAUSE_KEYS[key]
				}
				queryString += fmt.Sprintf("%s (%s)", qBinder, query)
				args = append(args, newArgs...)
			} else if operator, ok := WHERE_CLAUSE_KEYS[key]; ok {
				if value == nil {
					queryString += fmt.Sprintf(" %s NULL", operator)
				} else {
					queryString += fmt.Sprintf(" %s $%d ", operator, *idx)
					if _, ok := REQUIRE_WILDCARD_TRANSFORMATION_KEYS[key]; ok {
						args = append(args, fmt.Sprintf("%%%s%%", value))
					} else {
						args = append(args, value)
					}
					*idx += 1
				}
			} else if model.isRelationColumnWithAggregation(key) {
				qBinder := binder
				if len(queryString) > len(initialQuery) {
					if len(binder) == 0 {
						qBinder = "AND"
					}
				}
				relationWhereAggregate := RelationWhereAggregate{
					binder:   qBinder,
					body:     value,
					modelKey: key,
					alias:    alias,
				}
				query, newArgs := model.BuildRelationalWhereAggregate(relationWhereAggregate, idx)
				queryString += query
				args = append(args, newArgs...)
			} else if model.isRelationColumn(key) {
				qBinder := binder
				if len(queryString) > len(initialQuery) {
					if len(binder) == 0 {
						qBinder = "AND"
					}
				}
				referencedModel, _ := model.GetModelRelation(key)
				referencedModelInfo, _ := model.GetModelRelationInfo(key)
				queryString += fmt.Sprintf(" %s %s.%s IN ( SELECT %s FROM %s.%s ",
					qBinder,
					alias,
					referencedModelInfo.FromColumn,
					referencedModelInfo.ToColumn,
					referencedModelInfo.Database,
					referencedModelInfo.ToTable,
				)
				query, newArgs := referencedModel.BuildWhereClause(value, referencedModelInfo.ToTable, idx, "WHERE", "")
				queryString += query
				queryString += ")"
				args = append(args, newArgs...)
			}
		}
	}
	return queryString, args
}

func (model *Model) BuildRelationalWhereAggregate(relationWhereAggregate RelationWhereAggregate, idx *int) (string, []interface{}) {
	args := make([]interface{}, 0)
	queryString := ""
	qBinder := relationWhereAggregate.binder

	referencedModelInfo, _ := model.GetModelRelationInfo(relationWhereAggregate.modelKey)
	aggreStr := ""
	parsedValue, err := IsMapToInterface(relationWhereAggregate.body)

	if err != nil {
		return queryString, args
	}
	if len(parsedValue) <= 0 {
		return queryString, args
	}

	if len(qBinder) == 0 {
		qBinder = "AND"
	}

	for aggregationKey, payload := range parsedValue {

		if _, ok := AGGREGATION_KEYS[aggregationKey]; !ok {
			return queryString, args
		}
		if aggregationKey == "_count" {
			operatorKey, parsedPayload, err := GetFirstKeyFromMap(payload)
			if err != nil {
				return queryString, args
			}
			operator, ok := WHERE_CLAUSE_KEYS[operatorKey]
			if !ok {
				return queryString, args
			}
			aggreStr += fmt.Sprintf(" %s (SELECT COUNT(*) FROM %s.%s WHERE %s.%s = %s.%s) %s $%d",
				qBinder,
				referencedModelInfo.Database,
				referencedModelInfo.ToTable,
				relationWhereAggregate.alias,
				referencedModelInfo.FromColumn,
				referencedModelInfo.ToTable,
				referencedModelInfo.ToColumn,
				operator,
				*idx,
			)

			parsedValue, ok := parsedPayload[operatorKey]
			if !ok {
				return queryString, args
			}
			args = append(args, parsedValue)
			*idx += 1
		} else {
			column, data, err := GetFirstKeyFromMap(payload)
			if err != nil {
				return queryString, args
			}
			if !model.isModelColumn(column) {
				return queryString, args
			}
			operatorKey, parsedPayload, err := GetFirstKeyFromMap(data[column])
			if err != nil {
				return queryString, args
			}
			operator, ok := WHERE_CLAUSE_KEYS[operatorKey]
			if !ok {
				return queryString, args
			}

			aggreStr += fmt.Sprintf(" %s (SELECT %s(%s) FROM %s.%s WHERE %s.%s = %s.%s ) %s $%d",
				qBinder,
				AGGREGATION_KEYS[aggregationKey],
				column,
				referencedModelInfo.Database,
				referencedModelInfo.ToTable,
				relationWhereAggregate.alias,
				referencedModelInfo.FromColumn,
				referencedModelInfo.ToTable,
				referencedModelInfo.ToColumn,
				operator,
				*idx,
			)

			parsedValue, ok := parsedPayload[operatorKey]
			if !ok {
				return queryString, args
			}
			args = append(args, parsedValue)
			*idx += 1
		}

	}
	queryString += fmt.Sprintf(" %s %s.%s IN ( SELECT %s FROM %s.%s WHERE %s.%s = %s.%s %s )",
		relationWhereAggregate.binder,
		relationWhereAggregate.alias,
		referencedModelInfo.FromColumn,
		referencedModelInfo.ToColumn,
		referencedModelInfo.Database,
		referencedModelInfo.ToTable,
		relationWhereAggregate.alias,
		referencedModelInfo.FromColumn,
		referencedModelInfo.ToTable,
		referencedModelInfo.ToColumn,
		aggreStr,
	)
	return queryString, args
}

func (model *Model) BuildLimit(body interface{}, idx *int) (string, []interface{}) {
	parsedBody, err := IsMapToInterface(body)
	queryString := ""
	args := make([]interface{}, 0)
	if err != nil {
		return queryString, args
	}
	limit, ok := parsedBody["_limit"]
	if !ok {
		return queryString, args
	}
	queryString = fmt.Sprintf("LIMIT $%d", *idx)
	args = append(args, limit)
	*idx += 1
	return queryString, args
}

func (model *Model) BuildOffset(body interface{}, idx *int) (string, []interface{}) {
	parsedBody, err := IsMapToInterface(body)
	queryString := ""
	args := make([]interface{}, 0)
	if err != nil {
		return queryString, args
	}
	offset, ok := parsedBody["_offset"]
	if !ok {
		return queryString, args
	}
	queryString = fmt.Sprintf("OFFSET $%d", *idx)
	args = append(args, offset)
	*idx += 1
	return queryString, args
}

func (model *Model) GetPagination(body interface{}, idx *int) (string, []interface{}) {
	limitQuery, args := model.BuildLimit(body, idx)
	offsetQuery, offsetArgs := model.BuildOffset(body, idx)

	queryString := fmt.Sprintf(" %s %s ", limitQuery, offsetQuery)
	args = append(args, offsetArgs...)
	return queryString, args
}

func (model *Model) BuildOrderBy(body interface{}, alias string, idx *int) (string, []interface{}) {
	queryString := ""
	args := make([]interface{}, 0)
	if !IsEligibleOrderByOperation(body) {
		return queryString, args
	}

	parsedBody, _ := IsMapToInterface(body)
	orderByFields, ok := parsedBody["_orderBy"]
	if !ok {
		return queryString, args
	}
	fields, err := IsMapToInterface(orderByFields)
	if err != nil {
		return queryString, args
	}
	parts := make([]string, 0)
	for key, value := range fields {
		switch parsedValue := value.(type) {
		case string:
			if model.isModelColumn(key) {
				if direction, ok := ORDER_BY_KEYS[parsedValue]; ok {
					parts = append(parts, fmt.Sprintf("%s %s", key, direction))
				}
			}
			// if model.isRelationColumn(key) {
			// 	relatedModel,err := model.GetModelRelation(key)
			// 	if err != nil {
			// 		return queryString,args
			// 	}
			// 	relatedModelInfo,err := model.GetModelRelationInfo(key)
			// 	if err != nil {
			// 		return queryString,args
			// 	}
			// 	orderByQuery,newArgs := relatedModel.BuildOrderBy(value,relatedModel.Table,idx)
			// 		args = append(args, newArgs...)
			// 	query := fmt.Sprintf("( SELECT )")
			// }
		}

	}
	if len(parts) > 0 {
		queryString = fmt.Sprintf(" ORDER BY %s ", strings.Join(parts, ","))
	}

	return queryString, args
}

func (model *Model) BuildDistinctOn(body interface{}, alias string) (string, []interface{}) {
	parsedBody, err := IsMapToInterface(body)
	queryString := ""
	args := make([]interface{}, 0)
	if err != nil {
		return queryString, args
	}

	distinctOnFields, ok := parsedBody["_distinct"]
	if !ok {
		return queryString, args
	}

	fields, err := IsArray(distinctOnFields)
	if err != nil {
		return queryString, args
	}
	parts := make([]string, 0)
	for _, key := range fields {
		switch column := key.(type) {
		case string:
			if model.isModelColumn(column) {
				parts = append(parts, fmt.Sprintf("%s.%s", alias, column))
			}
		}
	}

	if len(parts) > 0 {
		queryString = fmt.Sprintf(" DISTINCT ON (%s) ", strings.Join(parts, ","))
	}

	return queryString, args

}

func (model *Model) BuildGroupBy(body interface{}, alias string) (string, []interface{}) {
	parsedBody, err := IsMapToInterface(body)
	queryString := ""
	args := make([]interface{}, 0)
	if err != nil {
		return queryString, args
	}

	groupByFields, ok := parsedBody["_groupBy"]
	if !ok {
		return queryString, args
	}

	fields, err := IsArray(groupByFields)
	if err != nil {
		return queryString, args
	}
	parts := make([]string, 0)
	for _, key := range fields {
		switch column := key.(type) {
		case string:
			if model.isModelColumn(column) {
				parts = append(parts, fmt.Sprintf("%s.%s", alias, column))
			}
		}
	}

	if len(parts) > 0 {
		queryString = fmt.Sprintf(" GROUP BY %s ", strings.Join(parts, ","))
	}

	return queryString, args

}

func (model *Model) BuildAggregate(role string, body interface{}, alias string) string {
	queryParts := make([]string, 0)
	countParts := model.BuildCountAggregate(role, body)
	maxParts := model.BuildMaxAggregate(role, body, alias)
	minParts := model.BuildMinAggregate(role, body, alias)
	sumParts := model.BuildSumAggregate(role, body, alias)
	avgParts := model.BuildAVGAggregate(role, body, alias)

	if len(countParts) > 0 {
		queryParts = append(queryParts, countParts)
	}

	if len(maxParts) > 0 {
		queryParts = append(queryParts, maxParts)
	}

	if len(minParts) > 0 {
		queryParts = append(queryParts, minParts)
	}

	if len(sumParts) > 0 {
		queryParts = append(queryParts, sumParts)
	}

	if len(avgParts) > 0 {
		queryParts = append(queryParts, avgParts)
	}

	if len(queryParts) > 0 {
		return fmt.Sprintf("json_build_object(%s)", strings.Join(queryParts, ","))
	}

	return ""
}

func (model *Model) BuildAggregateForHaving(role string, body interface{}, alias string) string {
	queryParts := make([]string, 0)
	countParts := model.BuildCountAggregate(role, body)
	maxParts := model.BuildMaxAggregate(role, body, alias)
	minParts := model.BuildMinAggregate(role, body, alias)
	sumParts := model.BuildSumAggregate(role, body, alias)
	avgParts := model.BuildAVGAggregate(role, body, alias)

	if len(countParts) > 0 {
		queryParts = append(queryParts, countParts)
	}

	if len(maxParts) > 0 {
		queryParts = append(queryParts, maxParts)
	}

	if len(minParts) > 0 {
		queryParts = append(queryParts, minParts)
	}

	if len(sumParts) > 0 {
		queryParts = append(queryParts, sumParts)
	}

	if len(avgParts) > 0 {
		queryParts = append(queryParts, avgParts)
	}

	if len(queryParts) > 0 {
		return fmt.Sprintf("json_build_object(%s)", strings.Join(queryParts, ","))
	}

	return ""
}

func (model *Model) BuildCountAggregate(role string, body interface{}) string {
	parsedBody, err := IsMapToInterface(body)
	if err != nil {
		return ""
	}

	count, ok := parsedBody["_count"]
	if !ok {
		return ""
	}
	switch count.(type) {
	case bool:
		return "'count',COUNT(*)"
	default:
		return ""
	}

}

func (model *Model) BuildMinAggregate(role string, body interface{}, alias string) string {
	parsedBody, err := IsMapToInterface(body)
	if err != nil {
		return ""
	}

	min, ok := parsedBody["_min"]

	if !ok {
		return ""
	}
	parsedMin, err := IsArray(min)
	if err != nil {
		return ""
	}
	allowedColumns, err := model.GetAllowedColumnsMapByRole(role)
	if err != nil {
		return ""
	}
	parts := make([]string, 0)
	for _, key := range parsedMin {
		switch column := key.(type) {
		case string:
			if _, ok := allowedColumns[column]; ok {
				parts = append(parts, fmt.Sprintf("'%s',MIN(%s.%s)", column, alias, column))
			}
		}
	}

	if len(parts) > 0 {
		return fmt.Sprintf("'min',json_build_object(%s)", strings.Join(parts, ","))
	}
	return ""
}

func (model *Model) BuildMaxAggregate(role string, body interface{}, alias string) string {
	parsedBody, err := IsMapToInterface(body)
	if err != nil {
		return ""
	}

	max, ok := parsedBody["_max"]
	if !ok {
		return ""
	}
	parsedMax, err := IsArray(max)
	if err != nil {
		return ""
	}
	allowedColumns, err := model.GetAllowedColumnsMapByRole(role)
	if err != nil {
		return ""
	}
	parts := make([]string, 0)

	for _, key := range parsedMax {
		switch column := key.(type) {
		case string:
			if _, ok := allowedColumns[column]; ok {
				parts = append(parts, fmt.Sprintf("'%s',MAX(%s.%s)", column, alias, column))
			}
		}
	}

	if len(parts) > 0 {
		return fmt.Sprintf("'max',json_build_object(%s)", strings.Join(parts, ","))
	}
	return ""
}

func (model *Model) BuildSumAggregate(role string, body interface{}, alias string) string {
	parsedBody, err := IsMapToInterface(body)
	if err != nil {
		return ""
	}

	sum, ok := parsedBody["_sum"]
	if !ok {
		return ""
	}
	parsedSum, err := IsArray(sum)
	if err != nil {
		return ""
	}
	allowedColumns, err := model.GetAllowedColumnsMapByRole(role)
	if err != nil {
		return ""
	}
	parts := make([]string, 0)

	for _, key := range parsedSum {
		switch column := key.(type) {
		case string:
			if _, ok := allowedColumns[column]; ok {
				parts = append(parts, fmt.Sprintf("'%s',SUM(%s.%s)", column, alias, column))
			}
		}

	}

	if len(parts) > 0 {
		return fmt.Sprintf("'sum',json_build_object(%s)", strings.Join(parts, ","))
	}
	return ""
}

func (model *Model) BuildAVGAggregate(role string, body interface{}, alias string) string {
	parsedBody, err := IsMapToInterface(body)
	if err != nil {
		return ""
	}

	avg, ok := parsedBody["_avg"]
	if !ok {
		return ""
	}
	parsedAVG, err := IsArray(avg)
	if err != nil {
		return ""
	}
	allowedColumns, err := model.GetAllowedColumnsMapByRole(role)
	if err != nil {
		return ""
	}
	parts := make([]string, 0)

	for _, key := range parsedAVG {
		switch column := key.(type) {
		case string:
			if _, ok := allowedColumns[column]; ok {
				parts = append(parts, fmt.Sprintf("'%s',AVG(%s.%s)", column, alias, column))
			}
		}
	}

	if len(parts) > 0 {
		return fmt.Sprintf("'avg',json_build_object(%s)", strings.Join(parts, ","))
	}
	return ""
}

func (model *Model) GetModelColumnsWithAlias(role string, body interface{}, alias string) string {
	columns := []string{}
	prefix := ""
	if len(alias) > 0 {
		prefix = fmt.Sprintf("%s.", alias)
	}
	parsedBody, err := IsMapToInterface(body)

	hasSelect := false
	_select := make(map[string]interface{})
	if err == nil {
		x, ok := parsedBody["_select"]

		if ok {
			switch y := x.(type) {
			case map[string]interface{}:
				if len(y) > 0 {
					_select = y
					hasSelect = true
				}
			}

		}
	}

	allowedColumnsMap, err := model.GetAllowedColumnsMapByRole(role)
	if err != nil {
		return ""
	}
	for column := range allowedColumnsMap {
		if hasSelect {
			if _, ok := _select[column]; ok {
				columns = append(columns, fmt.Sprintf("%s%s", prefix, column))

			}
		} else {
			columns = append(columns, fmt.Sprintf("%s%s", prefix, column))
		}
	}

	return strings.Join(columns, ",")
}

func (model *Model) GetAllowedColumnsMapByRole(role string) (ColumnsMap, error) {
	allowedColumnsMap, ok := model.RLS[role]
	if !ok && len(model.RLS) > 0 {
		return nil, fmt.Errorf("no columns are available")
	}

	if len(model.RLS) == 0 {
		allowedColumnsMap = model.ColumnsMap
	}

	return allowedColumnsMap, nil
}

func (model *Model) GetArgumentValueByColumnType(value interface{}, key string) (interface{}, error) {
	columnType, ok := model.ColumnsMap[key]
	if !ok {
		return nil, fmt.Errorf("invalid column %s for parsed value", key)
	}

	if strings.HasSuffix(columnType, "[]") {
		return pq.Array(value), nil
	}

	switch columnType {
	case "json":
	case "jsonb":
		parsedValue, err := json.Marshal(value)
		if err != nil {
			return value, err
		}
		return parsedValue, nil
	default:
		break
	}
	return value, nil
}

func (model *Model) GetRelationalColumnsFromPayload(payload map[string]interface{}) []string {
	relations := make([]string, 0)
	for key := range payload {
		_, err := model.GetModelRelationInfo(key)
		if err == nil {
			relations = append(relations, key)
		}
	}

	return relations
}
