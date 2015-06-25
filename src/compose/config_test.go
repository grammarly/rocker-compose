package compose

import (
	// "fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReadConfigFile(t *testing.T) {
	config, err := ReadConfigFile("testdata/compose.yml")
	if err != nil {
		t.Fatal(err)
	}

	// fmt.Printf("config: %q\n", config)

	// TODO: more config assertions
	assert.Equal(t, "patterns", config.Namespace)
	assert.Equal(t, "dockerhub.grammarly.io/patterns:{{patterns_version}}", config.Containers["main"].Image)
	assert.Equal(t, "dockerhub.grammarly.io/patterns-config:{{patterns_config_version}}", config.Containers["config"].Image)
}

func TestConfigExtend(t *testing.T) {
	config, err := ReadConfigFile("testdata/compose.yml")
	if err != nil {
		t.Fatal(err)
	}

	// fmt.Printf("config: %q\n", config)

	// TODO: more config assertions
	assert.Equal(t, "patterns", config.Namespace)
	assert.Equal(t, "dockerhub.grammarly.io/patterns:{{patterns_version}}", config.Containers["main2"].Image)
}
