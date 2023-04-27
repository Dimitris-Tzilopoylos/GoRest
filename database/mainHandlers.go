package database

import (
	"context"
	"database/sql"
	"fmt"
)

func (e *Engine) SelectExec(role string, db *sql.DB, database string, body interface{}) ([]byte, error) {

	databaseExists := e.DatabaseExists(database)

	if !databaseExists {
		return nil, fmt.Errorf("database %s doesn't exist", database)
	}

	args, err := IsMapToInterface(body)
	if err != nil {
		return []byte{}, err
	}

	var response []byte
	response = append(response, '{')
	i := 0
	for key, modelBody := range args {
		if i > 0 {
			response = append(response, ',')
		}
		model, err := e.GetModelByKey(database, key)

		if err != nil {
			return []byte{}, err
		}
		idx := 1
		query := ""
		args := make([]interface{}, 0)
		if IsAggregation(key) {
			queryString, newArgs := (*model).SelectAggregate(role, modelBody, 0, &idx, nil, fmt.Sprintf("_0_%s", key), key)
			query = queryString
			args = append(args, newArgs...)
		} else {
			queryString, newArgs := (*model).Select(role, modelBody, 0, &idx, nil, fmt.Sprintf("_0_%s", key))
			query = queryString
			args = append(args, newArgs...)
		}

		scanner := Query(db, query, args...)
		var r []byte
		cb := func(rows *sql.Rows) error {
			err := rows.Scan(&r)
			return err
		}
		err = scanner(cb)
		response = append(response, []byte(fmt.Sprintf("\"%s\":", key))...)
		response = append(response, r...)
		if err != nil {
			return []byte{}, err
		}
		i += 1
	}
	response = append(response, '}')
	return response, nil
}

func (e *Engine) InsertExec(role string, db *sql.DB, database string, body interface{}) (interface{}, error) {

	databaseExists := e.DatabaseExists(database)
	if !databaseExists {
		return nil, fmt.Errorf("database %s doesn't exist", database)
	}
	args, err := IsMapToInterface(body)

	if err != nil {
		return nil, err
	}
	tx, ctx, err := TransactionQueryStart(db)
	shouldRollback := new(bool)
	*shouldRollback = true
	defer func() {
		if *shouldRollback {
			err := TransactionQueryRollback(tx)
			if err != nil {
				fmt.Println(err)
			}
		}
	}()
	if err != nil {
		return nil, nil
	}

	results, err := e.InsertGo(role, database, ctx, tx, args)

	if err != nil {
		return nil, err
	}
	err = TransactionQueryCommit(tx)

	if err != nil {
		return nil, err
	}

	*shouldRollback = false

	return results, nil
}

func (e *Engine) UpdateExec(role string, db *sql.DB, database string, body interface{}) (interface{}, error) {
	databaseExists := e.DatabaseExists(database)

	if !databaseExists {
		return nil, fmt.Errorf("database %s doesn't exist", database)
	}

	args, err := IsMapToInterface(body)
	if err != nil {
		return nil, err
	}
	tx, ctx, err := TransactionQueryStart(db)
	shouldRollback := new(bool)
	*shouldRollback = true
	defer func() {
		if *shouldRollback {
			err := TransactionQueryRollback(tx)
			if err != nil {
				fmt.Println(err)
			}
		}
	}()
	if err != nil {
		return nil, nil
	}

	results, err := e.UpdateGo(role, database, ctx, tx, args)

	if err != nil {
		return nil, err
	}

	err = TransactionQueryCommit(tx)

	if err != nil {
		return nil, err
	}

	*shouldRollback = false

	return results, nil
}

func (e *Engine) DeleteExec(role string, db *sql.DB, database string, body interface{}) (interface{}, error) {
	databaseExists := e.DatabaseExists(database)

	if !databaseExists {
		return nil, fmt.Errorf("database %s doesn't exist", database)
	}
	args, err := IsMapToInterface(body)
	if err != nil {
		return nil, err
	}
	tx, ctx, err := TransactionQueryStart(db)
	shouldRollback := new(bool)
	*shouldRollback = true
	defer func() {
		if *shouldRollback {
			err := TransactionQueryRollback(tx)
			if err != nil {
				fmt.Println(err)
			}
		}
	}()
	if err != nil {
		return nil, nil
	}
	results, err := e.DeleteGo(role, database, ctx, tx, args)
	if err != nil {
		return nil, err
	}
	err = TransactionQueryCommit(tx)

	if err != nil {
		return nil, err
	}

	*shouldRollback = false

	return results, nil
}

