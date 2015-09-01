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
	"encoding/json"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	configConvertTestVars = map[string]interface{}{
		"version": map[string]string{
			"myapp": "1.9.2",
		},
	}
)

func TestConfigGetApiConfig(t *testing.T) {
	config, err := NewFromFile("testdata/compose.yml", configTestVars, map[string]interface{}{})
	if err != nil {
		t.Fatal(err)
	}

	expected, err := ioutil.ReadFile("testdata/container_main_config.json")
	if err != nil {
		t.Fatal(err)
	}

	actual, err := json.Marshal(config.Containers["main"].GetApiConfig())
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, strings.TrimSpace(string(expected)), string(actual))
}

func TestConfigGetApiHostConfig(t *testing.T) {
	config, err := NewFromFile("testdata/compose.yml", configTestVars, map[string]interface{}{})
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
