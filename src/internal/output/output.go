package output

import (
	"encoding/json"
	"fmt"
)

func JSON(data any, pretty bool) (string, error) {
	if pretty {
		b, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			return "", err
		}
		return string(b), nil
	}

	b, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func ErrorMessage(respBody map[string]any) string {
	if respBody == nil {
		return ""
	}
	code, _ := respBody["code"].(string)
	message, _ := respBody["message"].(string)
	if code == "" && message == "" {
		return ""
	}
	if code == "" {
		return message
	}
	if message == "" {
		return code
	}
	return fmt.Sprintf("%s: %s", code, message)
}
