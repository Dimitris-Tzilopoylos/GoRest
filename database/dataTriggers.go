package database

import (
	environment "application/environment"
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

var RestDataTrigger string = "REST"
var GraphQLDataTrigger string = "GRAPHQL"

type DataTriggerPayload struct {
	Database  string `json:"database"`
	Table     string `json:"table"`
	Data      any    `json:"data"`
	Operation string `json:"operation"`
}

type DataTriggerInput struct {
	Database  string
	Table     string
	Type      string
	Operation string
	Auth      string
	Payload   any
}

type TriggerConfig struct {
	RestEnabled    bool
	GraphQLEnabled bool
	InsertEnabled  bool
	UpdateEnabled  bool
	DeleteEnabled  bool
	ErrorEnabled   bool
}

type DataTrigger struct {
	Id            string
	CreatedAt     string
	Database      string
	Table         string
	TriggerConfig TriggerConfig
}

func ParseTriggerConfigBooleanValue(key string, value map[string]interface{}) bool {
	x, ok := value[key]
	if !ok {
		return false
	}

	booleanValue, ok := x.(bool)
	if !ok {
		return false
	}

	return booleanValue
}

func (enigne *Engine) LoadDataTriggers(db *sql.DB) map[string]map[string]DataTrigger {
	scanner := Query(db, ENGINE_GET_DATA_TRIGGERS)
	dataTriggers := make(map[string]map[string]DataTrigger)
	callback := func(rows *sql.Rows) error {
		var row DataTrigger
		var trigger_config_str string
		err := rows.Scan(&row.Id, &row.CreatedAt, &row.Database, &row.Table, &trigger_config_str)
		if err != nil {
			return err
		}
		var trigger_interface any
		err = json.Unmarshal([]byte(trigger_config_str), &trigger_interface)
		if err != nil {
			return err
		}
		parsedTriggerInterface, err := IsMapToInterface(trigger_interface)
		if err != nil {
			return err
		}
		row.TriggerConfig.InsertEnabled = ParseTriggerConfigBooleanValue("insert", parsedTriggerInterface)
		row.TriggerConfig.UpdateEnabled = ParseTriggerConfigBooleanValue("update", parsedTriggerInterface)
		row.TriggerConfig.DeleteEnabled = ParseTriggerConfigBooleanValue("delete", parsedTriggerInterface)
		row.TriggerConfig.RestEnabled = ParseTriggerConfigBooleanValue("rest", parsedTriggerInterface)
		row.TriggerConfig.GraphQLEnabled = ParseTriggerConfigBooleanValue("graphql", parsedTriggerInterface)
		row.TriggerConfig.ErrorEnabled = ParseTriggerConfigBooleanValue("error", parsedTriggerInterface)

		if err != nil {
			return err
		}
		if _, ok := dataTriggers[row.Database]; !ok {
			dataTriggers[row.Database] = make(map[string]DataTrigger)
		}
		dataTriggers[row.Database][row.Table] = row
		return err
	}
	err := scanner(callback)
	if err != nil {
		panic(err)
	}
	enigne.DataTriggers = dataTriggers
	return dataTriggers
}

func (engine *Engine) GetDatabaseDataTriggers(database string) (map[string]DataTrigger, error) {
	tableTriggers, ok := engine.DataTriggers[database]
	if !ok {
		return nil, fmt.Errorf("there are no data triggers for database: %s", database)
	}

	return tableTriggers, nil
}

func (engine *Engine) GetDatabaseTableDataTrigger(database string, table string) (*DataTrigger, error) {
	tableTriggers, err := engine.GetDatabaseDataTriggers(database)
	if err != nil {
		return nil, err
	}

	dataTirgger, ok := tableTriggers[table]

	if !ok {
		return nil, fmt.Errorf("no data trigger for database: %s and table: %s", database, table)
	}

	return &dataTirgger, nil
}

func (engine *Engine) PostEvent(input DataTriggerInput) {

	payload := DataTriggerPayload{
		Database:  input.Database,
		Table:     input.Table,
		Data:      input.Payload,
		Operation: strings.ToLower(input.Operation),
	}

	jsonBody, err := json.Marshal(payload)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	var WEBSOCKET_SERVICE string = environment.GetEnvValue("WEBSOCKET_SERVICE")

	url := fmt.Sprintf("%s/data-trigger", WEBSOCKET_SERVICE)
	fmt.Println(url)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", input.Auth))
	req.Header.Set("X-Api-key", environment.GetEnvValue("DATA_TRIGGER_SERVICE_API_KEY"))

	client := http.Client{}
	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	defer res.Body.Close()
}

func (engine *Engine) ExecuteDataTrigger(dataTriggerInput DataTriggerInput) {
	trigger, err := engine.GetDatabaseTableDataTrigger(dataTriggerInput.Database, dataTriggerInput.Table)
	if err != nil {
		return
	}

	if dataTriggerInput.Type == RestDataTrigger {
		if !trigger.TriggerConfig.RestEnabled {
			return
		}
	}

	if dataTriggerInput.Type == GraphQLDataTrigger {
		if !trigger.TriggerConfig.GraphQLEnabled {
			return
		}
	}

	switch dataTriggerInput.Operation {
	case INSERT_OPERATION:
		if !trigger.TriggerConfig.InsertEnabled {
			return
		}
		engine.PostEvent(dataTriggerInput)
	case UPDATE_OPERATION:
		if !trigger.TriggerConfig.UpdateEnabled {
			return
		}
		engine.PostEvent(dataTriggerInput)
	case DELETE_OPERATION:
		if !trigger.TriggerConfig.DeleteEnabled {
			return
		}
		engine.PostEvent(dataTriggerInput)
	case ERROR_OPERATION:
		if !trigger.TriggerConfig.ErrorEnabled {
			return
		}
		engine.PostEvent(dataTriggerInput)
	}

}
