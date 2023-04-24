package database

import (
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
	results := make(map[string][]interface{})
	args, err := IsMapToInterface(body)
	if err != nil {
		return []byte{}, err
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

	for key, input := range args {
		model, err := e.GetModelByKey(database, key)
		if err != nil {
			return []byte{}, err
		}
		parsedInput, err := IsMapToInterface(input)
		if err != nil {
			return []byte{}, err
		}

		objects, ok := parsedInput["objects"]
		if !ok {
			return []byte{}, fmt.Errorf("no input was found")
		}

		parsedObjects, err := IsArray(objects)
		if err != nil {
			return []byte{}, err
		}
		results[key] = make([]interface{}, 0)
		for _, entry := range parsedObjects {
			result, err := model.Insert(role, ctx, tx, entry)
			if err != nil {
				return []byte{}, err
			}
			results[key] = append(results[key], result)
		}
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
	results := make(map[string][]interface{})
	args, err := IsMapToInterface(body)
	if err != nil {
		return []byte{}, err
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
	for key, input := range args {
		model, err := e.GetModelByKey(database, key)
		if err != nil {
			return []byte{}, err
		}
		if err != nil {
			return []byte{}, err
		}

		results[key] = make([]interface{}, 0)
		result, err := model.Update(role, ctx, tx, input)
		if err != nil {
			return []byte{}, err
		}

		results[key] = result
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
	results := make(map[string][]interface{})
	args, err := IsMapToInterface(body)
	if err != nil {
		return []byte{}, err
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
	for key, input := range args {
		model, err := e.GetModelByKey(database, key)
		if err != nil {
			return []byte{}, err
		}
		if err != nil {
			return []byte{}, err
		}

		results[key] = make([]interface{}, 0)
		result, err := model.Delete(role, ctx, tx, input)
		if err != nil {
			return []byte{}, err
		}

		results[key] = result
	}

	err = TransactionQueryCommit(tx)

	if err != nil {
		return nil, err
	}

	*shouldRollback = false

	return results, nil
}
