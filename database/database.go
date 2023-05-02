package database

import (
	"database/sql"
	"fmt"
)

func GetDatabases(db *sql.DB) ([]string, error) {
	var database string
	databases := []string{}
	scanner := Query(db, GET_DATABASES)
	cb := func(rows *sql.Rows) error {
		err := rows.Scan(&database)
		databases = append(databases, database)
		return err
	}
	err := scanner(cb)
	return databases, err
}

func CreateDataBase(db *sql.DB, database string) error {
	query := fmt.Sprintf(CREATE_DATABASE, database)
	_, err := db.Query(query)
	fmt.Println(query)
	return err
}

func DropDataBase(db *sql.DB, database string) error {
	query := fmt.Sprintf(DROP_DATABASE, database)
	_, err := db.Query(query)
	fmt.Println(query)
	return err
}
