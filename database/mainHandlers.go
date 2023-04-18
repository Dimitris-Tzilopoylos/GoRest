package database

import (
	"database/sql"
	"fmt"
)

func (e *Engine) SelectExec(db *sql.DB, body interface{}) ([]byte, error) {
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
		model, err := e.GetModelByKey(key)

		if err != nil {
			return []byte{}, err
		}
		idx := 1
		query := ""
		args := make([]interface{}, 0)
		if IsAggregation(key) {
			queryString, newArgs := (*model).SelectAggregate(modelBody, 0, &idx, nil, fmt.Sprintf("_0_%s", key), key)
			query = queryString
			args = append(args, newArgs...)
		} else {
			queryString, newArgs := (*model).Select(modelBody, 0, &idx, nil, fmt.Sprintf("_0_%s", key))
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
