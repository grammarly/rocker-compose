package config

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	configTemplateVars = map[string]interface{}{"mykey": "myval"}
)

func TestProcessConfigTemplate_Basic(t *testing.T) {
	result, err := ProcessConfigTemplate("test", strings.NewReader("this is a test {{.mykey}}"), configTemplateVars, map[string]interface{}{})
	if err != nil {
		t.Fatal(err)
	}
	// fmt.Printf("Template result: %s\n", result)
	assert.Equal(t, "this is a test myval", result.String(), "template should be rendered")
}

func TestProcessConfigTemplate_Default(t *testing.T) {
	// Default when value is present
	result, err := ProcessConfigTemplate("test", strings.NewReader("this is a test {{.mykey | default \"none\"}}"), configTemplateVars, map[string]interface{}{})
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "this is a test myval", result.String(), "template should be rendered")

	// Default when value is undefined
	result2, err := ProcessConfigTemplate("test", strings.NewReader("this is a test {{.mykey2 | default \"none\"}}"), configTemplateVars, map[string]interface{}{})
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "this is a test none", result2.String(), "template should be rendered")
}
