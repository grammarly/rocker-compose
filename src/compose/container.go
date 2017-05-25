/*-
 * Copyright 2015 Grammarly, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package compose

import (
	"github.com/snkozlov/rocker-compose/src/compose/config"
	"github.com/snkozlov/rocker-compose/src/util"
	"strings"
	"time"

	"github.com/go-yaml/yaml"
	"github.com/grammarly/rocker/src/imagename"

	log "github.com/Sirupsen/logrus"
	"github.com/fsouza/go-dockerclient"
)

// Container object represents a single container produced by a rocker-compose spec
type Container struct {
	ID            string
	Image         *imagename.ImageName
	ImageResolved *imagename.ImageName
	ImageID       string
	Name          *config.ContainerName
	Created       time.Time
	State         *ContainerState
	Config        *config.Container
	Io            *ContainerIo

	container *docker.Container
}

// ContainerState represents the state of a container.
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

// GetContainersFromConfig returns the list of Container objects from
// a spec Config object.
func GetContainersFromConfig(cfg *config.Config) []*Container {
	var containers []*Container
	for name, containerConfig := range cfg.Containers {
		if strings.HasPrefix(name, "_") {
			continue
		}
		containerName := config.NewContainerName(cfg.Namespace, name)
		containers = append(containers, NewContainerFromConfig(containerName, containerConfig))
	}
	return containers
}

// NewContainerFromConfig makes a single Container object from a spec Config object.
func NewContainerFromConfig(name *config.ContainerName, containerConfig *config.Container) *Container {
	container := &Container{
		Name: name,
		State: &ContainerState{
			Running: containerConfig.State.Bool(),
		},
		Config: containerConfig,
	}
	if containerConfig.Image != nil {
		container.Image = imagename.NewFromString(*containerConfig.Image)
	}
	return container
}

// NewContainerFromDocker converts a container object given by
// docker client to a local Container object
func NewContainerFromDocker(dockerContainer *docker.Container) (*Container, error) {
	cfg, err := config.NewFromDocker(dockerContainer)
	if err != nil {
		if _, ok := err.(config.ErrNotRockerCompose); !ok {
			return nil, err
		}
	}
	return &Container{
		ID:      dockerContainer.ID,
		Image:   imagename.NewFromString(dockerContainer.Config.Image),
		ImageID: dockerContainer.Image,
		Name:    config.NewContainerNameFromString(dockerContainer.Name),
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
		Config:    cfg,
		container: dockerContainer,
	}, nil
}

// String returns container name
func (a Container) String() string {
	return a.Name.String()
}

// IsSameNamespace returns true if current and given containers are from same namespace
func (a *Container) IsSameNamespace(b *Container) bool {
	return a.Name.IsEqualNs(b.Name)
}

// IsSameKind returns true if current and given containers have same name,
// without considering namespace
func (a *Container) IsSameKind(b *Container) bool {
	// TODO: compare other properties?
	return a.Name.IsEqualTo(b.Name)
}

// IsEqualTo returns true if current and given containers are equal by
// all dimensions. It compares configuration, image id (image can be updated),
// state (running, craeted)
func (a *Container) IsEqualTo(b *Container) bool {
	// check name
	if !a.IsSameKind(b) {
		return false
	}

	// check configuration
	if !a.Config.IsEqualTo(b.Config) {
		log.Debugf("Comparing '%s' and '%s': found difference in '%s'",
			a.Name.String(),
			b.Name.String(),
			a.Config.LastCompareField())
		return false
	}

	// check image version
	if a.Image != nil && !a.Image.Contains(b.Image) {
		log.Debugf("Comparing '%s' and '%s': image version '%s' is not satisfied (was %s should satisfy %s)",
			a.Name.String(),
			b.Name.String(),
			a.Image,
			b.Image,
			a.Image)
		return false
	}

	// check image id
	if a.ImageID != "" && b.ImageID != "" && a.ImageID != b.ImageID {
		log.Debugf("Comparing '%s' and '%s': image '%s' updated (was %.12s became %.12s)",
			a.Name.String(),
			b.Name.String(),
			a.Image,
			b.ImageID,
			a.ImageID)
		return false
	}

	// One of exit codes is always '0' since once of containers (a or b) is always loaded from config
	if a.Config.State.IsRan() && a.State.ExitCode+b.State.ExitCode > 0 {
		log.Debugf("Comparing '%s' and '%s': container should run once, but previous exit code was %d",
			a.Name.String(),
			b.Name.String(),
			a.State.ExitCode+b.State.ExitCode)
		return false
	}

	// check state
	if !a.State.IsEqualState(b.State) {
		log.Debugf("Comparing '%s' and '%s': found difference in state: Running: %t != %t",
			a.Name.String(),
			b.Name.String(),
			a.State.Running,
			b.State.Running)
		return false
	}

	return true
}

// IsEqualState returns true if current and given containers have the same state
func (a *ContainerState) IsEqualState(b *ContainerState) bool {
	return a.Running == b.Running
}

// CreateContainerOptions returns create configuration eatable by go-dockerclient
func (a *Container) CreateContainerOptions() (*docker.CreateContainerOptions, error) {
	apiConfig := a.Config.GetAPIConfig()

	yamlData, err := yaml.Marshal(a.Config)
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
	apiConfig.Image = a.Image.String()

	return &docker.CreateContainerOptions{
		Name:       a.Name.String(),
		Config:     apiConfig,
		HostConfig: a.Config.GetAPIHostConfig(),
	}, nil
}
