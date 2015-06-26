package compose

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"text/template"
)

func ProcessConfigTemplate(name string, reader io.Reader, vars map[string]interface{}) (*bytes.Buffer, error) {
	var buf bytes.Buffer
	// read template
	data, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("Error reading template %s, error: %s", name, err)
	}
	funcMap := map[string]interface{}{
		"default": fnDefault,
	}
	tmpl, err := template.New(name).Funcs(funcMap).Parse(string(data))
	if err != nil {
		return nil, fmt.Errorf("Error parsing template %s, error: %s", name, err)
	}
	if err := tmpl.Execute(&buf, vars); err != nil {
		return nil, fmt.Errorf("Error executing template %s, error: %s", name, err)
	}
	return &buf, nil
}

func fnDefault(defaultVal interface{}, actualValue ...interface{}) interface{} {
	if len(actualValue) > 0 {
		return actualValue[0]
	} else {
		return defaultVal
	}
}
