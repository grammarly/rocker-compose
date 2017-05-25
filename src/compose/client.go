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
	"fmt"
	"os"
	"time"

	"github.com/snkozlov/rocker-compose/src/compose/config"
	"github.com/snkozlov/rocker-compose/src/util"

	"github.com/grammarly/rocker/src/dockerclient"
	"github.com/grammarly/rocker/src/imagename"
	"github.com/grammarly/rocker/src/storage/s3"
	"github.com/grammarly/rocker/src/template"
	"github.com/kr/pretty"

	log "github.com/Sirupsen/logrus"
	"github.com/fsouza/go-dockerclient"
)

// Client interface describes a rocker-compose client that can do various operations
// needed for rocker-compose to make changes.
type Client interface {
	GetContainers(global bool) ([]*Container, error)
	RemoveContainer(container *Container) error
	RunContainer(container *Container) error
	EnsureContainerExist(name *Container) error
	EnsureContainerState(name *Container) error
	PullAll(containers []*Container, vars template.Vars) error
	Clean(config *config.Config) error
	AttachToContainers(container []*Container) error
	AttachToContainer(container *Container) error
	FetchImages(containers []*Container, vars template.Vars) error
	WaitForContainer(container *Container) error
	GetPulledImages() []*imagename.ImageName
	GetRemovedImages() []*imagename.ImageName
	Pin(local, hub bool, vars template.Vars, containers []*Container) error
}

// DockerClient is an implementation of Client interface that do operations to a given docker client
type DockerClient struct {
	Docker     *docker.Client
	Attach     bool
	Wait       time.Duration
	Auth       *docker.AuthConfigurations
	KeepImages int
	Recover    bool

	pulledImages  []*imagename.ImageName
	removedImages []*imagename.ImageName
}

// ErrContainerBadState is an error that describes state inconsistency
// that can be checked by EnsureContainerState function
type ErrContainerBadState struct {
	Container  *Container
	Running    bool
	OOMKilled  bool
	ExitCode   int
	ErrorStr   string
	StartedAt  time.Time
	FinishedAt time.Time
}

// Error returns string representation of the error
func (e ErrContainerBadState) Error() string {
	str := fmt.Sprintf("Container %s exited with code %d", e.Container.Name, e.ExitCode)
	if e.ErrorStr != "" {
		str = fmt.Sprintf("%s, error: %s", str, e.ErrorStr)
	}
	return str
}

// NewClient makes a new DockerClient object based on configuration params
// that is given with input DockerClient object.
func NewClient(initialClient *DockerClient) (*DockerClient, error) {
	client := &DockerClient{
		Docker:     initialClient.Docker,
		Attach:     initialClient.Attach,
		Wait:       initialClient.Wait,
		Auth:       initialClient.Auth,
		KeepImages: initialClient.KeepImages,
		Recover:    initialClient.Recover,
	}
	return client, nil
}

