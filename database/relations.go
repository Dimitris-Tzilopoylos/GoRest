package database

import (
	"application/environment"
	"database/sql"
	"fmt"
)

var ARRAY string = "ARRAY"
var OBJECT string = "OBJECT"

type DatabaseRelationSchema struct {
	Id           int64  `json:"id"`
	Alias        string `json:"alias"`
	Database     string `json:"database"`
	FromTable    string `json:"from_table"`
	FromColumn   string `json:"from_column"`
	ToTable      string `json:"to_table"`
	ToColumn     string `json:"to_column"`
	RelationType string `json:"relation_type"`
	Table        string `json:"table"`
}

func CreateRelation(db *sql.DB, relationInput DatabaseRelationSchema) error {
	query := fmt.Sprintf(`INSERT INTO %s.relations(alias,db,from_table,to_table,from_column,to_column,relation) VALUES($1,$2,$3,$4,$5,$6,$7)`,
		environment.GetEnvValue("INTERNAL_SCHEMA_NAME"))
	_, err := db.Query(query, relationInput.Alias, relationInput.Database, relationInput.FromTable, relationInput.ToTable, relationInput.FromColumn, relationInput.ToColumn, relationInput.RelationType)
	return err
}

func UpdateRelationByID(db *sql.DB, relationInput DatabaseRelationSchema) error {
	query := fmt.Sprintf(`UPDATE %s.relations SET db = $1,from_table=$2,to_table=$3,from_column=$4,to_column=$5,alias=$6,relation=$7 WHERE id = $8`, environment.GetEnvValue("INTERNAL_SCHEMA_NAME"))
	_, err := db.Query(query, relationInput.Database, relationInput.FromTable, relationInput.ToTable, relationInput.FromColumn, relationInput.ToColumn, relationInput.Alias, relationInput.RelationType, relationInput.Id)
	return err
}

func DeleteRelationByID(db *sql.DB, relationInput DatabaseRelationSchema) error {
	query := fmt.Sprintf(`DELETE FROM %s.relations WHERE id = $1`, environment.GetEnvValue("INTERNAL_SCHEMA_NAME"))
	_, err := db.Query(query, relationInput.Id)
	return err
}

func DeleteRelationsByDatabase(db *sql.DB, relationInput DatabaseRelationSchema) error {
	query := fmt.Sprintf(`DELETE FROM %s.relations WHERE db = $1`, environment.GetEnvValue("INTERNAL_SCHEMA_NAME"))
	_, err := db.Query(query, relationInput.Database)
	return err
}

func DeleteRelationsByDatabaseTable(db *sql.DB, relationInput DatabaseRelationSchema) error {
	query := fmt.Sprintf(`DELETE FROM %s.relations WHERE db = $1 AND (from_table = $2 OR to_table = $3)`, environment.GetEnvValue("INTERNAL_SCHEMA_NAME"))
	_, err := db.Query(query, relationInput.Database, relationInput.Table, relationInput.Table)

	return err
}

func DeleteRelationsByDatabaseTableColumn(db *sql.DB, relationInput DatabaseRelationSchema) error {
	query := fmt.Sprintf(`DELETE FROM %s.relations WHERE db = $1 AND ((from_table = $2 AND from_column = $3) OR (to_table=$4 AND to_column=$5))`, environment.GetEnvValue("INTERNAL_SCHEMA_NAME"))
	_, err := db.Query(query, relationInput.Database, relationInput.Table, relationInput.FromColumn, relationInput.Table, relationInput.ToColumn)

	return err
}

func ValidateRelationParts(engineMap map[string]map[string]*Model, relationInput DatabaseRelationSchema) (bool, error) {

	if ARRAY != relationInput.RelationType && OBJECT != relationInput.RelationType {
		return false, fmt.Errorf("please provide a valid relation_type")
	}

	tablesMap, ok := engineMap[relationInput.Database]

	if !ok {
		return false, fmt.Errorf("database %s doesn't exist", relationInput.Database)
	}

	fromTable, ok := tablesMap[relationInput.FromTable]

	if !ok {
		return false, fmt.Errorf("table %s doesn't exist for database %s", relationInput.FromTable, relationInput.Database)
	}

	_, ok = fromTable.ColumnsMap[relationInput.FromColumn]

	if !ok {
		return false, fmt.Errorf("column %s doesn't exist for table %s of database %s", relationInput.FromColumn, relationInput.FromTable, relationInput.Database)
	}

	toTable, ok := tablesMap[relationInput.ToTable]

	if !ok {
		return false, fmt.Errorf("table %s doesn't exist for database %s", relationInput.ToTable, relationInput.Database)

	}

	_, ok = toTable.ColumnsMap[relationInput.ToColumn]

	if !ok {
		return false, fmt.Errorf("column %s doesn't exist for table %s of database %s", relationInput.ToColumn, relationInput.ToTable, relationInput.Database)
	}

	return true, nil

}
