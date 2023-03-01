package engine

import (
	"io/ioutil"
	"strings"
)

func ReadEnv(fileName string) map[string]string {
	data, err := ioutil.ReadFile(fileName)
	if err != nil {
		panic("NO SUCH FILE")
	}
	fileContent := string(data)
	fileLines := strings.Split(fileContent, "\n")
	variables := map[string]string{}
	for _, line := range fileLines {
		pairs := strings.Split(line, "=")
		key := strings.TrimSpace(pairs[0])
		value := strings.TrimSpace(pairs[1])
		variables[key] = value
	}

	return variables
}
