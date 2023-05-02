package database

import (
	"database/sql"
	"fmt"
	"sort"
	"strings"
)

var FOREIGN string = "FOREIGN"
var UNIQUE string = "UNIQUE"
var PRIMARY string = "PRIMARY"

type IndexType string

type Index struct {
	Name            string    `json:"name"`
	Type            IndexType `json:"type"`
	Table           string    `json:"table"`
	Column          string    `json:"column"`
	ReferenceTable  string    `json:"reference_table"`
	ReferenceColumn string    `json:"reference_column"`
	Database        string    `json:"database"`
}

type ColumnInput struct {
	Name          string
	Type          string
	MaxLength     int64
	Nullable      bool
	DefaultValue  interface{}
	AutoIncrement bool
}

type IndexInput struct {
	Name        string
	Columns     []ColumnInput
	Type        string
	RefDatabase string
	RefTable    string
	RefColumns  []ColumnInput
	OnDelete    string
	OnUpdate    string
}

type TableInput struct {
	Database string
	Name     string
	Columns  []ColumnInput
	Indexes  []IndexInput
}

type Column struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	MaxLength    int64  `json:"max_length"`
	Nullable     bool   `json:"nullable"`
	DefaultValue string `json:"default_value"`
}

func GetTableNames(db *sql.DB, database string) ([]string, error) {
	var table string
	tables := []string{}
	scanner := Query(db, GET_DATABASE_TABLES, database)
	cb := func(rows *sql.Rows) error {
		err := rows.Scan(&table)
		tables = append(tables, table)
		return err
	}
	err := scanner(cb)

	return tables, err
}

func GetTableColumns(db *sql.DB, database string, table string) ([]Column, error) {
	columns := []Column{}

	scanner := Query(db, GET_DATABASE_TABLE_COLUMN, database, table)
	cb := func(rows *sql.Rows) error {
		var column Column
		var maxLength sql.NullInt64
		var defaultValue sql.NullString
		err := rows.Scan(&column.Name, &column.Type, &maxLength, &column.Nullable, &defaultValue)
		if err != nil {
			panic(err)
		}
		if maxLength.Valid {
			column.MaxLength = maxLength.Int64
		}
		if defaultValue.Valid {
			column.DefaultValue = defaultValue.String
		}
		columns = append(columns, column)
		return err
	}
	err := scanner(cb)

	return columns, err
}

func GetTableIndexes(db *sql.DB, database string, table string) ([]Index, error) {

	indexes := []Index{}

	scanner := Query(db, GET_DATABASE_TABLE_INDEXES, database, table)
	cb := func(rows *sql.Rows) error {
		var index Index
		err := rows.Scan(&index.Name, &index.Table, &index.Column, &index.ReferenceTable, &index.ReferenceColumn, &index.Type)
		if err != nil {
			panic(err)
		}

		indexes = append(indexes, index)
		return err
	}
	err := scanner(cb)

	scannerUnique := Query(db, GET_UNIQUE_INDXES, database, table)
	unqCB := func(rows *sql.Rows) error {
		var index Index
		var unq bool = false
		err = rows.Scan(&index.Database, &index.Table, &index.Name, &unq, &index.Column)
		if err != nil {
			panic(err)
		}

		if unq {
			columns := strings.Split(index.Column, ",")
			for _, col := range columns {
				colName := strings.Trim(col, " ")
				newIndex := Index{
					Name:   index.Name,
					Type:   "UNIQUE KEY",
					Table:  index.Table,
					Column: colName,
				}
				indexes = append(indexes, newIndex)
			}
		}

		return err
	}

	err = scannerUnique(unqCB)

	return indexes, err
}

func GetColumnFromInputToSqlString(column ColumnInput) (string, error) {
	str := fmt.Sprintf("%s %s", column.Name, column.Type)
	if column.MaxLength != 0 {
		str += fmt.Sprintf("(%d)", column.MaxLength)
	}

	if !column.Nullable {
		str += " NOT NULL "
	}

	if column.DefaultValue != nil {
		str += fmt.Sprintf(` DEFAULT %v `, column.DefaultValue)

	}
	return str, nil
}

func CreateTable(db *sql.DB, table TableInput) error {
	columnParts := make([]string, 0)

	if len(table.Database) == 0 || len(table.Name) == 0 {
		return fmt.Errorf("provide database name and table name")
	}

	for _, col := range table.Columns {
		column, err := GetColumnFromInputToSqlString(col)
		if err != nil {
			return err
		}
		columnParts = append(columnParts, column)
	}

	if len(columnParts) == 0 {
		return fmt.Errorf("no columns were provided")
	}

	query := fmt.Sprintf(CREATE_TABLE, table.Database, table.Name, strings.Join(columnParts, ","))

	_, err := db.Query(query)
	LogSql(query)

	return err
}

func DropTable(db *sql.DB, table TableInput) error {
	query := fmt.Sprintf(DROP_TABLE, table.Database, table.Name)
	_, err := db.Query(query)
	LogSql(query)

	return err
}

func DropIndex(db *sql.DB, indexName string) error {
	query := fmt.Sprintf(DROP_INDEX, indexName)
	_, err := db.Query(query)
	LogSql(query)

	return err
}

func GetColumnNamesInStringArr(columns []ColumnInput) []string {
	colNames := make([]string, 0)
	for _, col := range columns {
		colNames = append(colNames, col.Name)
	}
	return colNames
}

