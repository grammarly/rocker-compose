package compose

import (
	"github.com/fsouza/go-dockerclient"
)

type Client struct {
	Docker  *docker.Client
	Timeout int // Timeout for fetch, stop, start and other possible actions
}

func NewClient(initialClient *Client) (*Client, error) {
	client := &Client{
		Docker: initialClient.Docker,
	}
	return client, nil
}

func (client *Client) GetContainers() ([]*Container, error) {
	containers := []*Container{}
	return containers, nil
}

func (client *Client) RemoveContainer(container *Container) error {
	// Note: optionally stop if kill_timeout is set
	// TODO: RemoveVolumes ?
	return nil
}

func (client *Client) CreateContainer(container *Container) error {
	return nil
}

func (client *Client) PullImage(imageName *ImageName) error {
	return nil
}
