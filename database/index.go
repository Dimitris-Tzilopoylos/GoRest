package database

import (
	"context"
	"database/sql"
	"fmt"
)

func SelectQueryContext(ctx context.Context, db *sql.Conn, query string, args ...any) func(func(rows *sql.Rows) error) error {
	rows, err := db.QueryContext(ctx, query, args...)
	return func(callback func(rows *sql.Rows) error) error {
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			err = callback(rows)
			if err != nil {
				return err
			}
		}

		return err
	}
}

func Query(db *sql.DB, query string, args ...any) func(func(rows *sql.Rows) error) error {
	rows, err := db.Query(query, args...)
	return func(callback func(rows *sql.Rows) error) error {
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			err = callback(rows)
			if err != nil {
				return err
			}
		}

		return err
	}
}

func QueryContext(ctx context.Context, tx *sql.Tx, query string, args ...any) func(func(rows *sql.Rows) error) error {
	rows, err := tx.QueryContext(ctx, query, args...)
	return func(callback func(rows *sql.Rows) error) error {
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			err = callback(rows)
			if err != nil {
				return err
			}
		}

		return err
	}
}

func QueryExecContext(ctx context.Context, tx *sql.Tx, query string, args ...any) func(func(rows sql.Result) error) error {
	x, err := tx.ExecContext(ctx, query, args...)
	return func(callback func(result sql.Result) error) error {
		if err != nil {
			return err
		}
		err := callback(x)
		return err
	}
}

func TransactionQuery(db *sql.DB, callback func(context.Context, *sql.Tx) error) error {
	ctx := context.Background()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	err = callback(ctx, tx)
	if err != nil {
		err = tx.Rollback()
		if err != nil {
			return err
		}
		return err
	}
	err = tx.Commit()
	return err
}

func TransactionQueryStart(db *sql.DB) (*sql.Tx, context.Context, error) {
	ctx := context.Background()
	tx, err := db.BeginTx(ctx, nil)
	return tx, ctx, err
}

func TransactionQueryCommit(tx *sql.Tx) error {
	return tx.Commit()
}

func TransactionQueryRollback(tx *sql.Tx) error {
	return tx.Rollback()
}

func TransactionQueryExec(ctx context.Context, db *sql.DB, callback func(context.Context, *sql.Tx) error) error {

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	err = callback(ctx, tx)
	if err != nil {
		err = tx.Rollback()
		if err != nil {
			return err
		}
		return err
	}
	err = tx.Commit()
	return err
}

func CheckSQLStringValidity(db *sql.DB, query string) error {
	if len(query) == 0 {
		return fmt.Errorf("query was not provided")
	}
	stmt, err := db.Prepare(query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	return nil

}

func (e *Engine) EngineModelsToNotCycledValue() []Model {
	models := []Model{}
	for _, model := range e.Models {
		newModel := *model
		newModel.Relations = nil
		models = append(models, newModel)

	}
	return models
}
