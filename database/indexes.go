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

type IndexInput struct {
	Database    string        `json:"database"`
	Table       string        `json:"table"`
	Name        string        `json:"name"`
	Columns     []ColumnInput `json:"columns"`
	Type        string        `json:"type"`
	RefDatabase string        `json:"refDatabase"`
	RefTable    string        `json:"refTable"`
	RefColumns  []ColumnInput `json:"refColumns"`
	OnDelete    string        `json:"onDelete"`
	OnUpdate    string        `json:"onUpdate"`
}

func ValidateForeignKeyActions(input IndexInput) bool {
	allowedInputs := []string{"RESTRICT", "CASCADE", "NO ACTION"}
	isUpdateOk := true
	isDeleteOk := true
	if len(input.OnUpdate) > 0 {
		for i := 0; i < len(allowedInputs); i++ {
			if strings.EqualFold(allowedInputs[i], input.OnUpdate) {
				break
			}
			if i == 2 {
				isUpdateOk = false
			}
		}
	}

	if len(input.OnDelete) > 0 {
		for i := 0; i < len(allowedInputs); i++ {
			if strings.EqualFold(allowedInputs[i], input.OnDelete) {
				break
			}
			if i == 2 {
				isDeleteOk = false
			}
		}
	}

	return isDeleteOk && isUpdateOk
}

func MutateForeignKeyActions(index *IndexInput) IndexInput {
	if len(index.OnUpdate) > 0 {
		index.OnUpdate = fmt.Sprintf(" ON UPDATE %s", index.OnUpdate)
	}

	if len(index.OnDelete) > 0 {
		index.OnDelete = fmt.Sprintf(" ON DELETE %s", index.OnDelete)
	}

	return *index
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

	isValid := ValidateForeignKeyActions(index)
	if !isValid {
		return fmt.Errorf("invalid action configuration for foreign key")
	}
	index = MutateForeignKeyActions(&index)
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
		index.OnUpdate,
		index.OnDelete,
	)

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
		if err != nil {
			errRB := tx.Rollback()
			if errRB != nil {
				return errRB
			}
			return err
		}

		_, err = tx.Exec(alterColumnAutoIncrementQuery)
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
