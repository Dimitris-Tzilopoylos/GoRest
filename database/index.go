package database

import (
	"context"
	"database/sql"
)

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
