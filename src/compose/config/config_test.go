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
    image: ubuntu:14.04
    cmd: whoami`

	config, err := ReadConfig("test", strings.NewReader(configStr), configTestVars, map[string]interface{}{})
	if err != nil {
		t.Fatal(err)
	}

	assert.NotNil(t, config.Containers["whoami"].Cmd)
	assert.Equal(t, Cmd{"/bin/sh", "-c", "whoami"}, config.Containers["whoami"].Cmd)
}

func TestConfigNoImageSpecified(t *testing.T) {
	configStr := `namespace: test
containers:
  test:
    cmd: whoami`

	_, err := ReadConfig("test", strings.NewReader(configStr), configTestVars, map[string]interface{}{})
	assert.Equal(t, "Image should be specified for container: test", err.Error())
}

func TestConfigImageNoTag(t *testing.T) {
	configStr := `namespace: test
containers:
  test:
    image: ubuntu`

	_, err := ReadConfig("test", strings.NewReader(configStr), configTestVars, map[string]interface{}{})
	assert.Equal(t, "Image `ubuntu` for container `test`: image without tag is not allowed", err.Error())
}

func TestConfigLinkFromString1(t *testing.T) {
	type assertion struct {
		namespace string
		name      string
		alias     string
		str       string
	}

	assertions := map[string]assertion{
		"nginx":                    assertion{"", "nginx", "nginx", "nginx:nginx"},
		"base.nginx":               assertion{"base", "nginx", "nginx", "base.nginx:nginx"},
		"nginx:balancer":           assertion{"", "nginx", "balancer", "nginx:balancer"},
		"base.nginx:balancer":      assertion{"base", "nginx", "balancer", "base.nginx:balancer"},
		"nginx:capi.grammarly.com": assertion{"", "nginx", "capi.grammarly.com", "nginx:capi.grammarly.com"},
		"nginx_proxy":              assertion{"", "nginx_proxy", "nginx-proxy", "nginx_proxy:nginx-proxy"},
		"nginx:nginx_proxy":        assertion{"", "nginx", "nginx-proxy", "nginx:nginx-proxy"},
	}

	for in, out := range assertions {
		t.Logf("Checking alias %q", in)
		link := NewLinkFromString(in)
		assert.Equal(t, out.namespace, link.Namespace, "Namespace does not match")
		assert.Equal(t, out.name, link.Name, "Name does not match")
		assert.Equal(t, out.alias, link.Alias, "Alias does not match")
		assert.Equal(t, out.str, link.String(), "String representation does not match")
	}
}
