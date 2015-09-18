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
	"compose/config"
	"fmt"
	"time"
	"util"

	"github.com/grammarly/rocker/src/rocker/imagename"
	"github.com/kr/pretty"

	log "github.com/Sirupsen/logrus"
	"github.com/fsouza/go-dockerclient"
)

// Client interface describes a rocker-compose client that can do various operations
// needed for rocker-compose to make changes.
type Client interface {
	GetContainers() ([]*Container, error)
	RemoveContainer(container *Container) error
	RunContainer(container *Container) error
	EnsureContainerExist(name *Container) error
	EnsureContainerState(name *Container) error
	PullAll(config *config.Config) error
	Clean(config *config.Config) error
	AttachToContainers(container []*Container) error
	AttachToContainer(container *Container) error
	FetchImages(containers []*Container) error
	WaitForContainer(container *Container) error
	GetPulledImages() []*imagename.ImageName
	GetRemovedImages() []*imagename.ImageName
}

// DockerClient is an implementation of Client interface that do operations to a given docker client
type DockerClient struct {
	Docker     *docker.Client
	Global     bool // Search for existing containers globally, not only ones started with compose
	Attach     bool
	Wait       time.Duration
	Auth       *AuthConfig
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

// AuthConfig is a docker auth configuration object
type AuthConfig struct {
	Username      string
	Password      string
	Email         string
	ServerAddress string
}

// ToDockerAPI converts AuthConfig to be eatable by go-dockerclient
func (a *AuthConfig) ToDockerAPI() *docker.AuthConfiguration {
	if a == nil {
		return &docker.AuthConfiguration{}
	}
	return &docker.AuthConfiguration{
		Username:      a.Username,
		Password:      a.Password,
		Email:         a.Email,
		ServerAddress: a.ServerAddress,
	}
}

// NewClient makes a new DockerClient object based on configuration params
// that is given with input DockerClient object.
func NewClient(initialClient *DockerClient) (Client, error) {
	client := &DockerClient{
		Docker:     initialClient.Docker,
		Global:     initialClient.Global,
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
func (client *DockerClient) GetContainers() ([]*Container, error) {
	filters := map[string][]string{}
	if !client.Global {
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
	numResponses := 0

	for _, apiContainer := range apiContainers {
		go func(apiContainer docker.APIContainers) {
			chResponse := new(chResponse)
			chResponse.container, chResponse.err = client.Docker.InspectContainer(apiContainer.ID)
			ch <- chResponse
		}(apiContainer)
	}

	log.Infof("Gathering info about %d containers", len(apiContainers))

	for {
		select {
		case resp := <-ch:
			numResponses++
			if resp.err != nil {
				return nil, fmt.Errorf("Failed to fetch container, error: %s", resp.err)
			}
			container, err := NewContainerFromDocker(resp.container)
			if err != nil {
				return nil, fmt.Errorf("Failed to initialize config container instance from docker api, error: %s", err)
			}
			containers = append(containers, container)

		case <-time.After(30 * time.Second):
			// todo: you may have to use client.Timeout
			return nil, fmt.Errorf("Timeout while fetching containers")
		}

		if len(apiContainers) == numResponses {
			break
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
	if err := client.Docker.StartContainer(container.ID, container.Config.GetAPIHostConfig()); err != nil {
		if !client.Attach {
			client.flushContainerLogs(container)
		}
		return fmt.Errorf("Failed to start container, error: %s", err)
	}

	if container.Config.State.IsRan() {
		// TODO: refactor to use DockerClient.WaitForContainer() ?
		exitCode, err := client.Docker.WaitContainer(container.Name.String())
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
func (client *DockerClient) PullAll(config *config.Config) error {
	// do not pull same image twice
	pulledImages := map[string]struct{}{}
	containers := GetContainersFromConfig(config)

	// do pull
	for _, container := range containers {
		imageName := container.Image.String()
		if _, ok := pulledImages[imageName]; ok {
			continue
		}
		if err := client.pullImageForContainer(container); err != nil {
			return err
		}
		pulledImages[imageName] = struct{}{}
	}
	return nil
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
			imageName := imagename.New(repoTag)
			for img := range images {
				if img.IsSameKind(*imageName) {
					images[img].Items = append(images[img].Items, &imagename.Tag{
						Id:      image.ID,
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
func (client *DockerClient) FetchImages(containers []*Container) error {
	type message struct {
		container *Container
		result    chan error
	}

	wg := util.NewErrorWaitGroup(len(containers))
	chPullImages := make(chan message)
	done := make(chan struct{}, 1)
	checkedImages := map[string]struct{}{}

	// Pull worker
	// We do not want to pull images in parallel
	// instead, we use an actor to pull images sequentially
	//
	// TODO: WAAAT? we can simplify it by going though a list of images sequentially
	// 			 and pull those images which are missing.
	go func() {
		for {
			select {
			case msg := <-chPullImages:
				msg.result <- client.pullImageForContainer(msg.container)
			case <-done:
				return
			}
		}
	}()

	defer func() {
		done <- struct{}{}
	}()

	for _, container := range containers {
		if container.Image == nil {
			return fmt.Errorf("Image is not specified for container %s", container.Name)
		}
		if _, ok := checkedImages[container.Image.String()]; ok {
			wg.Done(nil)
			continue
		}
		checkedImages[container.Image.String()] = struct{}{}

		go func(container *Container) {
			image, err := client.Docker.InspectImage(container.Image.String())
			if err == docker.ErrNoSuchImage {
				msg := message{container, make(chan error)}
				chPullImages <- msg
				wg.Done(<-msg.result)
				return
			} else if err != nil {
				wg.Done(err)
				return
			}
			container.ImageID = image.ID
			wg.Done(nil)
		}(container)
	}

	return wg.Wait()
}

// GetPulledImages returns the list of images pulled by a recent run
func (client *DockerClient) GetPulledImages() []*imagename.ImageName {
	return client.pulledImages
}

// GetRemovedImages returns the list of images removed by a recent run
func (client *DockerClient) GetRemovedImages() []*imagename.ImageName {
	return client.removedImages
}

// Internal

func (client *DockerClient) pullImageForContainer(container *Container) error {
	if container.Image == nil {
		return fmt.Errorf("Image is not specified for container: %s", container.Name)
	}

	log.Infof("Pulling image: %s for %s", container.Image, container.Name)

	if err := PullDockerImage(client.Docker, container.Image, client.Auth.ToDockerAPI()); err != nil {
		return fmt.Errorf("Failed to pull image %s for container %s, error: %s", container.Image, container.Name, err)
	}

	client.pulledImages = append(client.pulledImages, container.Image)

	return nil
}

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
