package compose

import (
	"encoding/json"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	configConvertTestVars = map[string]interface{}{
		"version": map[string]string{
			"patterns": "1.9.2",
		},
	}
)

func TestConfigGetApiConfig(t *testing.T) {
	// a := (int64)(512)
	// c := &ConfigContainer{Hostname: "pattern1", CpuShares: &a}

	config, err := ReadConfigFile("testdata/compose.yml", configTestVars)
	if err != nil {
		t.Fatal(err)
	}

	expected, err := ioutil.ReadFile("testdata/container_main_config.json")
	if err != nil {
		t.Fatal(err)
	}

	// assert.Equal(t, "pattern1", config.Containers["main"].GetApiConfig().Hostname)

	actual, err := json.Marshal(config.Containers["main"].GetApiConfig())
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, strings.TrimSpace(string(expected)), string(actual))
}

func TestConfigGetApiHostConfig(t *testing.T) {
	// a := (int64)(512)
	// c := &ConfigContainer{Hostname: "pattern1", CpuShares: &a}

	config, err := ReadConfigFile("testdata/compose.yml", configTestVars)
	if err != nil {
		t.Fatal(err)
	}

	expected, err := ioutil.ReadFile("testdata/container_main_host_config.json")
	if err != nil {
		t.Fatal(err)
	}

	actual, err := json.Marshal(config.Containers["main"].GetApiHostConfig())
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, strings.TrimSpace(string(expected)), string(actual))
}
