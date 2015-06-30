package compose

import (
	"fmt"
	"strings"
	"time"
	log "github.com/Sirupsen/logrus"
	"github.com/fsouza/go-dockerclient"
)

type Container struct {
	Id        string
	Image     *ImageName
	Name      *ContainerName
	Created   time.Time
	State     *ContainerState
	Config    *ConfigContainer

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
	image := &ImageName{}
	if containerConfig.Image != nil {
		image = NewImageNameFromString(*containerConfig.Image)
	}
	return &Container{
		Image: image,
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
	var same bool
	if same = a.IsSameKind(b); same {
		if !a.Config.IsEqualTo(b.Config) {
			log.Debugf("Comparing '%s' and '%s': found difference in '%s'",
				a.Name.String(),
				b.Name.String(),
				a.Config.LastCompareField())
			return false
		}
		if !a.State.IsEqualState(b.State) {
			log.Debugf("Comparing '%s' and '%s': found difference in state: Running: %t != %t",
				a.Name.String(),
				b.Name.String(),
				a.State.Running,
				b.State.Running)
			return false
		}
	}
	return same
}

func (a *ContainerState) IsEqualState(b *ContainerState) bool {
	return a.Running == b.Running
}

func (container *Container) CreateContainerOptions() docker.CreateContainerOptions {
	return docker.CreateContainerOptions{
		Name:       container.Name.String(),
		Config:     container.Config.GetApiConfig(),
		HostConfig: container.Config.GetApiHostConfig(),
	}
}
