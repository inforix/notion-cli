package output

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"text/template"
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

func Render(data any, pretty bool, formatArg string) (string, error) {
	formatArg = strings.TrimSpace(formatArg)
	if formatArg == "" {
		return JSON(data, pretty)
	}

	tmplText, err := readTemplateArg(formatArg)
	if err != nil {
		return "", err
	}

	tmpl, err := template.New("output").
		Option("missingkey=zero").
		Funcs(template.FuncMap{
			"json":       func(v any) string { return marshalJSON(v, false) },
			"prettyjson": func(v any) string { return marshalJSON(v, true) },
		}).
		Parse(tmplText)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func readTemplateArg(arg string) (string, error) {
	if strings.HasPrefix(arg, "@") {
		path := strings.TrimPrefix(arg, "@")
		if path == "-" {
			data, err := io.ReadAll(os.Stdin)
			if err != nil {
				return "", err
			}
			return string(data), nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		return string(data), nil
	}
	return arg, nil
}

func marshalJSON(value any, pretty bool) string {
	var (
		b   []byte
		err error
	)
	if pretty {
		b, err = json.MarshalIndent(value, "", "  ")
	} else {
		b, err = json.Marshal(value)
	}
	if err != nil {
		return ""
	}
	return string(b)
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
