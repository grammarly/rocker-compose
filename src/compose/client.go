package compose

import (
	"fmt"
	"time"

	"github.com/fsouza/go-dockerclient"
	log "github.com/Sirupsen/logrus"
)

type Client interface {
	GetContainers() ([]*Container, error)
	RemoveContainer(container *Container) error
	CreateContainer(container *Container) error
	EnsureContainer(name *Container) error
	PullImage(imageName *ImageName) error
	PullAll(config *Config) error
}

type ClientCfg struct {
	Docker  *docker.Client
	Timeout int  // Timeout for fetch, stop, start and other possible actions
	Global  bool // Search for existing containers globally, not only ones started with compose
}

func NewClient(initialClient *ClientCfg) (Client, error) {
	client := &ClientCfg{
		Docker:  initialClient.Docker,
		Timeout: initialClient.Timeout,
		Global:  initialClient.Global,
	}
	return client, nil
}

func (client *ClientCfg) GetContainers() ([]*Container, error) {
	filters := map[string][]string{}
	if !client.Global {
		filters["label"] = []string{"rocker-compose-id"}
	}

	apiContainers, err := client.Docker.ListContainers(docker.ListContainersOptions{
		All:     true,
		Filters: filters,
	})
	if err != nil {
		return nil, err
	}

	containers := []*Container{}

	if len(apiContainers) == 0 {
		return containers, nil
	}

	// Fetch detailed information about all containers in parallel
	type chResponse struct {
		container *docker.Container
		err       error
	}

	ch := make(chan *chResponse, len(apiContainers))
	numResponses := 0

	for _, apiContainer := range apiContainers {
		go func(apiContainer docker.APIContainers) {
			chResponse := new(chResponse)
			chResponse.container, chResponse.err = client.Docker.InspectContainer(apiContainer.ID)
			ch <- chResponse
		}(apiContainer)
	}

	log.Infof("Fetching %d containers\n", len(apiContainers))

	for {
		select {
		case resp := <-ch:
			numResponses++
			if resp.err != nil {
				return nil, fmt.Errorf("Failed to fetch container, error: %s", resp.err)
			}
			containers = append(containers, NewContainerFromDocker(resp.container))

		case <-time.After(30 * time.Second): // todo: you may have to use client.Timeout
			return nil, fmt.Errorf("Timeout while fetching containers")
		}

		if len(apiContainers) == numResponses {
			break
		}
	}

	return containers, nil
}

func (client *ClientCfg) RemoveContainer(container *Container) error {
	// Note: optionally stop if kill_timeout is set
	// TODO: RemoveVolumes ?
	return nil
}

func (client *ClientCfg) CreateContainer(container *Container) error {
	return nil
}

func (client *ClientCfg) EnsureContainer(container *Container) error {
	return nil
}

func (client *ClientCfg) PullImage(imageName *ImageName) error {
	return nil
}

func (client *ClientCfg) PullAll(config *Config) error {
	return nil
}
