package compose

import (
	"encoding/json"
	"io/ioutil"
	"strings"
	"testing"
	"time"

	"github.com/fsouza/go-dockerclient"
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

	// actualPretty, err := json.MarshalIndent(config.Containers["test"].GetApiConfig(), "", "    ")
	// if err != nil {
	// 	t.Fatal(err)
	// }
	// pretty.Println(config.Containers["test"])
	// println(string(actualPretty))

	actual, err := json.Marshal(config.Containers["main"].GetApiConfig())
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, strings.TrimSpace(string(expected)), string(actual))
}

func TestConfigGetApiHostConfig(t *testing.T) {
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
	// println(string(actual))

	assert.Equal(t, strings.TrimSpace(string(expected)), string(actual))
}

func TestNewContainerConfigFromDocker(t *testing.T) {
	apiContainer := &docker.Container{
		Config: &docker.Config{
			Image:      "dockerhub.grammarly.io/patterns:1.9.2",
			Hostname:   "pattern1",
			Domainname: "grammarly.com",
			User:       "root",
			Memory:     314572800,
			MemorySwap: 1073741824,
			CPUShares:  512,
			CPUSet:     "0-2",
			ExposedPorts: map[docker.Port]struct{}{
				(docker.Port)("23456/tcp"): struct{}{},
			},
			Env: []string{"AWS_KEY=asdqwe"},
			Cmd: []string{"param1", "param2"},
			Volumes: map[string]struct{}{
				"/var/log": struct{}{},
			},
			WorkingDir:      "/app",
			Entrypoint:      []string{"/bin/patterns"},
			NetworkDisabled: true,
			Labels: map[string]string{
				"num":     "1",
				"service": "pattern",
			},
		},
		State: docker.State{
			Running: true,
		},
		Name: "/patterns.main",
		HostConfig: &docker.HostConfig{
			Binds:      []string{"/tmp/patterns/tmpfs:/tmp/tmpfs", "/tmp/patterns/log:/opt/gr-pat/log:ro"},
			Privileged: true,
			PortBindings: map[docker.Port][]docker.PortBinding{
				(docker.Port)("23456"):    []docker.PortBinding{docker.PortBinding{"", "23456"}},
				(docker.Port)("5005/tcp"): []docker.PortBinding{docker.PortBinding{"0.0.0.0", "5005"}},
				(docker.Port)("5006/tcp"): []docker.PortBinding{docker.PortBinding{"", "5006"}},
			},
			Links:           []string{"monitoring.sensu"},
			PublishAllPorts: true,
			DNS:             []string{"8.8.8.8"},
			ExtraHosts:      []string{"capi.grammarly.com:127.0.0.1"},
			VolumesFrom:     []string{"patterns.config", "patterns.extdata", "monitoring.sensu"},
			NetworkMode:     "host",
			PidMode:         "host",
			RestartPolicy:   docker.RestartPolicy{Name: "always"},
			Memory:          314572800,
			MemorySwap:      1073741824,
			CPUShares:       512,
			CPUSet:          "0-2",
			CPUPeriod:       0,
			Ulimits: []docker.ULimit{
				docker.ULimit{"nofile", 1024, 2048},
			},
		},
		// RestartCount: 5, // TODO: test it
	}

	config, err := ReadConfigFile("testdata/compose.yml", configTestVars)
	if err != nil {
		t.Fatal(err)
	}

	// fmt.Printf("%# v\n", pretty.Formatter(config.Containers["main"]))

	configFromApi := NewContainerConfigFromDocker(apiContainer)

	// fmt.Printf("%# v\n", pretty.Formatter(configFromApi))

	// newContainer := &docker.Container{
	// 	Config:     configFromApi.GetApiConfig(),
	// 	HostConfig: configFromApi.GetApiHostConfig(),
	// }
	// jsonResult, err := json.Marshal(newContainer)
	// if err != nil {
	// 	t.Fatal(err)
	// }
	// println(string(jsonResult))

	compareResult := config.Containers["main"].IsEqualTo(configFromApi)
	assert.True(t, compareResult,
		"container spec converted from API should be equal to one fetched from config file, failed on field: %s", config.Containers["main"].LastCompareField())
}

func TestNewContainerFromDocker(t *testing.T) {
	createdTime := time.Now()
	id := "2201c17d77c64d51a422c5732cb6368e010dfa47df8724378f4076f465de84c3"

	apiContainer := &docker.Container{
		ID: id,
		Config: &docker.Config{
			Image: "dockerhub.grammarly.io/patterns:1.9.2",
		},
		State: docker.State{
			Running: true,
		},
		Name:       "/patterns.main",
		Created:    createdTime,
		HostConfig: &docker.HostConfig{},
	}

	container := NewContainerFromDocker(apiContainer)

	// pretty.Println(container)

	assertionImage := &ImageName{
		Registry: "dockerhub.grammarly.io",
		Name:     "patterns",
		Tag:      "1.9.2",
	}
	assertionName := &ContainerName{
		Namespace: "patterns",
		Name:      "main",
	}

	assert.Equal(t, id, container.Id)
	assert.Equal(t, &ContainerState{Running: true}, container.State)
	assert.Equal(t, createdTime, container.Created)
	assert.Equal(t, assertionImage, container.Image)
	assert.Equal(t, assertionImage.String(), *container.Config.Image)
	assert.Equal(t, assertionName, container.Name)
}
