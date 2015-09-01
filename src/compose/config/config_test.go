/*-
 * Copyright 2015 Grammarly, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package config

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	configTestVars = map[string]interface{}{
		"version": map[string]string{
			"myapp": "1.9.2",
		},
	}
)

func TestNewFromFile(t *testing.T) {
	config, err := NewFromFile("testdata/compose.yml", configTestVars, map[string]interface{}{})
	if err != nil {
		t.Fatal(err)
	}

	// TODO: more config assertions
	assert.Equal(t, "myapp", config.Namespace)
	assert.Equal(t, "quay.io/myapp:1.9.2", *config.Containers["main"].Image)
	assert.Equal(t, "quay.io/myapp-config:latest", *config.Containers["config"].Image)
	assert.Equal(t, "container:myapp.main", config.Containers["test"].Net.String())
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

func TestNewContainerNameFromString(t *testing.T) {
	type assertion struct {
		namespace string
		name      string
		str       string
	}

	assertions := map[string]assertion{
		"":                     assertion{"", "", ""},
		"nginx":                assertion{"", "nginx", "nginx"},
		".nginx":               assertion{"", "nginx", "nginx"},
		"base.nginx":           assertion{"base", "nginx", "base.nginx"},
		"base.namespace.nginx": assertion{"base.namespace", "nginx", "base.namespace.nginx"},
	}

	for in, out := range assertions {
		t.Logf("Checking container name %q", in)
		cname := NewContainerNameFromString(in)
		assert.Equal(t, out.namespace, cname.Namespace, "Namespace does not match")
		assert.Equal(t, out.name, cname.Name, "Name does not match")
		assert.Equal(t, out.str, cname.String(), "String representation does not match")
	}
}

func TestConfigLinkFromString1(t *testing.T) {
	type assertion struct {
		namespace string
		name      string
		alias     string
		str       string
	}

	assertions := map[string]assertion{
		"nginx":                   assertion{"", "nginx", "nginx", "nginx:nginx"},
		"base.nginx":              assertion{"base", "nginx", "nginx", "base.nginx:nginx"},
		"nginx:balancer":          assertion{"", "nginx", "balancer", "nginx:balancer"},
		"base.nginx:balancer":     assertion{"base", "nginx", "balancer", "base.nginx:balancer"},
		"nginx:www.grammarly.com": assertion{"", "nginx", "www.grammarly.com", "nginx:www.grammarly.com"},
		"nginx_proxy":             assertion{"", "nginx_proxy", "nginx-proxy", "nginx_proxy:nginx-proxy"},
		"nginx:nginx_proxy":       assertion{"", "nginx", "nginx-proxy", "nginx:nginx-proxy"},
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

func TestDockerComposeFormat(t *testing.T) {
	config, err := NewFromFile("testdata/docker-compose.yml", map[string]interface{}{}, map[string]interface{}{})
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "testdata", config.Namespace)
	assert.Equal(t, "postgres:latest", *config.Containers["db"].Image)
}
