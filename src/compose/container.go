package compose

import (
	"fmt"
	"strings"
	"time"
	"util"

	"github.com/go-yaml/yaml"

	log "github.com/Sirupsen/logrus"
	"github.com/fsouza/go-dockerclient"
)

type Container struct {
	Id      string
	Image   *ImageName
	ImageId string
	Name    *ContainerName
	Created time.Time
	State   *ContainerState
	Config  *ConfigContainer
	Io      *ContainerIo

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
	Alias     string
}

func (containerName *ContainerName) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var name string
	if err := unmarshal(&name); err != nil {
		return err
	}
	*containerName = *NewContainerNameFromString(name)
	return nil
}

func (containerName ContainerName) MarshalYAML() (interface{}, error) {
	return containerName.String(), nil
}

func (containerName *ContainerName) String() string {
	name := containerName.Name
	if containerName.Namespace != "" {
		name = fmt.Sprintf("%s.%s", containerName.Namespace, name)
	}
	if containerName.Alias != "" {
		name = fmt.Sprintf("%s:%s", name, containerName.Alias)
	}
	return name
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
	return &ContainerName{namespace, name, ""}
}

// format: name | namespace.name | name:alias | namespace.name:alias
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
	split2 := strings.SplitN(containerName.Name, ":", 2)
	if len(split2) > 1 {
		containerName.Name = split2[0]
		containerName.Alias = split2[1]
	} else {
		containerName.Name = split2[0]
	}
	return containerName
}

func NewContainerFromConfig(name *ContainerName, containerConfig *ConfigContainer) *Container {
	container := &Container{
		Name: name,
		State: &ContainerState{
			Running: containerConfig.State.RunningBool(),
		},
		Config: containerConfig,
	}
	if containerConfig.Image != nil {
		container.Image = NewImageNameFromString(*containerConfig.Image)
	}
	return container
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
		if a.ImageId != "" && b.ImageId != "" {
			if a.ImageId != b.ImageId {
				log.Debugf("Comparing '%s' and '%s': image '%s' updated (was %s became %s)",
					a.Name.String(),
					b.Name.String(),
					a.Image,
					util.TruncateID(b.ImageId),
					util.TruncateID(a.ImageId))
				return false
			}
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

func (container *Container) CreateContainerOptions() (*docker.CreateContainerOptions, error) {
	apiConfig := container.Config.GetApiConfig()

	yamlData, err := yaml.Marshal(container.Config)
	if err != nil {
		return nil, err
	}

	// Copy labels because we want to assign some extra stuff
	labels := map[string]string{}
	for k, v := range apiConfig.Labels {
		labels[k] = v
	}
	labels["rocker-compose-id"] = util.GenerateRandomID()
	labels["rocker-compose-config"] = string(yamlData)

	apiConfig.Labels = labels

	return &docker.CreateContainerOptions{
		Name:       container.Name.String(),
		Config:     apiConfig,
		HostConfig: container.Config.GetApiHostConfig(),
	}, nil
}