func CreateIndexName(prefix string, table TableInput, columns []ColumnInput) string {
	colNames := GetColumnNamesInStringArr(columns)
	sort.Strings(colNames)

	return fmt.Sprintf("%s_%s_%s_%s", prefix, table.Database, table.Name, strings.Join(colNames, "_"))

}

func CreateUniqueIndex(db *sql.DB, table TableInput, index IndexInput) error {
	columnParts := GetColumnNamesInStringArr(index.Columns)

	if len(columnParts) == 0 {
		return fmt.Errorf("no columns were provided")
	}

	indexName := CreateIndexName("unique_idx", table, index.Columns)

	query := fmt.Sprintf(CREATE_UNIQUE_INDEX, indexName, table.Database, table.Name, strings.Join(columnParts, ","))

	_, err := db.Query(query)
	if err != nil {
		LogSql(err.Error())
	}
	LogSql(query)

	return nil
}

func CreateForeignIndex(db *sql.DB, table TableInput, index IndexInput) error {

	if len(index.Columns) == 0 || len(index.Columns) != len(index.RefColumns) {
		return fmt.Errorf("invalid configuration for foreign key")
	}

	columnNames := GetColumnNamesInStringArr(index.Columns)
	refColumnNames := GetColumnNamesInStringArr(index.RefColumns)

	indexName := CreateIndexName("foreign_idx", table, index.Columns)

	query := fmt.Sprintf(CREATE_FOREIGN_INDEX,
		table.Database,
		table.Name,
		indexName,
		strings.Join(columnNames, ","),
		index.RefDatabase,
		index.RefTable,
		strings.Join(refColumnNames, ","),
	)

	if len(index.OnDelete) > 0 {
		query += fmt.Sprintf(" ON DELETE %s", index.OnDelete)
	}

	if len(index.OnUpdate) > 0 {
		query += fmt.Sprintf(" ON UPDATE %s", index.OnUpdate)
	}

	_, err := db.Query(query)
	LogSql(query)

	return err
}

func GetSequenceName(table TableInput, column ColumnInput) string {
	return fmt.Sprintf("%s.%s_%s", table.Database, table.Name, column.Name)
}

func GetAutoIncrementSequenceQuery(table TableInput, columns []ColumnInput) (string, error) {
	for _, column := range columns {
		if column.AutoIncrement {
			return fmt.Sprintf(CREATE_SEQUENCE, GetSequenceName(table, column)), nil

		}
	}

	return "", fmt.Errorf("no auto increment column was found")
}

func GetAlterColumnSetToSequenceQuery(table TableInput, columns []ColumnInput) (string, error) {
	for _, column := range columns {
		if column.AutoIncrement {
			return fmt.Sprintf(AUTO_INCREMENT_COLUMN, table.Database, table.Name, column.Name, GetSequenceName(table, column)), nil

		}
	}

	return "", fmt.Errorf("no auto increment column was found")
}

func CreatePrimaryIndex(db *sql.DB, table TableInput, index IndexInput) error {

	columnParts := GetColumnNamesInStringArr(index.Columns)

	if len(columnParts) == 0 {
		return fmt.Errorf("no columns were provided")
	}

	existingIndexes, err := GetTableIndexes(db, table.Database, table.Name)

	if err == nil && len(existingIndexes) > 0 {
		for _, tableIndex := range existingIndexes {
			if tableIndex.Type == "PRIMARY KEY" {
				LogSql("primary key index already exists for table: " + tableIndex.Table)
				return nil

			}
		}
	}

	sequenceQuery, _ := GetAutoIncrementSequenceQuery(table, index.Columns)
	alterColumnAutoIncrementQuery, _ := GetAlterColumnSetToSequenceQuery(table, index.Columns)

	tx, err := db.Begin()
	if err != nil {
		errRB := tx.Rollback()
		if errRB != nil {
			return errRB
		}
		return err
	}

	if len(sequenceQuery) > 0 && len(alterColumnAutoIncrementQuery) > 0 {
		_, err = tx.Exec(sequenceQuery)
		fmt.Println(sequenceQuery)
		if err != nil {
			errRB := tx.Rollback()
			if errRB != nil {
				return errRB
			}
			return err
		}

		_, err = tx.Exec(alterColumnAutoIncrementQuery)
		fmt.Println(alterColumnAutoIncrementQuery)
		if err != nil {
			errRB := tx.Rollback()
			if errRB != nil {
				return errRB
			}
			return err
		}
	}

	indexName := CreateIndexName("primary_idx", table, index.Columns)
	query := fmt.Sprintf(CREATE_PRIMARY_INDEX, table.Database, table.Name, indexName, strings.Join(columnParts, ","))

	_, err = tx.Exec(query)
	LogSql(query)
	if err != nil {
		errRB := tx.Rollback()
		if errRB != nil {
			return errRB
		}
		return err
	}

	err = tx.Commit()
	if err != nil {
		errRB := tx.Rollback()
		if errRB != nil {
			return errRB
		}
		return err
	}

	return err

}

func CreateIndex(db *sql.DB, table TableInput, index IndexInput) error {
	var err error

	switch index.Type {
	case UNIQUE:
		err = CreateUniqueIndex(db, table, index)
	case FOREIGN:
		err = CreateForeignIndex(db, table, index)
	case PRIMARY:
		err = CreatePrimaryIndex(db, table, index)
	default:
		break
	}

	return err
}

func CreateIndexes(db *sql.DB, table TableInput) error {
	if len(table.Indexes) == 0 {
		return nil
	}
	for _, index := range table.Indexes {
		err := CreateIndex(db, table, index)
		if err != nil {
			LogSql(err.Error())
			return err
		}
	}

	return nil
}
