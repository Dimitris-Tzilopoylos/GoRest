package database

type OrderedMap struct {
	keys   []string
	values map[string]interface{}
}

func NewOrderedMap() *OrderedMap {
	return &OrderedMap{
		keys:   []string{},
		values: make(map[string]interface{}),
	}
}

func (om *OrderedMap) Set(key string, value interface{}) {
	if _, ok := om.values[key]; !ok {
		om.keys = append(om.keys, key)
	}
	om.values[key] = value
}

func (om *OrderedMap) Get(key string) (interface{}, bool) {
	value, ok := om.values[key]
	return value, ok
}

func (om *OrderedMap) Delete(key string) {
	delete(om.values, key)
	// Remove the key from the slice
	for i, k := range om.keys {
		if k == key {
			om.keys = append(om.keys[:i], om.keys[i+1:]...)
			break
		}
	}
}

func (om *OrderedMap) Iter(callback func(key string, value interface{})) {
	for _, key := range om.keys {
		value := om.values[key]
		callback(key, value)
	}
}

func (om *OrderedMap) GetMap() map[string]any {
	return om.values
}

func (om *OrderedMap) GetKeys() []string {
	return om.keys
}
