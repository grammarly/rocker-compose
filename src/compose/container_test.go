package compose

import (
	"compose/config"
	"testing"
	"time"

	"github.com/fsouza/go-dockerclient"
	"github.com/stretchr/testify/assert"
)

var (
	containerTestVars = map[string]interface{}{
		"version": map[string]string{
			"patterns": "1.9.2",
		},
	}
)

// type T struct {
// 	Names []*Name
// }

// type Name struct {
// 	Value string
// }

// func (name *Name) MarshalYAML() (interface{}, error) {
// 	return name.Value, nil
// }

func TestCreateContainerOptions(t *testing.T) {
	// val := &T{
	// 	Names: []*Name{&Name{"asd"}},
	// }
	// data, err := yaml.Marshal(val)
	// if err != nil {
	// 	t.Fatal(err)
	// }

	// pretty.Println(string(data))

	cfg, err := config.NewFromFile("config/testdata/compose.yml", containerTestVars, map[string]interface{}{})
	if err != nil {
		t.Fatal(err)
	}

	container := NewContainerFromConfig(config.NewContainerName("patterns", "main"), cfg.Containers["main"])

	opts, err := container.CreateContainerOptions()
	if err != nil {
		t.Fatal(err)
	}

	// pretty.Println(opts.Config.Labels["rocker-compose-config"])

	assert.IsType(t, &docker.CreateContainerOptions{}, opts)
}

func TestConfigGetContainers(t *testing.T) {
	cfg, err := config.NewFromFile("config/testdata/compose.yml", containerTestVars, map[string]interface{}{})
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
			Image: "dockerhub.grammarly.io/patterns:1.9.2",
			Labels: map[string]string{
				"rocker-compose-config": "image: dockerhub.grammarly.io/patterns:1.9.2",
			},
		},
		State: docker.State{
			Running: true,
		},
		Name:       "/patterns.main",
		Created:    createdTime,
		HostConfig: &docker.HostConfig{},
	}

	container, err := NewContainerFromDocker(apiContainer)
	if err != nil {
		t.Fatal(err)
	}

	// pretty.Println(container)

	assertionImage := &ImageName{
		Registry: "dockerhub.grammarly.io",
		Name:     "patterns",
		Tag:      "1.9.2",
	}
	assertionName := &config.ContainerName{
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

func TestNewFromDocker(t *testing.T) {
	cfg, err := config.NewFromFile("config/testdata/compose.yml", containerTestVars, map[string]interface{}{})
	if err != nil {
		t.Fatal(err)
	}

	container := NewContainerFromConfig(config.NewContainerName("patterns", "main"), cfg.Containers["main"])

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
		Name: "/patterns.main",
		// RestartCount: 5, // TODO: test it
	}

	// fmt.Printf("%# v\n", pretty.Formatter(cfg.Containers["main"]))

	configFromApi, err := config.NewFromDocker(apiContainer)
	if err != nil {
		t.Fatal(err)
	}

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

	// pretty.Println(cfg.Containers["main"])

	// pretty.Println(configFromApi)
	// pretty.Println(cfg.Containers["main"].Labels)
	// pretty.Println(configFromApi.Labels)

	compareResult := cfg.Containers["main"].IsEqualTo(configFromApi)
	assert.True(t, compareResult,
		"container spec converted from API should be equal to one fetched from config file, failed on field: %s", cfg.Containers["main"].LastCompareField())
}
