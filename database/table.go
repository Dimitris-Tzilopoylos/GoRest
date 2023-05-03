package database

import (
	"database/sql"
	"fmt"
	"strings"
)

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
