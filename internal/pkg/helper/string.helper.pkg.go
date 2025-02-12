package helper

import "encoding/json"

func StringToStruct[I any](payload string, result *I) error {
	err := json.Unmarshal([]byte(payload), &result)
	if err != nil {
		return err
	}
	return nil
}

func StringToJSON(payload string) (map[string]interface{}, error) {
	var result map[string]interface{}
	err := json.Unmarshal([]byte(payload), &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}
