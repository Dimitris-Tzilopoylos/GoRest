package database

var SELECT_BODY_KEYS map[string]bool = map[string]bool{
	"_where":    true,
	"_select":   true,
	"_orderBy":  true,
	"_groupBy":  true,
	"_distinct": true,
	"_offset":   true,
	"_limit":    true,
}

var WHERE_CLAUSE_KEYS map[string]string = map[string]string{
	"_and":            "AND",
	"_or":             "OR",
	"_eq":             "=",
	"_neq":            "<>",
	"_gt":             ">",
	"_gte":            ">=",
	"_lt":             "<",
	"_lte":            "<=",
	"_ilike":          "iLIKE",
	"_like":           "LIKE",
	"_is":             "IS",
	"_is_not":         "IS NOT",
	"_in":             "= ANY",
	"_any":            "= ANY",
	"_nany":           "<> ANY",
	"_all":            "= ALL",
	"_nin":            "<> ALL",
	"_contains":       "@>",
	"_contained_in":   "<@",
	"_key_exists":     "?",
	"_key_exists_any": "?|",
	"_key_exists_all": "?&",
	"_text_search":    "tsquery",
}

var QUERY_BINDER_KEYS map[string]bool = map[string]bool{
	"_and": true,
	"_or":  true,
}

var REQUIRE_ARRAY_TRANSFORMATION_KEYS map[string]bool = map[string]bool{
	"_in":   true,
	"_nin":  true,
	"_any":  true,
	"_nany": true,
	"_all":  true,
}

var REQUIRE_WILDCARD_TRANSFORMATION_KEYS map[string]bool = map[string]bool{
	"_like":  true,
	"_ilike": true,
}

var AGGREGATION_KEYS map[string]string = map[string]string{
	"_count": "COUNT",
	"_min":   "MIN",
	"_max":   "MAX",
	"_avg":   "AVG",
	"_sum":   "SUM",
}

var ORDER_BY_KEYS map[string]string = map[string]string{
	"ASC":              "ASC",
	"ASC_NULLS_FIRST":  "ASC NULLS FIRST",
	"ASC_NULLS_LAST":   "ASC NULLS LAST",
	"DESC":             "DESC",
	"DESC_NULLS_FIRST": "DESC NULLS FIRST",
	"DESC_NULLS_LAST":  "DESC NULLS LAST",
}

var UPDATE_SELF_REFERENCING_OPERATORS = map[string]string{
	"_inc":  " + ",
	"_dec":  " - ",
	"_div":  " / ",
	"_mult": " * ",
}
