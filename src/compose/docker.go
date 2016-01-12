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
	"io"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/pkg/term"
	"github.com/fsouza/go-dockerclient"
	"github.com/grammarly/rocker/src/rocker/dockerclient"
	"github.com/grammarly/rocker/src/rocker/imagename"
)

const emptyImageName = "gliderlabs/alpine:3.2"

// GetBridgeIP gets the ip address of docker network bridge
// it is useful when you want to loose couple containers and not have tightly link them
// container A may publish port 8125 to host network and container B may access this port through
// a bridge ip address; it's a hacky solution, any better way to obtain bridge ip without ssh access
// to host machine is welcome
//
// Here we create a dummy container and look at .NetworkSettings.Gateway value
//
// TODO: maybe we don't need this anymore since docker 1.8 seem to specify all existing containers
// 			 in a /etc/hosts file of every contianer. Need to research it further.
//
// https://github.com/docker/docker/issues/1143
// https://github.com/docker/docker/issues/11247
//
func GetBridgeIP(client *docker.Client) (ip string, err error) {
	// Ensure empty image existing
	_, err = client.InspectImage(emptyImageName)
	if err != nil && err.Error() == "no such image" {
		log.Infof("Pulling image %s to obtain network bridge address", emptyImageName)
		if _, err := PullDockerImage(client, imagename.NewFromString(emptyImageName), nil); err != nil {
			return "", err
		}
	} else if err != nil {
		return "", fmt.Errorf("Failed to inspect image %s, error: %s", emptyImageName, err)
	}

	container, err := client.CreateContainer(docker.CreateContainerOptions{
		Config: &docker.Config{
			Image: emptyImageName,
			Cmd:   []string{"/bin/sh", "-c", "while true; do sleep 1; done"},
		},
		HostConfig: &docker.HostConfig{},
	})
	if err != nil {
		return "", fmt.Errorf("Failed to create dummy network container, error: %s", err)
	}
	defer func() {
		removeOpts := docker.RemoveContainerOptions{
			ID:            container.ID,
			Force:         true,
			RemoveVolumes: true,
		}
		if err2 := client.RemoveContainer(removeOpts); err2 != nil && err == nil {
			err = err2
		}
	}()

	if err := client.StartContainer(container.ID, &docker.HostConfig{}); err != nil {
		return "", fmt.Errorf("Failed to start dummy network container %.12s, error: %s", container.ID, err)
	}

	inspect, err := client.InspectContainer(container.ID)
	if err != nil {
		return "", fmt.Errorf("Failed to inspect dummy network container %.12s, error: %s", container.ID, err)
	}

	return inspect.NetworkSettings.Gateway, nil
}

// PullDockerImage pulls an image and streams to a logger respecting terminal features
func PullDockerImage(client *docker.Client, image *imagename.ImageName, auth *docker.AuthConfigurations) (*docker.Image, error) {
	pipeReader, pipeWriter := io.Pipe()

	pullOpts := docker.PullImageOptions{
		Repository:    image.NameWithRegistry(),
		Registry:      image.Registry,
		Tag:           image.Tag,
		OutputStream:  pipeWriter,
		RawJSONStream: true,
	}

	repoAuth := dockerclient.GetAuthForRegistry(auth, image.Registry)
	errch := make(chan error, 1)

	go func() {
		err := client.PullImage(pullOpts, repoAuth)

		if err := pipeWriter.Close(); err != nil {
			log.Errorf("Failed to close pull image stream for %s, error: %s", image, err)
		}

		errch <- err
	}()

	def := log.StandardLogger()
	fd, isTerminal := term.GetFdInfo(def.Out)
	out := def.Out

	if !isTerminal {
		out = def.Writer()
	}

	if err := jsonmessage.DisplayJSONMessagesStream(pipeReader, out, fd, isTerminal); err != nil {
		return nil, fmt.Errorf("Failed to process json stream for image: %s, error: %s", image, err)
	}

	if err := <-errch; err != nil {
		return nil, fmt.Errorf("Failed to pull image %s, error: %s", image, err)
	}

	img, err := client.InspectImage(image.String())
	if err != nil {
		return nil, fmt.Errorf("Failed to inspect image %s after pull, error: %s", image, err)
	}

	return img, nil
}