// GetContainers implements the retrieval of existing containers from the docker daemon.
// It fetches the list and then inspects every container in parallel (pmap).
// Timeouts after 30 seconds if some inspect operations hanged.
func (client *DockerClient) GetContainers(global bool) ([]*Container, error) {
	filters := map[string][]string{}
	if !global {
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

	for _, apiContainer := range apiContainers {
		go func(apiContainer docker.APIContainers) {
			chResponse := new(chResponse)
			chResponse.container, chResponse.err = client.Docker.InspectContainer(apiContainer.ID)
			ch <- chResponse
		}(apiContainer)
	}

	log.Infof("Gathering info about %d containers", len(apiContainers))

	timeout := time.After(30 * time.Second)

	for range apiContainers {
		select {
		case resp := <-ch:
			if resp.err != nil {
				return nil, fmt.Errorf("Failed to fetch container, error: %s", resp.err)
			}
			container, err := NewContainerFromDocker(resp.container)
			if err != nil {
				return nil, fmt.Errorf("Failed to initialize config container instance from docker api, error: %s", err)
			}
			containers = append(containers, container)

		case <-timeout:
			// todo: you may have to use client.Timeout
			return nil, fmt.Errorf("Timeout while fetching containers")
		}
	}

	return containers, nil
}

// RemoveContainer implements removing a container
func (client *DockerClient) RemoveContainer(container *Container) error {
	log.Infof("Removing container %s id:%.12s", container.Name, container.ID)

	if container.Config.KillTimeout != nil && *container.Config.KillTimeout > 0 {
		if err := client.Docker.StopContainer(container.ID, *container.Config.KillTimeout); err != nil {
			return fmt.Errorf("Failed to stop container, error: %s", err)
		}
	}
	keepVolumes := container.Config.KeepVolumes != nil && *container.Config.KeepVolumes
	removeOptions := docker.RemoveContainerOptions{
		ID:            container.ID,
		RemoveVolumes: !keepVolumes,
		Force:         true,
	}
	if err := client.Docker.RemoveContainer(removeOptions); err != nil {
		return fmt.Errorf("Failed to remove container, error: %s", err)
	}

	return nil
}

// RunContainer implements creating and optionally running a container
// depending on its state preference.
func (client *DockerClient) RunContainer(container *Container) error {
	log.Infof("Create container %s", container.Name)

	opts, err := container.CreateContainerOptions()
	if err != nil {
		return fmt.Errorf("Failed to initialize container options, error: %s", err)
	}
	log.Debugf("Creating container with opts: %# v", pretty.Formatter(opts))

	apiContainer, err := client.Docker.CreateContainer(*opts)
	if err != nil {
		return fmt.Errorf("Failed to create container, error: %s", err)
	}
	container.ID = apiContainer.ID

	if container.State.Running || container.Config.State.IsRan() {
		if client.Attach {
			if err := client.AttachToContainer(container); err != nil {
				return err
			}
		}

		if err := client.StartContainer(container); err != nil {
			return err
		}
	}

	return nil
}

// StartContainer implements starting a container
// If contianer state is "ran" then it waits until container exit and checks exit code;
// otherwise it waits for configurable '--wait' seconds interval and ensures container
// not exited.
func (client *DockerClient) StartContainer(container *Container) error {
	log.Infof("Starting container %s id:%.12s from image %s", container.Name, container.ID, container.Image)

	// TODO: HostConfig may be changed without re-creation of containers
	// so of Volumes or Links are changed, we just need to restart container
	if err := client.Docker.StartContainer(container.ID, nil); err != nil {
		if !client.Attach {
			client.flushContainerLogs(container)
		}
		return fmt.Errorf("Failed to start container, error: %s", err)
	}

	if container.Config.State.IsRan() {
		// TODO: refactor to use DockerClient.WaitForContainer() ?
		exitCode, err := client.Docker.WaitContainer(container.Name.String())
		if !client.Attach && (err != nil || exitCode != 0) {
			client.flushContainerLogs(container)
		}
		if err != nil {
			return err
		}
		if exitCode != 0 {
			return fmt.Errorf("Container %s exited with code %d", container.Name, exitCode)
		}
	} else if client.Wait > 0 {
		log.Infof("Waiting for %s to ensure %s not exited abnormally...", client.Wait, container.Name)
		time.Sleep(client.Wait)

		if err := client.EnsureContainerState(container); err != nil {
			if !client.Attach {
				client.flushContainerLogs(container)
			}
			return err
		}
	}
	return nil
}

// EnsureContainerExist implements ensuring that container exists in docker daemon
func (client *DockerClient) EnsureContainerExist(container *Container) error {
	log.Infof("Checking container exist %s", container.Name)
	if _, err := client.Docker.InspectContainer(container.Name.String()); err != nil {
		return err
	}
	return nil
}

// EnsureContainerState checks that the state of existing docker daemon container
// equals expected state specified in the spec.
func (client *DockerClient) EnsureContainerState(container *Container) error {
	log.Debugf("Checking container state %s", container.Name)
	inspect, err := client.Docker.InspectContainer(container.Name.String())
	if err != nil {
		return err
	}
	err = ErrContainerBadState{
		Container:  container,
		Running:    inspect.State.Running,
		OOMKilled:  inspect.State.OOMKilled,
		ExitCode:   inspect.State.ExitCode,
		ErrorStr:   inspect.State.Error,
		StartedAt:  inspect.State.StartedAt,
		FinishedAt: inspect.State.FinishedAt,
	}
	log.Debugf("Container state for %s: %# v", container.Name, inspect.State)

	if client.Recover && !inspect.State.Running && container.State.Running {
		return client.StartContainer(container)
	}
	if inspect.State.ExitCode != 0 {
		return err
	}
	return nil
}

// PullAll grabs all image names from containers in spec and pulls all of them
func (client *DockerClient) PullAll(containers []*Container, vars template.Vars) error {
	return client.pullImageForContainers(true, vars, containers...)
}

// Clean finds the obsolete image tags from container specs that exist in docker daemon,
// skipping topN images that we want to keep (keep_images, default 5) and deletes them.
func (client *DockerClient) Clean(config *config.Config) error {
	// do not pull same image twice
	images := map[imagename.ImageName]*imagename.Tags{}
	keep := client.KeepImages

	// keep 5 latest images by default
	if keep == 0 {
		keep = 5
	}

	for _, container := range GetContainersFromConfig(config) {
		if container.Image == nil {
			continue
		}
		images[*container.Image] = &imagename.Tags{}
	}

	if len(images) == 0 {
		return nil
	}

	// Go through every image and list existing tags
	all, err := client.Docker.ListImages(docker.ListImagesOptions{})
	if err != nil {
		return fmt.Errorf("Failed to list all images, error: %s", err)
	}

	// collect tags for every image
	for _, image := range all {
		for _, repoTag := range image.RepoTags {
			imageName := imagename.NewFromString(repoTag)
			for img := range images {
				if img.IsSameKind(*imageName) {
					images[img].Items = append(images[img].Items, &imagename.Tag{
						ID:      image.ID,
						Name:    *imageName,
						Created: image.Created,
					})
				}
			}
		}
	}

	// for every image, delete obsolete tags
	for name, tags := range images {
		toDelete := tags.GetOld(keep)
		if len(toDelete) == 0 {
			continue
		}

		log.Infof("Cleanup: removing %d tags of image %s", len(toDelete), name.NameWithRegistry())
		for _, n := range toDelete {
			if name.GetTag() == n.GetTag() {
				log.Infof("Cleanup: skipping %s because it is in the spec", n)
				continue
			}

			wasRemoved := true

			log.Infof("Cleanup: remove %s", n)
			if err := client.Docker.RemoveImageExtended(n.String(), docker.RemoveImageOptions{Force: false}); err != nil {
				// 409 is conflict, which means there is a container exists running under this image
				if e, ok := err.(*docker.Error); ok && e.Status == 409 {
					log.Infof("Cleanup: skip %s because there is an existing container using it", n)
					wasRemoved = false
				} else {
					return err
				}
			}

			// cannot refer to &n because of for loop
			if wasRemoved {
				removed := n
				client.removedImages = append(client.removedImages, &removed)
			}
		}
	}

	return nil
}

// AttachToContainer attaches to a running container and redirects its streams to log
func (client *DockerClient) AttachToContainer(container *Container) error {
	success := make(chan struct{})
	errors := make(chan error, 1)

	if container.Io == nil {
		container.Io = NewContainerIo(container)
	} else {
		container.Io.Resurrect()
	}

	go func() {
		var err error
		defer container.Io.Done(err)

		err = client.Docker.AttachToContainer(docker.AttachToContainerOptions{
			Container:    container.Name.String(),
			OutputStream: container.Io.Stdout,
			ErrorStream:  container.Io.Stderr,
			Stdout:       true,
			Stderr:       true,
			Stream:       true,
			Success:      success,
		})
	}()

	select {
	case err := <-errors:
		return err
	case ack := <-success:
		success <- ack
	}

	return nil
}

// AttachToContainers attaches to all containers that specified to be running
func (client *DockerClient) AttachToContainers(containers []*Container) error {
	running := []*Container{}
	for _, c := range containers {
		if c.State.Running {
			running = append(running, c)
		}
	}

	// Listen to events of all containers and re-attach if necessary
	go client.listenReAttach(containers)

	wg := util.NewErrorWaitGroup(len(running))

	for _, container := range running {
		if container.Io == nil {
			if err := client.AttachToContainer(container); err != nil {
				return err
			}
		}

		go func(container *Container) {
			wg.Done(container.Io.Wait())
		}(container)
	}

	return wg.Wait()
}

// WaitForContainer waits for a container and checks exit code at the end
// If exitCode != 0 then fires an error
func (client *DockerClient) WaitForContainer(container *Container) (err error) {
	var (
		inspect  *docker.Container
		exitCode int
	)
	if inspect, err = client.Docker.InspectContainer(container.Name.String()); err != nil {
		return
	}
	// Wait only if the container if not long-running and still not exited
	if !container.Config.State.Bool() && inspect.State.Running == true {
		log.Infof("Waiting container to finish %s", container.Name)
		if exitCode, err = client.Docker.WaitContainer(container.Name.String()); err != nil {
			return
		}
	}

	if exitCode != 0 {
		return fmt.Errorf("Container %s exited with code %d", container.Name, exitCode)
	}

	return nil
}

// FetchImages fetches the missing images for all containers in the manifest
func (client *DockerClient) FetchImages(containers []*Container, vars template.Vars) error {
	return client.pullImageForContainers(false, vars, containers...)
}

// GetPulledImages returns the list of images pulled by a recent run
func (client *DockerClient) GetPulledImages() []*imagename.ImageName {
	return client.pulledImages
}

// GetRemovedImages returns the list of images removed by a recent run
func (client *DockerClient) GetRemovedImages() []*imagename.ImageName {
	return client.removedImages
}

// Pin resolves versions for given containers
func (client *DockerClient) Pin(local, hub bool, vars template.Vars, containers []*Container) error {
	return client.resolveVersions(local, hub, vars, containers)
}

// Internal

func (client *DockerClient) listenReAttach(containers []*Container) {
	// The code is partially borrowed from https://github.com/jwilder/docker-gen
	eventChan := make(chan *docker.APIEvents, 100)
	defer close(eventChan)

	if err := client.Docker.AddEventListener(eventChan); err != nil {
		log.Errorf("Failed to start listening for Docker events, error: %s", err)
		return
	}

	for {
		// TODO: we can reconnect here
		if err := client.Docker.Ping(); err != nil {
			log.Errorf("Unable to ping docker daemon: %s", err)
			return
		}

		select {

		case event := <-eventChan:
			if event == nil {
				log.Errorf("Got nil event from Docker API")
				return
			}

			// Filter out events which we are not interested in
			var container *Container
			for _, c := range containers {
				if c.ID == event.ID {
					container = c
					break
				}
			}

			if container != nil {
				log.Infof("Container %s (%.12s) - %s", container.Name, container.ID, event.Status)
			}

			// We are interested only in "start" events here
			if event.Status != "start" {
				break
			}

			go func(event *docker.APIEvents) {
				inspect, err := client.Docker.InspectContainer(event.ID)
				if err != nil {
					log.Errorf("Failed to inspect container %.12s, error: %s", event.ID, err)
					return
				}
				eventContainer, err := NewContainerFromDocker(inspect)
				if err != nil {
					// Ignore ErrNotRockerCompose error
					if _, ok := err.(config.ErrNotRockerCompose); !ok {
						log.Errorf("Failed to init container %.12s from Docker API, error: %s", event.ID, err)
					}
					return
				}
				// Look for such container in the namespace
				var container *Container
				for _, c := range containers {
					if c.IsSameKind(eventContainer) {
						container = c
						container.ID = eventContainer.ID
						break
					}
				}
				if container == nil {
					return
				}

				log.Infof("Container %s (%.12s) - %s", container.Name, container.ID, event.Status)

				// For running containers, in case it is started or restarted, we want to re-attach
				if !container.State.Running {
					return
				}
				if err := client.AttachToContainer(container); err != nil {
					log.Errorf("Failed to re-attach to the container %s (%.12s), error %s", container.Name, container.ID, err)
					return
				}

			}(event)

		case <-time.After(10 * time.Second):
			// check for docker liveness
		}
	}
}

func (client *DockerClient) flushContainerLogs(container *Container) {
	log.Debugf("Flush logs for container %s", container.Name)

	if container.Io == nil {
		container.Io = NewContainerIo(container)
	}

	err2 := client.Docker.Logs(docker.LogsOptions{
		Container:    container.Name.String(),
		OutputStream: container.Io.Stdout,
		ErrorStream:  container.Io.Stderr,
		Stdout:       true,
		Stderr:       true,
	})
	if err2 != nil {
		log.Errorf("Failed to read logs of container %s, error: %s", container.Name, err2)
	}
}

// pullImageForContainers goes through all containers and inspects their images
// it pulls images if they cannot be found locally or forceUpdate flag is set to true
func (client *DockerClient) pullImageForContainers(forceUpdate bool, vars template.Vars, containers ...*Container) (err error) {

	if err := client.resolveVersions(true, forceUpdate, vars, containers); err != nil {
		return err
	}

	var (
		img    *docker.Image
		pulled = map[string]*docker.Image{}
	)

	// check images for each container
	for _, container := range containers {
		if container.Image == nil {
			err = fmt.Errorf("Cannot find image for container %s", container.Name)
			return
		}
		// already pulled it for other container, skip
		if img, ok := pulled[container.Image.String()]; ok {
			container.ImageID = img.ID
			continue
		}

		isSha := container.Image.TagIsSha()

		if img, err = client.Docker.InspectImage(container.Image.String()); err == docker.ErrNoSuchImage || (forceUpdate && !isSha) {
			log.Infof("Pulling image: %s for %s", container.Image, container.Name)
			if img, err = PullDockerImage(client.Docker, container.Image, client.Auth); err != nil {
				err = fmt.Errorf("Failed to pull image %s for container %s, error: %s", container.Image, container.Name, err)
				return
			}
			client.pulledImages = append(client.pulledImages, container.Image)
		}
		if err != nil {
			return
		}

		container.ImageID = img.ID
		pulled[container.Image.String()] = img
	}

	return
}

// resolveVersions walks through the list of images and resolves their tags in case they are not strict
func (client *DockerClient) resolveVersions(local, hub bool, vars template.Vars, containers []*Container) (err error) {

	// Provide function getter of all images to fetch only once
	var available []*imagename.ImageName
	getImages := func() ([]*imagename.ImageName, error) {
		if available == nil {
			available = []*imagename.ImageName{}

			if !local {
				return available, nil
			}

			// retrieving images currently available in docker
			var dockerImages []docker.APIImages
			if dockerImages, err = client.Docker.ListImages(docker.ListImagesOptions{}); err != nil {
				return nil, err
			}

			for _, image := range dockerImages {
				for _, repoTag := range image.RepoTags {
					available = append(available, imagename.NewFromString(repoTag))
				}
			}
		}
		return available, nil
	}

	resolved := map[string]*imagename.ImageName{}

	// check images for each container
	for _, container := range containers {
		// error in configuration, fail fast
		if container.Image == nil {
			err = fmt.Errorf("Image is not specified for the container: %s", container.Name)
			return
		}

		// Version specified in variables
		var k string
		k = fmt.Sprintf("v_image_%s", container.Image.NameWithRegistry())
		if tag, ok := vars[k]; ok {
			log.Infof("Resolve %s --> %s (derived by variable %s)", container.Image, tag, k)
			container.Image.SetTag(tag.(string))
		}
		k = fmt.Sprintf("v_container_%s", container.Name.Name)
		if tag, ok := vars[k]; ok {
			log.Infof("Resolve %s --> %s (derived by variable %s)", container.Image, tag, k)
			container.Image.SetTag(tag.(string))
		}

		// Do not resolve anything if the image is strict, e.g. "redis:2.8.11" or "redis:latest"
		if container.Image.IsStrict() {
			continue
		}

		// already resolved it for other container
		if _, ok := resolved[container.Image.String()]; ok {
			container.Image = resolved[container.Image.String()]
			continue
		}

		// Override to not change the common images slice
		var images []*imagename.ImageName
		if images, err = getImages(); err != nil {
			return err
		}

		// looking locally first
		candidate := container.Image.ResolveVersion(images, true)

		// in case we want to include external images as well, pulling list of available
		// images from repository or central docker hub
		if hub || candidate == nil {
			log.Debugf("Getting list of tags for %s from the registry", container.Image)

			var remote []*imagename.ImageName

			if container.Image.Storage == imagename.StorageS3 {
				s3storage := s3.New(client.Docker, os.TempDir())
				remote, err = s3storage.ListTags(container.Image.String())
			} else {
				remote, err = dockerclient.RegistryListTags(container.Image, client.Auth)
			}

			if err != nil {
				return fmt.Errorf("Failed to list tags of image %s for container %s from the remote registry, error: %s",
					container.Image, container.Name, err)
			}

			log.Debugf("remote: %v", remote)

			// Re-Resolve having hub tags
			candidate = container.Image.ResolveVersion(append(images, remote...), false)
		}

		if candidate == nil {
			err = fmt.Errorf("Image not found: %s", container.Image)
			return
		}
		candidate.IsOldS3Name = container.Image.IsOldS3Name

		log.Infof("Resolve %s --> %s", container.Image, candidate.GetTag())

		container.Image = candidate
		resolved[container.Image.String()] = candidate
	}

	return
}
