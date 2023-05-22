package database

import (
	"database/sql"
	"fmt"
	"strings"
)

type EngineRLS struct {
	PolicyName  string `json:"policy_name"`
	PolicyFor   string `json:"policy_for"`
	PolicyType  string `json:"policy_type"`
	Database    string `json:"database"`
	Table       string `json:"table"`
	Id          int    `json:"id"`
	Enabled     bool   `json:"enabled"`
	CreatedAt   string `json:"created_at"`
	SQL         string `json:"sql"`
	Description string `json:"description"`
}

type RLS struct {
	RLSConfig  EngineRLS `json:"rls_config"`
	PolicyName string    `json:"policy_name"`
	Table      string    `json:"table"`
	Database   string    `json:"database"`
}

type RLSInput EngineRLS

type EnableRLSForDatabaseInput struct {
	Database string `json:"database"`
}

type EnableRlsForDatabaseTableInput struct {
	Database string `json:"database"`
	Table    string `json:"table"`
	Force    bool   `json:"force"`
}

func ValidateRLSPolicyFor(input RLSInput) error {
	switch input.PolicyFor {
	case "SELECT", "INSERT", "UPDATE", "DELETE":
		return nil
	default:
		return fmt.Errorf("not supported policy for configuration")
	}

	return nil
}

func ValidateRLSPolicyType(input RLSInput) error {
	switch input.PolicyType {
	case "PERMISSIVE", "RESTRICTIVE":
		return nil
	default:
		return fmt.Errorf("not supported policy type")
	}

	return nil
}

func (e *Engine) ValidatePolicyName(input RLSInput, unique bool) error {
	if len(input.PolicyName) == 0 {
		return fmt.Errorf("no policy name was provided")
	}

	if unique {
		for _, policy := range e.EngineRLS {
			if policy.PolicyName == input.PolicyName {
				return fmt.Errorf("policy name is not unique")
			}
		}
	}

	return nil
}

func ValidateRLSSQL(input RLSInput) error {
	sql := strings.Trim(input.SQL, " ")
	if len(sql) == 0 {
		return fmt.Errorf("please provide condition for policy")
	}

	return nil
}

func FormatPolicyName(input RLSInput) string {
	return strings.Trim(strings.ToLower(input.PolicyName), " ")
}

func DeriveStaticQueryStringByPolicyFor(input RLSInput) string {
	if input.PolicyFor == "SELECT" {
		return CREATE_QUERY_POLICY
	}

	return CREATE_STATEMENT_POLICY
}

func (e *Engine) LoadRLS(db *sql.DB) ([]RLS, error) {
	policies := []RLS{}
	scanner := Query(db, GET_POLICIES)
	cb := func(rows *sql.Rows) error {
		var policy RLS
		err := rows.Scan(&policy.PolicyName, &policy.Table, &policy.Database)
		if err != nil {
			return err
		}

		policies = append(policies, policy)
		return err
	}
	err := scanner(cb)
	if err != nil {
		return nil, err
	}

	engine_rls := []EngineRLS{}

	scanner = Query(db, ENGINE_GET_RLS)
	cb = func(rows *sql.Rows) error {
		var policy EngineRLS
		err := rows.Scan(&policy.Id, &policy.PolicyName, &policy.PolicyFor, &policy.PolicyType, &policy.Database, &policy.Table, &policy.Enabled, &policy.CreatedAt, &policy.SQL, &policy.Description)
		if err != nil {
			return err
		}

		engine_rls = append(engine_rls, policy)
		return err
	}
	err = scanner(cb)
	if err != nil {
		return nil, err
	}

	realPolicies := []RLS{}
	for _, rls := range engine_rls {
		for _, rls_original := range policies {
			if rls.PolicyName == rls_original.PolicyName && rls.Database == rls_original.Database && rls.Table == rls_original.Table {
				rls.Enabled = true
			}
		}
		realPolicies = append(realPolicies, RLS{
			PolicyName: rls.PolicyName,
			Database:   rls.Database,
			Table:      rls.Table,
			RLSConfig:  rls,
		})
	}
	e.EngineRLS = realPolicies
	return realPolicies, nil
}

func EnableRLS(db *sql.DB) error {
	_, err := db.Exec(ENABLE_RLS_FOR_DATABASE_POSTGRES)
	return err
}

func (e *Engine) EnableRLSForDatabase(db *sql.DB, input EnableRLSForDatabaseInput) error {
	tables, ok := e.DatabaseToTableToModelMap[input.Database]
	if !ok {
		return fmt.Errorf("no tables available for database %s", input.Database)
	}

	for _, model := range tables {
		_, err := db.Exec(fmt.Sprintf(ENABLE_RLS_FOR_TABLE, model.Database, model.Table))
		if err != nil {
			return fmt.Errorf("%s: failed on enabled row level security for %s %s", err.Error(), model.Database, model.Table)
		}
		_, err = db.Exec(fmt.Sprintf(FORCE_RLS_FOR_TABLE, model.Database, model.Table))
		if err != nil {
			return fmt.Errorf("%s: failed on forcing row level security for %s %s", err.Error(), model.Database, model.Table)
		}
	}

	return nil
}

