package database

import "github.com/google/uuid"

type Listener struct {
	id      string
	handler func(args ...any)
}

type EventEmitter struct {
	EventMap map[string]map[string]Listener
}

func NewEventEmitter() *EventEmitter {
	return &EventEmitter{
		EventMap: make(map[string]map[string]Listener),
	}
}

func (emitter *EventEmitter) MakeUniqueListenerId() string {
	return uuid.New().String()
}

func (emitter *EventEmitter) EventNameSpaceExists(eventName string) bool {
	_, ok := emitter.EventMap[eventName]
	return ok
}

func (emitter *EventEmitter) EventNameSpaceListenerExists(eventName string, listenerId string) bool {
	if emitter.EventNameSpaceExists(eventName) {
		_, ok := emitter.EventMap[eventName][listenerId]
		return ok
	}

	return false
}

func (emitter *EventEmitter) AddListener(eventName string, listener Listener) {

}

func (emitter *EventEmitter) RemoveListener(eventName, listenerId string) {
	if emitter.EventNameSpaceListenerExists(eventName, listenerId) {
		delete(emitter.EventMap[eventName], listenerId)
	}
}

func (emitter *EventEmitter) RemoveAllListeners(eventName string) {
	if emitter.EventNameSpaceExists(eventName) {
		delete(emitter.EventMap, eventName)
	}
}

func (emitter *EventEmitter) Emit(eventName string, args ...any) {
	if emitter.EventNameSpaceExists(eventName) {
		listeners := emitter.EventMap[eventName]
		if len(listeners) == 0 {
			return
		}

		for _, listener := range listeners {
			listener.handler(args...)
		}
	}
}

func (emitter *EventEmitter) Subscribe(eventName string, listener func(args ...any)) func() {
	if _, ok := emitter.EventMap[eventName]; !ok {
		emitter.EventMap[eventName] = make(map[string]Listener)
	}

	listenerId := emitter.MakeUniqueListenerId()

	emitter.EventMap[eventName][listenerId] = Listener{
		id:      listenerId,
		handler: listener,
	}

	return func() {
		emitter.RemoveListener(eventName, listenerId)
	}
}
