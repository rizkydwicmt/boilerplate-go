package helper

import (
	"os"
	"strconv"
)

func GetEnv(key string) string {
	return os.Getenv(key)
}

func GetEnvAsInt(name string) int {
	if val, ok := os.LookupEnv(name); ok {
		if intVal, err := strconv.Atoi(val); err == nil {
			return intVal
		}
	}
	return 0
}
