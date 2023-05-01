package database

import (
	"database/sql"
	"fmt"
)

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
