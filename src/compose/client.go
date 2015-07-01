package compose

import (
	"fmt"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/fsouza/go-dockerclient"
)

type Client interface {
	GetContainers() ([]*Container, error)
	RemoveContainer(container *Container) error
	RunContainer(container *Container) error
	EnsureContainer(name *Container) error
	PullImage(imageName *ImageName) error
	PullAll(config *Config) error
	AttachToContainers(container []*Container) error
}

type ClientCfg struct {
	Docker *docker.Client
	Global bool // Search for existing containers globally, not only ones started with compose
}

func NewClient(initialClient *ClientCfg) (Client, error) {
	client := &ClientCfg{
		Docker: initialClient.Docker,
		Global: initialClient.Global,
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
			container, err := NewContainerFromDocker(resp.container)
			if err != nil {
				return nil, fmt.Errorf("Failed to initialize config container instance from docker api, error: %s", err)
			}
			containers = append(containers, container)

		case <-time.After(30 * time.Second):
		// todo: you may have to use client.Timeout
			return nil, fmt.Errorf("Timeout while fetching containers")
		}

		if len(apiContainers) == numResponses {
			break
		}
	}

	return containers, nil
}

func (client *ClientCfg) RemoveContainer(container *Container) error {
	if container.Config.KillTimeout != nil && *container.Config.KillTimeout > 0 {
		if err := client.Docker.StopContainer(container.Id, *container.Config.KillTimeout); err != nil {
			return fmt.Errorf("Failed to stop container, error: %s", err)
		}
	}
	keepVolumes := container.Config.KeepVolumes != nil && *container.Config.KeepVolumes
	removeOptions := docker.RemoveContainerOptions{
		ID:            container.Id,
		RemoveVolumes: !keepVolumes,
		Force:         true,
	}
	if err := client.Docker.RemoveContainer(removeOptions); err != nil {
		return fmt.Errorf("Failed to remove container, error: %s", err)
	}

	log.Infof("Removed container %s\n", container.Id)
	return nil
}

func (client *ClientCfg) RunContainer(container *Container) error {
	opts, err := container.CreateContainerOptions()
	if err != nil {
		return fmt.Errorf("Failed to initialize container options, error: %s", err)
	}
	apiContainer, err := client.Docker.CreateContainer(*opts)
	if err != nil {
		return fmt.Errorf("Failed to create container, error: %s", err)
	}
	container.Id = apiContainer.ID

	if container.State.Running {
		log.Infof("Starting container %s\n", container.Id)
		// TODO: HostConfig may be changed without re-creation of containers
		// so of Volumes or Links are changed, we just need to restart container
		if err := client.Docker.StartContainer(apiContainer.ID, &docker.HostConfig{}); err != nil {
			return fmt.Errorf("Failed to start container, error: %s", err)
		}
	}

	return nil
}

func (client *ClientCfg) EnsureContainer(container *Container) error {
	if _, err := client.Docker.InspectContainer(container.Name.String()); err != nil {
		return err
	}
	log.Infof("Checking container %s\n", container.Id)
	return nil
}


func (client *ClientCfg) PullImage(imageName *ImageName) error {
	return nil
}

func (client *ClientCfg) AttachToContainers(containers []*Container) error {
	var counter int = len(containers)
	errors := make(chan error, counter)

	for _, container := range containers {
		go func(container *Container, errors chan error) {
			def := log.StandardLogger()
			logger := &log.Logger{
				Out:    def.Out,
				Formatter: NewContainerFormatter(container, log.InfoLevel),
				Level: def.Level,
			}
			errors <- client.Docker.AttachToContainer(docker.AttachToContainerOptions{
				Container:       container.Name.String(),
				OutputStream:    logger.Writer(),
				Stdout: true,
				Stream:true,
			})
		}(container, errors)
	}

	for {
		select {
		case err := <-errors:
			if err != nil {
				return err
			}
			if counter -= 1; counter == 0 {
				return nil
			}
		}
	}
}

func (client *ClientCfg) PullAll(config *Config) error {
	return nil
}
