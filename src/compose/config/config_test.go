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

	"github.com/grammarly/rocker/src/template"
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
	config, err := NewFromFile("testdata/compose.yml", configTestVars, map[string]interface{}{}, false)
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

	config, err := ReadConfig("test", strings.NewReader(configStr), configTestVars, map[string]interface{}{}, false)
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

	_, err := ReadConfig("test", strings.NewReader(configStr), configTestVars, map[string]interface{}{}, false)
	assert.Equal(t, "Image should be specified for container: test", err.Error())
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
		".nginx":               assertion{".", "nginx", "nginx"},
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
		".nginx":                  assertion{"", "nginx", "nginx", "nginx:nginx"},
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
		assert.Equal(t, out.namespace, link.GetNamespace(), "Namespace does not match")
		assert.Equal(t, out.name, link.ContainerName.Name, "Name does not match")
		assert.Equal(t, out.alias, link.Alias, "Alias does not match")
		assert.Equal(t, out.str, link.String(), "String representation does not match")
	}
}

func TestDockerComposeFormat(t *testing.T) {
	config, err := NewFromFile("testdata/docker-compose.yml", map[string]interface{}{}, map[string]interface{}{}, false)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "testdata", config.Namespace)
	assert.Equal(t, "postgres:latest", *config.Containers["db"].Image)
}

func TestContainerNameGlobalNs(t *testing.T) {
	// redis: ns="" name="redis"
	// .redis: ns="." name="redis"
	// g.redis: ns="g" name="redis"

	c1 := NewContainerNameFromString("redis")
	c2 := NewContainerNameFromString(".redis")
	c3 := NewContainerNameFromString("g.redis")

	assert.True(t, c2.IsEqualTo(c1))

	c1.DefaultNamespace("test")
	c2.DefaultNamespace("test")
	c3.DefaultNamespace("test")

	assert.Equal(t, "test.redis", c1.String())
	assert.Equal(t, "redis", c2.String())
	assert.Equal(t, "g.redis", c3.String())
}

func TestLinkGlobalNs(t *testing.T) {
	// redis: ns="" name="redis"
	// .redis: ns="." name="redis"
	// g.redis: ns="g" name="redis"

	c1 := NewLinkFromString("redis")
	c2 := NewLinkFromString(".redis")
	c3 := NewLinkFromString("g.redis")

	c1.DefaultNamespace("test")
	c2.DefaultNamespace("test")
	c3.DefaultNamespace("test")

	assert.Equal(t, "test.redis:redis", c1.String())
	assert.Equal(t, "redis:redis", c2.String())
	assert.Equal(t, "g.redis:redis", c3.String())
}

func TestConfigHasExternalRefs(t *testing.T) {
	assertions := map[string]bool{
		`namespace: test
containers:
  main:
    image: redis:latest
    volumes_from: redis_data`: false,

		`namespace: test
containers:
  main:
    image: web:latest
    links: .redis`: true,

		`namespace: test
containers:
  main:
    image: web:latest
    links: redis.main`: true,
	}

	for in, out := range assertions {
		t.Logf("Checking config %q", in)
		cfg, err := ReadConfig("test", strings.NewReader(in), template.Vars{}, map[string]interface{}{}, false)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, out, cfg.HasExternalRefs())
	}
}