func (e *Engine) DisableRLSForDatabase(db *sql.DB, input EnableRLSForDatabaseInput) error {
	tables, ok := e.DatabaseToTableToModelMap[input.Database]
	if !ok {
		return fmt.Errorf("no tables available for database %s", input.Database)
	}

	for _, model := range tables {
		_, err := db.Exec(fmt.Sprintf(DISABLE_RLS_FOR_TABLE, model.Database, model.Table))
		if err != nil {
			return fmt.Errorf("%s: failed on enabled row level security for %s %s", err.Error(), model.Database, model.Table)
		}
	}

	return nil
}

func (e *Engine) EnableRLSForTable(db *sql.DB, input EnableRlsForDatabaseTableInput) error {
	tables, ok := e.DatabaseToTableToModelMap[input.Database]
	if !ok {
		return fmt.Errorf("no tables available for database %s", input.Database)
	}

	model, ok := tables[input.Table]
	if !ok {
		return fmt.Errorf("no table %s available for database %s", input.Table, input.Database)
	}
	_, err := db.Exec(fmt.Sprintf(ENABLE_RLS_FOR_TABLE, model.Database, model.Table))
	if err != nil {
		return fmt.Errorf("%s: failed on enabled row level security for %s %s", err.Error(), model.Database, model.Table)
	}
	if input.Force {
		_, err := db.Exec(fmt.Sprintf(FORCE_RLS_FOR_TABLE, model.Database, model.Table))
		if err != nil {
			return fmt.Errorf("%s: failed on forcing row level security for %s %s", err.Error(), model.Database, model.Table)
		}
	}

	return nil

}

func (e *Engine) DisableRLSForTable(db *sql.DB, input EnableRlsForDatabaseTableInput) error {
	tables, ok := e.DatabaseToTableToModelMap[input.Database]
	if !ok {
		return fmt.Errorf("no tables available for database %s", input.Database)
	}

	model, ok := tables[input.Table]
	if !ok {
		return fmt.Errorf("no table %s available for database %s", input.Table, input.Database)
	}
	_, err := db.Exec(fmt.Sprintf(DISABLE_RLS_FOR_TABLE, model.Database, model.Table))
	if err != nil {
		return fmt.Errorf("%s: failed on enabled row level security for %s %s", err.Error(), model.Database, model.Table)
	}

	return nil

}

func (e *Engine) CreateEngineRLS(db *sql.DB, input RLSInput) error {
	_, err := e.GetModelByKey(input.Database, input.Table)

	if err != nil {
		return err
	}

	err = ValidateRLSPolicyFor(input)
	if err != nil {
		return err
	}
	err = ValidateRLSPolicyType(input)
	if err != nil {
		return err
	}

	input.PolicyName = FormatPolicyName(input)

	err = e.ValidatePolicyName(input, true)
	if err != nil {
		return err
	}

	err = ValidateRLSSQL(input)
	if err != nil {
		return err
	}

	staticQuery := DeriveStaticQueryStringByPolicyFor(input)

	input.SQL = fmt.Sprintf(staticQuery,
		input.PolicyName,
		input.Database,
		input.Table,
		input.PolicyType,
		input.PolicyFor,
		input.SQL,
	)

	_, err = db.Exec(input.SQL)
	if err != nil {
		return err
	}

	_, err = db.Exec(CREATE_ENGINE_RLS,
		input.PolicyName,
		input.PolicyFor,
		input.PolicyType,
		input.Database,
		input.Table,
		input.Enabled,
		input.SQL,
		input.Description,
	)

	if err != nil {
		return err
	}

	err = EnableRLS(db)
	if err != nil {
		return err
	}

	enableDataBaseRLSInput := EnableRLSForDatabaseInput{
		Database: input.Database,
	}
	err = e.EnableRLSForDatabase(db, enableDataBaseRLSInput)
	if err != nil {
		return err
	}
	enableDatabaseTableInput := EnableRlsForDatabaseTableInput{
		Database: input.Database,
		Table:    input.Table,
		Force:    true,
	}
	err = e.EnableRLSForTable(db, enableDatabaseTableInput)

	return err
}

func (e *Engine) DropEngineRLS(db *sql.DB, input RLSInput) error {
	_, err := e.GetModelByKey(input.Database, input.Table)
	if err != nil {
		return err
	}

	staticQuery := fmt.Sprintf(DROP_TABLE_POLICY,
		input.PolicyName,
		input.Database,
		input.Table,
	)

	_, err = db.Exec(staticQuery)
	if err != nil {
		return err
	}

	_, err = db.Exec(DELETE_ENGINE_RLS, input.PolicyName)

	return err

}
