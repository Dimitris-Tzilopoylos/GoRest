package database

import (
	"database/sql"
	"fmt"
)

func CreateDataBase(db *sql.DB, database string) error {
	_, err := db.Query(fmt.Sprintf(CREATE_DATABASE, database))

	return err
}

func DropDataBase(db *sql.DB, database string) error {
	_, err := db.Query(fmt.Sprintf(DROP_DATABASE, database))

	return err
}