func (e *Engine) Process(role string, db *sql.DB, database string, body interface{}) (interface{}, error) {
	databaseExists := e.DatabaseExists(database)

	if !databaseExists {
		return nil, fmt.Errorf("database %s doesn't exist", database)
	}
	results := make(map[string][]interface{})

	tx, ctx, err := TransactionQueryStart(db)
	shouldRollback := new(bool)
	*shouldRollback = true
	defer func() {
		if *shouldRollback {
			err := TransactionQueryRollback(tx)
			if err != nil {
				fmt.Println(err)
			}
		}
	}()
	if err != nil {
		return nil, nil
	}

	parsedBody, err := IsMapToInterface(body)
	if err != nil {
		return nil, fmt.Errorf("invalid input")
	}

	transactionsPayload, ok := parsedBody["transactions"]

	if !ok {
		return nil, fmt.Errorf("transactions key is missing")
	}

	parsedTransactions, err := IsArray(transactionsPayload)

	if err != nil {
		return nil, fmt.Errorf("process many transactions payload should be an array")
	}

	for i, entry := range parsedTransactions {
		parsedEntry, err := IsMapToInterface(entry)
		if err != nil {
			return nil, fmt.Errorf("invalid operation")
		}

		if ProcessEntryIsInsert(parsedEntry) {
			_, ok := results["insert"]
			if !ok {
				results["insert"] = make([]interface{}, 0)
			}
			parsedPayload, err := IsMapToInterface(parsedEntry["insert"])
			if err != nil {
				return nil, fmt.Errorf("invalid input")
			}
			result, err := e.InsertGo(role, database, ctx, tx, parsedPayload)
			if err != nil {
				return nil, fmt.Errorf("insert operation [%d] failed", i)
			}

			results["insert"] = append(results["insert"], result)
			continue
		}

		if ProcessEntryIsUpdate(parsedEntry) {
			_, ok := results["update"]
			if !ok {
				results["update"] = make([]interface{}, 0)
			}
			parsedPayload, err := IsMapToInterface(parsedEntry["update"])
			if err != nil {
				return nil, fmt.Errorf("invalid input")
			}
			result, err := e.UpdateGo(role, database, ctx, tx, parsedPayload)
			if err != nil {
				return nil, fmt.Errorf("update operation [%d] failed", i)
			}

			results["update"] = append(results["update"], result)
			continue
		}

		if ProcessEntryIsDelete(parsedEntry) {
			_, ok := results["delete"]
			if !ok {
				results["delete"] = make([]interface{}, 0)
			}
			parsedPayload, err := IsMapToInterface(parsedEntry["delete"])
			if err != nil {
				return nil, fmt.Errorf("invalid input")
			}
			result, err := e.DeleteGo(role, database, ctx, tx, parsedPayload)
			if err != nil {
				return nil, fmt.Errorf("delete operation [%d] failed", i)
			}

			results["delete"] = append(results["delete"], result)
			continue
		}

		return nil, fmt.Errorf("invalid operation")
	}

	err = TransactionQueryCommit(tx)

	if err != nil {
		return nil, err
	}

	*shouldRollback = false

	return results, nil
}

func (e *Engine) InsertGo(role string, database string, ctx context.Context, tx *sql.Tx, args map[string]interface{}) (interface{}, error) {
	results := make(map[string][]interface{})

	for key, input := range args {
		model, err := e.GetModelByKey(database, key)
		if err != nil {
			return nil, err
		}
		parsedInput, err := IsMapToInterface(input)
		if err != nil {
			return nil, err
		}

		objects, ok := parsedInput["objects"]
		if !ok {
			return nil, fmt.Errorf("no input was found")
		}

		parsedObjects, err := IsArray(objects)
		if err != nil {
			return nil, err
		}
		results[key] = make([]interface{}, 0)
		for _, entry := range parsedObjects {
			result, err := model.Insert(role, ctx, tx, entry)
			if err != nil {
				return nil, err
			}
			results[key] = append(results[key], result)
		}
	}

	return results, nil
}

func (e *Engine) DeleteGo(role string, database string, ctx context.Context, tx *sql.Tx, args map[string]interface{}) (interface{}, error) {
	results := make(map[string][]interface{})
	for key, input := range args {
		model, err := e.GetModelByKey(database, key)
		if err != nil {
			return nil, err
		}
		if err != nil {
			return nil, err
		}

		results[key] = make([]interface{}, 0)
		result, err := model.Delete(role, ctx, tx, input)
		if err != nil {
			return nil, err
		}

		results[key] = result
	}
	return results, nil
}

func (e *Engine) UpdateGo(role string, database string, ctx context.Context, tx *sql.Tx, args map[string]interface{}) (interface{}, error) {
	results := make(map[string][]interface{})
	for key, input := range args {
		model, err := e.GetModelByKey(database, key)
		if err != nil {
			return nil, err
		}
		if err != nil {
			return nil, err
		}

		results[key] = make([]interface{}, 0)
		result, err := model.Update(role, ctx, tx, input)
		if err != nil {
			return nil, err
		}

		results[key] = result
	}
	return results, nil
}
