package database

import (
	"database/sql"
	"fmt"
)

type ColumnInput struct {
	Name          string      `json:"name"`
	Type          string      `json:"type"`
	MaxLength     int64       `json:"maxLength"`
	Nullable      bool        `json:"nullable"`
	DefaultValue  interface{} `json:"defaultValue"`
	AutoIncrement bool        `json:"auto_increment"`
}

func ValidateCreateColumn(engineMap map[string]map[string]*Model, tableInput TableInput) (bool, error) {
	tableMap, ok := engineMap[tableInput.Database]
	if !ok {
		return false, fmt.Errorf("database %s doesn't exist", tableInput.Database)
	}

	model, ok := tableMap[tableInput.Name]

	if !ok {
		return false, fmt.Errorf("table %s of database %s doesn't exist", tableInput.Name, tableInput.Database)
	}

	if len(tableInput.Columns) == 0 {
		return false, fmt.Errorf("no column provided for table %s of  database %s", tableInput.Name, tableInput.Database)
	}

	column := tableInput.Columns[0]

	_, ok = model.ColumnsMap[column.Name]

	if ok {
		return false, fmt.Errorf("column %s already exists for table %s of  database %s", column.Name, tableInput.Name, tableInput.Database)
	}

	return true, nil

}

func ValidateDropColumn(engineMap map[string]map[string]*Model, tableInput TableInput) (bool, error) {
	tableMap, ok := engineMap[tableInput.Database]
	if !ok {
		return false, fmt.Errorf("database %s doesn't exist", tableInput.Database)
	}

	model, ok := tableMap[tableInput.Name]

	if !ok {
		return false, fmt.Errorf("table %s of database %s doesn't exist", tableInput.Name, tableInput.Database)
	}

	if len(tableInput.Columns) == 0 {
		return false, fmt.Errorf("no column provided for table %s of  database %s", tableInput.Name, tableInput.Database)
	}

	column := tableInput.Columns[0]

	_, ok = model.ColumnsMap[column.Name]

	if !ok {
		return false, fmt.Errorf("column %s doesn't exists for table %s of  database %s", column.Name, tableInput.Name, tableInput.Database)
	}

	return true, nil

}

func CreateColumn(db *sql.DB, tableInput TableInput) error {

	if len(tableInput.Columns) == 0 {
		return fmt.Errorf("no column provided for table %s of  database %s", tableInput.Name, tableInput.Database)
	}

	column := tableInput.Columns[0]

	columnDefinition, err := GetColumnFromInputToSqlString(column)
	if err != nil {
		return err
	}

	query := fmt.Sprintf(`ALTER TABLE %s.%s ADD COLUMN %s;`, tableInput.Database, tableInput.Name, columnDefinition)

	_, err = db.Query(query)

	return err
}

func DropColumn(db *sql.DB, tableInput TableInput) error {
	if len(tableInput.Columns) == 0 {
		return fmt.Errorf("no column provided for table %s of  database %s", tableInput.Name, tableInput.Database)
	}

	column := tableInput.Columns[0]
	query := fmt.Sprintf(`ALTER TABLE %s.%s DROP COLUMN %s;`, tableInput.Database, tableInput.Name, column.Name)

	// SHOULD DROP FOREIGN KEY CONSTRAINTS?

	_, err := db.Query(query)

	return err

}
