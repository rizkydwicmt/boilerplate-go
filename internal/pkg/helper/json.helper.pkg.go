package helper

import (
	"encoding/json"
	"fmt"
)

func JSONToString(payload any) (string, error) {
	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		fmt.Println("Error:", err)
		return "", err
	}

	jsonString := string(jsonBytes)
	return jsonString, nil
}

func JSONToStruct[I any](payload any, result *I) error {
	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		fmt.Println("Error:", err)
		return err
	}

	err = json.Unmarshal(jsonBytes, &result)
	if err != nil {
		fmt.Println("Error:", err)
		return err
	}

	return nil
}
