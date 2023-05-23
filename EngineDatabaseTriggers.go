package main

import (
	"application/database"
	engine "application/engine"
	"fmt"
)

func NormalizeEventData(eventData ...any) any {
	if len(eventData) == 1 {
		return eventData[0]
	}
	return eventData
}

func WebsocketEventResponse(app *engine.Router, eventData ...any) {
	for client := range app.Rooms {
		go func(client *engine.Client) {
			err := client.Conn.WriteJSON(NormalizeEventData(eventData))
			if err != nil {
				err = client.Conn.Close()
				if err != nil {
					fmt.Println(err)
				}
			}
		}(client)
	}
}

func RegisterDatabaseTriggers(app *engine.Router) func() {
	unsubscribeInsert := app.Engine.EventEmitter.Subscribe(database.INSERT_OPERATION, func(eventData ...any) {
		go WebsocketEventResponse(app, eventData...)
	})

	unsubscribeUpdate := app.Engine.EventEmitter.Subscribe(database.UPDATE_OPERATION, func(eventData ...any) {
		go WebsocketEventResponse(app, eventData...)
	})

	unsubscribeDelete := app.Engine.EventEmitter.Subscribe(database.DELETE_OPERATION, func(eventData ...any) {
		go WebsocketEventResponse(app, eventData...)
	})

	return func() {
		unsubscribeDelete()
		unsubscribeUpdate()
		unsubscribeInsert()
	}
}
