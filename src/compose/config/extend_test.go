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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfigExtend(t *testing.T) {
	config, err := NewFromFile("testdata/compose.yml", configTestVars, map[string]interface{}{}, false)
	if err != nil {
		t.Fatal(err)
	}

	// TODO: more config assertions
	assert.Equal(t, "myapp", config.Namespace)
	assert.Equal(t, "quay.io/myapp:1.9.2", *config.Containers["main2"].Image)

	// should be inherited
	assert.Equal(t, Strings{"8.8.8.8"}, config.Containers["main2"].DNS)
	// should be overriden
	assert.Equal(t, Strings{"www.grammarly.com:127.0.0.2"}, config.Containers["main2"].AddHost)

	// should be inherited
	assert.EqualValues(t, 512, *config.Containers["main2"].CPUShares)

	// should inherit and merge labels
	assert.Equal(t, 3, len(config.Containers["main2"].Labels))
	assert.Equal(t, "myapp", config.Containers["main2"].Labels["service"])
	assert.Equal(t, "2", config.Containers["main2"].Labels["num"])
	assert.Equal(t, "replica", config.Containers["main2"].Labels["type"])

	// should not affect parent labels
	assert.Equal(t, 2, len(config.Containers["main"].Labels))
	assert.Equal(t, "myapp", config.Containers["main"].Labels["service"])
	assert.Equal(t, "1", config.Containers["main"].Labels["num"])

	// should be overriden
	assert.EqualValues(t, 200, *config.Containers["main2"].KillTimeout)
}
