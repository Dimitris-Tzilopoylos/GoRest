package environment

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

func LoadEnv() {
	err := godotenv.Load(".env")
	if err != nil {
		panic(err)
	}
}

func GetEnvValue(key string) string {
	return os.Getenv(key)
}

func GetEnvValueToIntWithDefault(key string, defaultVal int) int {
	value := os.Getenv(key)

	x, err := strconv.Atoi(value)
	if err != nil {
		return defaultVal
	}

	return x
}

func GetEnvValueToStringWithDefault(key string, defaultVal string) string {
	value := os.Getenv(key)

	if len(value) == 0 {
		return defaultVal
	}

	return value
}
