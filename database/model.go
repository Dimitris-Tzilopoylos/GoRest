package database

import (
	"fmt"
	"strings"
)

type Column struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	MaxLength    int64  `json:"max_length"`
	Nullable     bool   `json:"nullable"`
	DefaultValue string `json:"default_value"`
}

type ModelRelation *Model
type RelationMap map[string]ModelRelation
type RelationInfoMap map[string]DatabaseRelationSchema
type ColumnsMap map[string]string

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
	}
}

func IsAggregation(alias string) bool {
	return strings.HasSuffix(alias, "_aggregate")
}

func ClearAliasForAggregate(alias string) string {
	return strings.Split(alias, "_aggregate")[0]
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

func (model *Model) Select(body interface{}, depth int, idx *int, relationInfo *DatabaseRelationSchema, parentAlias string) (string, []interface{}) {
	query := ``
	args := make([]interface{}, 0)
	if !IsEligibleModelRequestBody(body) {
		return query, args
	}
	builder := GetRelationalCoalesceSymbols(model, relationInfo, depth, parentAlias)
	query = fmt.Sprintf(`SELECT coalesce(jsonb_agg(_%d_%s)%s,'%s') as %s FROM (`,
		depth,
		model.Table,
		builder.RelationExtractSymbol,
		builder.RelationCoalesceSymbol,
		builder.RelationAlias)
	makeQuery := func(model *Model, bodyEntities interface{}, aliasPart string) {
		parsedBody, err := IsMapToInterface(bodyEntities)
		currentAlias := fmt.Sprintf("_%d_%s", depth, aliasPart)
		modelColumnsString := GetModelColumnsWithAlias(model, currentAlias)

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
						queryStr, queryArgs := relatedModel.SelectAggregate(bodyRelation[key], depth+1, idx, relatedModelInfo, currentAlias, key)
						relationQueryAlias := fmt.Sprintf("_%d_%s", depth+1, relatedModel.Table)
						query = fmt.Sprintf(query, fmt.Sprintf(",%s.%s%s", relationQueryAlias, key, "%s"))
						query += fmt.Sprintf(` LEFT OUTER JOIN LATERAL (%s) AS %s on true `, queryStr, relationQueryAlias)
						args = append(args, queryArgs...)
					} else {
						queryStr, queryArgs := relatedModel.Select(bodyRelation[key], depth+1, idx, relatedModelInfo, currentAlias)
						relationQueryAlias := fmt.Sprintf("_%d_%s", depth+1, relatedModel.Table)
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

func (model *Model) SelectAggregate(body interface{}, depth int, idx *int, relationInfo *DatabaseRelationSchema, parentAlias string, aggregation_name string) (string, []interface{}) {
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
		queryString := model.BuildAggregate(parsedBody, currentAlias)

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
				referencedModel, _ := model.GetModelRelation(key)
				referencedModelInfo, _ := model.GetModelRelationInfo(key)
				query, newArgs := referencedModel.BuildWhereClause(value, referencedModelInfo.ToTable, idx, "WHERE", "")
				queryString += fmt.Sprintf(" %s %s.%s IN ( SELECT %s FROM %s.%s %s)",
					qBinder,
					alias,
					referencedModelInfo.FromColumn,
					referencedModelInfo.ToColumn,
					referencedModelInfo.Database,
					referencedModelInfo.ToTable,
					query,
				)
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

func (model *Model) BuildAggregate(body interface{}, alias string) string {
	queryParts := make([]string, 0)
	countParts := model.BuildCountAggregate(body)
	maxParts := model.BuildMaxAggregate(body, alias)
	minParts := model.BuildMinAggregate(body, alias)
	sumParts := model.BuildSumAggregate(body, alias)
	avgParts := model.BuildAVGAggregate(body, alias)

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

func (model *Model) BuildCountAggregate(body interface{}) string {
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

func (model *Model) BuildMinAggregate(body interface{}, alias string) string {
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

	parts := make([]string, 0)
	for _, key := range parsedMin {
		switch column := key.(type) {
		case string:
			if model.isModelColumn(column) {
				parts = append(parts, fmt.Sprintf("'%s',MIN(%s.%s)", column, alias, column))
			}
		}
	}

	if len(parts) > 0 {
		return fmt.Sprintf("'min',json_build_object(%s)", strings.Join(parts, ","))
	}
	return ""
}

func (model *Model) BuildMaxAggregate(body interface{}, alias string) string {
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

	parts := make([]string, 0)

	for _, key := range parsedMax {
		switch column := key.(type) {
		case string:
			if model.isModelColumn(column) {
				parts = append(parts, fmt.Sprintf("'%s',MAX(%s.%s)", column, alias, column))
			}
		}
	}

	if len(parts) > 0 {
		return fmt.Sprintf("'max',json_build_object(%s)", strings.Join(parts, ","))
	}
	return ""
}

func (model *Model) BuildSumAggregate(body interface{}, alias string) string {
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

	parts := make([]string, 0)

	for _, key := range parsedSum {
		switch column := key.(type) {
		case string:
			if model.isModelColumn(column) {
				parts = append(parts, fmt.Sprintf("'%s',SUM(%s.%s)", column, alias, column))
			}
		}

	}

	if len(parts) > 0 {
		return fmt.Sprintf("'sum',json_build_object(%s)", strings.Join(parts, ","))
	}
	return ""
}

func (model *Model) BuildAVGAggregate(body interface{}, alias string) string {
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

	parts := make([]string, 0)

	for _, key := range parsedAVG {
		switch column := key.(type) {
		case string:
			if model.isModelColumn(column) {
				parts = append(parts, fmt.Sprintf("'%s',AVG(%s.%s)", column, alias, column))
			}
		}
	}

	if len(parts) > 0 {
		return fmt.Sprintf("'avg',json_build_object(%s)", strings.Join(parts, ","))
	}
	return ""
}
