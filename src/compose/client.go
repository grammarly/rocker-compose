package compose

import (
	"github.com/fsouza/go-dockerclient"
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
	Timeout int // Timeout for fetch, stop, start and other possible actions
}

func NewClient(initialClient *ClientCfg) (Client, error) {
	client := &ClientCfg{
		Docker: initialClient.Docker,
		Timeout: initialClient.Timeout,
	}
	return client, nil
}

func (client *ClientCfg) GetContainers() ([]*Container, error) {
	containers := []*Container{}
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
