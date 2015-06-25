package compose

import (
	"fmt"
	"strings"
	"time"

	"github.com/fsouza/go-dockerclient"
)

type Container struct {
	Id      string
	Image   *ImageName
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

func (containerName *ContainerName) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var name string
	if err := unmarshal(&name); err != nil {
		return err
	}
	*containerName = *NewContainerNameFromString(name)
	return nil
}

func (containerName *ContainerName) String() string {
	if containerName.Namespace == "" {
		return containerName.Name
	}
	return fmt.Sprintf("%s.%s", containerName.Namespace, containerName.Name)
}

func (a *ContainerName) IsEqualTo(b *ContainerName) bool {
	return a.Namespace == b.Namespace && a.Name == b.Name
}

func (a *ContainerName) DefaultNamespace(ns string) *ContainerName {
	newContainerName := *a // copy object
	if newContainerName.Namespace == "" {
		newContainerName.Namespace = ns
	}
	return &newContainerName
}

func NewContainerName(namespace, name string) *ContainerName {
	return &ContainerName{namespace, name}
}

func NewContainerNameFromString(str string) *ContainerName {
	containerName := &ContainerName{}
	split := strings.SplitN(str, ".", 2)
	if len(split) > 1 {
		containerName.Namespace = split[0]
		containerName.Name = split[1]
	} else {
		containerName.Name = split[0]
	}
	return containerName
}

func NewContainerFromDocker(dockerContainer *docker.Container) *Container {
	return &Container{
		Id:      dockerContainer.ID,
		Image:   NewImageNameFromString(dockerContainer.Image),
		Name:    NewContainerNameFromString(dockerContainer.Name),
		Created: dockerContainer.Created,
		State: &ContainerState{
			Running:    dockerContainer.State.Running,
			Paused:     dockerContainer.State.Paused,
			Restarting: dockerContainer.State.Restarting,
			OOMKilled:  dockerContainer.State.OOMKilled,
			Pid:        dockerContainer.State.Pid,
			ExitCode:   dockerContainer.State.ExitCode,
			Error:      dockerContainer.State.Error,
			StartedAt:  dockerContainer.State.StartedAt,
			FinishedAt: dockerContainer.State.FinishedAt,
		},
		Config:    NewContainerConfigFromDocker(dockerContainer),
		container: dockerContainer,
	}
}

func NewContainerFromConfig(name *ContainerName, containerConfig *ConfigContainer) *Container {
	return &Container{
		Image:  NewImageNameFromString(containerConfig.Image),
		Name:   name,
		State:  &ContainerState{},
		Config: containerConfig,
	}
}

func (a *Container) IsSameKind(b *Container) bool {
	// TODO: compare other properties?
	return a.Name.IsEqualTo(b.Name)
}

func (a *Container) IsEqualTo(b *Container) bool {
	// TODO: compare other properties?
	return a.Config.IsEqualTo(b.Config)
}

func (container *Container) CreateContainerOptions() docker.CreateContainerOptions {
	return docker.CreateContainerOptions{}
}
