package database

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
)

type Webhook struct {
	Id                 string
	Endpoint           string
	Enabled            bool
	Database           string
	Table              string
	Operation          string
	RestEnabled        bool
	GraphQLEnabled     bool
	CreatedAt          string
	Type               string
	ForwardAuthHeaders bool
}

var PRE_EXEC string = "PRE_EXEC"

var POST_EXEC string = "POST_EXEC"

var SELECT_OPERATION string = "SELECT"
var INSERT_OPERATION string = "INSERT"
var UPDATE_OPERATION string = "UPDATE"
var DELETE_OPERATION string = "DELETE"
var ERROR_OPERATION string = "ERROR"

type WebhookExecInput struct {
	Database  string
	Table     string
	Operation string
	Type      string
	Payload   any
	Auth      string
}

type WebhookPayload struct {
	Database string `json:"database"`
	Table    string `json:"table"`
	Data     any    `json:"data"`
}

func (enigne *Engine) LoadWebhooks(db *sql.DB) map[string]map[string]map[string]map[string][]Webhook {
	scanner := Query(db, ENGINE_GET_WEBHOOKS)
	webhooks := make(map[string]map[string]map[string]map[string][]Webhook)
	callback := func(rows *sql.Rows) error {
		var row Webhook
		err := rows.Scan(&row.Id, &row.Endpoint, &row.Enabled, &row.Database, &row.Table, &row.Operation, &row.RestEnabled, &row.GraphQLEnabled, &row.CreatedAt, &row.Type, &row.ForwardAuthHeaders)
		if err != nil {
			return err
		}
		if _, ok := webhooks[row.Database]; !ok {
			webhooks[row.Database] = make(map[string]map[string]map[string][]Webhook)
		}

		if _, ok := webhooks[row.Database][row.Table]; !ok {
			webhooks[row.Database][row.Table] = make(map[string]map[string][]Webhook)
		}

		if _, ok := webhooks[row.Database][row.Table][row.Operation]; !ok {
			webhooks[row.Database][row.Table][row.Operation] = make(map[string][]Webhook)
		}

		if _, ok := webhooks[row.Database][row.Table][row.Operation][row.Type]; !ok {
			webhooks[row.Database][row.Table][row.Operation][row.Type] = make([]Webhook, 0)
		}

		webhooks[row.Database][row.Table][row.Operation][row.Type] = append(webhooks[row.Database][row.Table][row.Operation][row.Type], row)
		return err
	}
	scanner(callback)
	enigne.Webhooks = webhooks
	return webhooks
}

func (engine *Engine) GetDatabaseWebhooksMap(database string) (map[string]map[string]map[string][]Webhook, error) {
	value, ok := engine.Webhooks[database]
	if !ok {
		return nil, fmt.Errorf("there were no webhooks defined for database: %s", database)
	}
	return value, nil
}

func (engine *Engine) GetDatabaseTableWebhooksMap(database string, table string) (map[string]map[string][]Webhook, error) {
	tablesMap, err := engine.GetDatabaseWebhooksMap(database)
	if err != nil {
		return nil, err
	}

	value, ok := tablesMap[table]
	if !ok {
		return nil, fmt.Errorf("there were no webhooks defined for database: %s and table: %s", database, table)
	}
	return value, nil
}

func (engine *Engine) GetDatabaseTableOperationWebhooksMap(database string, table string, operation string) (map[string][]Webhook, error) {
	operationMap, err := engine.GetDatabaseTableWebhooksMap(database, table)
	if err != nil {
		return nil, err
	}

	value, ok := operationMap[operation]
	if !ok {
		return nil, fmt.Errorf("there were no webhooks defined for database: %s and table: %s and operation: %s", database, table, operation)
	}
	return value, nil
}

func (engine *Engine) GetDatabaseTableOperationTypeWebhooks(database, table, operation, webhookType string) ([]Webhook, error) {

	operationsMap, err := engine.GetDatabaseTableOperationWebhooksMap(database, table, operation)
	if err != nil {
		return nil, err
	}

	webhooks, ok := operationsMap[webhookType]

	if !ok {
		return nil, fmt.Errorf("there were no webhooks defined for database: %s and table: %s and operation: %s and type: %s", database, table, operation, webhookType)
	}

	return webhooks, nil

}

func (engine *Engine) ExecuteWebhook(webhook Webhook, input WebhookExecInput) {
	if !webhook.Enabled {
		return
	}

	payload := WebhookPayload{
		Database: input.Database,
		Table:    input.Table,
		Data:     input.Payload,
	}

	jsonBody, err := json.Marshal(payload)
	if err != nil {
		fmt.Println(err)
		return
	}

	req, err := http.NewRequest("POST", webhook.Endpoint, bytes.NewBuffer(jsonBody))
	if err != nil {
		fmt.Println(err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	if webhook.ForwardAuthHeaders {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", input.Auth))
	}

	client := http.Client{}
	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer res.Body.Close()

}

func (engine *Engine) ExecuteWebhooks(input WebhookExecInput) {
	webhooks, err := engine.GetDatabaseTableOperationTypeWebhooks(input.Database, input.Table, input.Operation, input.Type)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	for _, webhook := range webhooks {

		go engine.ExecuteWebhook(webhook, input)
	}
}
