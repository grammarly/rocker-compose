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

package compose

import (
	"github.com/snkozlov/rocker-compose/src/compose/config"
	"testing"
	"time"

	"github.com/fsouza/go-dockerclient"
	"github.com/stretchr/testify/assert"
)

var (
	containerTestVars = map[string]interface{}{
		"version": map[string]string{
			"myapp": "1.9.2",
		},
	}
)

func TestCreateContainerOptions(t *testing.T) {
	cfg, err := config.NewFromFile("config/testdata/compose.yml", containerTestVars, map[string]interface{}{}, false)
	if err != nil {
		t.Fatal(err)
	}

	container := NewContainerFromConfig(config.NewContainerName("myapp", "main"), cfg.Containers["main"])

	opts, err := container.CreateContainerOptions()
	if err != nil {
		t.Fatal(err)
	}

	assert.IsType(t, &docker.CreateContainerOptions{}, opts)
}

func TestConfigGetContainers(t *testing.T) {
	cfg, err := config.NewFromFile("config/testdata/compose.yml", containerTestVars, map[string]interface{}{}, false)
	if err != nil {
		t.Fatal(err)
	}

	containers := GetContainersFromConfig(cfg)

	assert.Equal(t, 5, len(containers), "bad containers number from config")
}

func TestNewContainerFromDocker(t *testing.T) {
	createdTime := time.Now()
	id := "2201c17d77c64d51a422c5732cb6368e010dfa47df8724378f4076f465de84c3"

	apiContainer := &docker.Container{
		ID: id,
		Config: &docker.Config{
			Image: "quay.io/myapp:1.9.2",
			Labels: map[string]string{
				"rocker-compose-config": "image: quay.io/myapp:1.9.2",
			},
		},
		State: docker.State{
			Running: true,
		},
		Name:       "/myapp.main",
		Created:    createdTime,
		HostConfig: &docker.HostConfig{},
	}

	container, err := NewContainerFromDocker(apiContainer)
	if err != nil {
		t.Fatal(err)
	}

	assertionName := &config.ContainerName{
		Namespace: "myapp",
		Name:      "main",
	}

	assert.Equal(t, id, container.ID)
	assert.Equal(t, &ContainerState{Running: true}, container.State)
	assert.Equal(t, createdTime, container.Created)
	assert.Equal(t, "quay.io/myapp:1.9.2", container.Image.String())
	assert.Equal(t, "quay.io/myapp:1.9.2", *container.Config.Image)
	assert.Equal(t, assertionName, container.Name)
}

func TestNewFromDocker(t *testing.T) {
	cfg, err := config.NewFromFile("config/testdata/compose.yml", containerTestVars, map[string]interface{}{}, false)
	if err != nil {
		t.Fatal(err)
	}

	container := NewContainerFromConfig(config.NewContainerName("myapp", "main"), cfg.Containers["main"])

	opts, err := container.CreateContainerOptions()
	if err != nil {
		t.Fatal(err)
	}

	apiContainer := &docker.Container{
		Config: &docker.Config{
			Labels: opts.Config.Labels,
		},
		State: docker.State{
			Running: true,
		},
		Name: "/myapp.main",
	}

	configFromAPI, err := config.NewFromDocker(apiContainer)
	if err != nil {
		t.Fatal(err)
	}

	compareResult := cfg.Containers["main"].IsEqualTo(configFromAPI)
	assert.True(t, compareResult,
		"container spec converted from API should be equal to one fetched from config file, failed on field: %s", cfg.Containers["main"].LastCompareField())
}
