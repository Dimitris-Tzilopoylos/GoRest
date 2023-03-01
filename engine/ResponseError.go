package engine

func ResponseErrorMessage(msg interface{}) map[string]interface{} {
	return map[string]interface{}{
		"error": msg,
	}
}
