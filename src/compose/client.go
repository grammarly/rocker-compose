package compose

import (
	"fmt"
	"github.com/fsouza/go-dockerclient"
)

type Client struct {
	Docker  *docker.Client
	Timeout int // Timeout for fetch, stop, start and other possible actions
}

type ContainerName struct {
	Namespace string
	Name      string
}

func (containerName *ContainerName) String() {
	return fmt.Printf("%s.%s", containerName.Namespace, containerName.Name)
}

func NewClient(initialClient *Client) (*Client, error) {
	client := &Client{
		Docker: initialClient.Docker,
	}
	return client, nil
}

func (client *Client) GetContainers() ([]*docker.Container, error) {
	containers := []*docker.Container{}
	return containers, nil
}

func (client *Client) RemoveContainer(name *ContainerName, config *ConfigContainer) error {
	// Note: optionally stop if kill_timeout is set
	// TODO: RemoveVolumes ?
	return nil
}

func (client *Client) CreateContainer(name *ContainerName, config *ConfigContainer) error {
	return nil
}
