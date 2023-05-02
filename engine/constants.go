package engine

var PROCESS_TRANSACTION_ELIGIBLE_KEYS = map[string]bool{
	"insert": true,
	"update": true,
	"delete": true,
}
