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
	return a.IsEqualNs(b) && a.Name == b.Name
}

func (a *ContainerName) IsEqualNs(b *ContainerName) bool {
	return a.Namespace == b.Namespace
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
	str = strings.TrimPrefix(str, "/") // TODO: investigate why Docker adds prefix slash to container names
	split := strings.SplitN(str, ".", 2)
	if len(split) > 1 {
		containerName.Namespace = split[0]
		containerName.Name = split[1]
	} else {
		containerName.Name = split[0]
	}
	return containerName
}

func NewContainerFromConfig(name *ContainerName, containerConfig *ConfigContainer) *Container {
	return &Container{
		Image: NewImageNameFromString(containerConfig.Image),
		Name:  name,
		State: &ContainerState{
			Running: containerConfig.State.RunningBool(),
		},
		Config: containerConfig,
	}
}

func (a *Container) IsSameNamespace(b *Container) bool {
	return a.Name.IsEqualNs(b.Name)
}

func (a *Container) IsSameKind(b *Container) bool {
	// TODO: compare other properties?
	return a.Name.IsEqualTo(b.Name)
}

func (a *Container) IsEqualTo(b *Container) bool {
	// TODO: compare other properties?
	return a.IsSameKind(b) &&
		a.Config.IsEqualTo(b.Config) &&
		a.State.IsEqualState(b.State)
}

func (a *ContainerState) IsEqualState(b *ContainerState) bool {
	return a.Running == b.Running

}

func (container *Container) CreateContainerOptions() docker.CreateContainerOptions {
	return docker.CreateContainerOptions{}
}
