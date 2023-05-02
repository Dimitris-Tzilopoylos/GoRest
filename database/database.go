package database

import (
	"application/environment"
	"database/sql"
	"fmt"
	"strings"
)

func FormatDatabaseName(database string) (string, error) {
	dbname := strings.Trim(database, " ")
	if len(dbname) == 0 {
		return "", fmt.Errorf("database name has 0 length")
	}
	dbname = strings.ToLower(dbname)

	if dbname == environment.GetEnvValue("INTERNAL_SCHEMA_NAME") {
		return "", fmt.Errorf("cannot create database with the same name as the internal schema")
	}

	return dbname, nil

}

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
	LogSql(query)
	return err
}

func DropDataBase(db *sql.DB, database string) error {
	query := fmt.Sprintf(DROP_DATABASE, database)
	_, err := db.Query(query)
	LogSql(query)
	return err
}
