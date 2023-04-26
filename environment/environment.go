package environment

import (
	"os"

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
