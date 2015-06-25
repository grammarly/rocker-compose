package compose

import (
	"fmt"
	"time"

	"github.com/fsouza/go-dockerclient"
)

type Container struct {
	Id      string
	Image   string
	Name    *ContainerName
	Created time.Time
	State   *ContainerState
	Config  *ConfigContainer

	container *docker.Container
}

// State represents the state of a container.
type ContainerState struct {
	Running    bool
	Paused     bool
	Restarting bool
	OOMKilled  bool
	Pid        int
	ExitCode   int
	Error      string
	StartedAt  time.Time
	FinishedAt time.Time
}

type ContainerName struct {
	Namespace string
	Name      string
}

func (containerName *ContainerName) String() string {
	return fmt.Sprintf("%s.%s", containerName.Namespace, containerName.Name)
}

func NewContainerFromDocker(dockerContainer *docker.Container) *Container {
	return &Container{}
}

func (container *Container) CreateContainerOptions() docker.CreateContainerOptions {
	return docker.CreateContainerOptions{}
}
