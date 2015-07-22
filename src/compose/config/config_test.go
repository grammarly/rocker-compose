package config

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	configTestVars = map[string]interface{}{
		"version": map[string]string{
			"patterns": "1.9.2",
		},
	}
)

func TestNewFromFile(t *testing.T) {
	config, err := NewFromFile("testdata/compose.yml", configTestVars, map[string]interface{}{})
	if err != nil {
		t.Fatal(err)
	}

	// fmt.Printf("config: %q\n", config)

	// TODO: more config assertions
	assert.Equal(t, "patterns", config.Namespace)
	assert.Equal(t, "dockerhub.grammarly.io/patterns:1.9.2", *config.Containers["main"].Image)
	assert.Equal(t, "dockerhub.grammarly.io/patterns-config:latest", *config.Containers["config"].Image)
	assert.Equal(t, "container:patterns.main", config.Containers["test"].Net.String())
}

func TestConfigMemoryInt64(t *testing.T) {
	assertions := map[string]int64{
		"-1":   -1,
		"0":    0,
		"100":  100,
		"100x": 100,
		"100b": 100,
		"100k": 102400,
		"100m": 104857600,
		"100g": 107374182400,
	}
	for input, expected := range assertions {
		actual, err := NewConfigMemoryFromString(input)
		if err != nil {
			t.Fatal(err)
		}
		assert.EqualValues(t, expected, *actual)
	}
}

func TestConfigCmdString(t *testing.T) {
	configStr := `namespace: test
containers:
  whoami:
    image: ubuntu
    cmd: whoami`

	config, err := ReadConfig("test", strings.NewReader(configStr), configTestVars, map[string]interface{}{})
	if err != nil {
		t.Fatal(err)
	}

	assert.NotNil(t, config.Containers["whoami"].Cmd)
	assert.Equal(t, Cmd{"/bin/sh", "-c", "whoami"}, config.Containers["whoami"].Cmd)
}
