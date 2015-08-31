/*-
 * Copyright 2014 Grammarly, Inc.
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
	"os"
	"util"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/pkg/term"
	"github.com/fsouza/go-dockerclient"
	"github.com/grammarly/rocker/src/rocker/imagename"
)

const emptyImageName = "gliderlabs/alpine:3.2"

type DockerClientConfig struct {
	Host      string
	Tlsverify bool
	Tlscacert string
	Tlscert   string
	Tlskey    string
}

func NewDockerClientConfig() *DockerClientConfig {
	certPath := util.StringOr(os.Getenv("DOCKER_CERT_PATH"), "~/.docker")
	return &DockerClientConfig{
		Host:      os.Getenv("DOCKER_HOST"),
		Tlsverify: os.Getenv("DOCKER_TLS_VERIFY") == "1" || os.Getenv("DOCKER_TLS_VERIFY") == "yes",
		Tlscacert: certPath + "/ca.pem",
		Tlscert:   certPath + "/cert.pem",
		Tlskey:    certPath + "/key.pem",
	}
}

func NewDockerClient() (*docker.Client, error) {
	return NewDockerClientFromConfig(NewDockerClientConfig())
}

func NewDockerClientFromConfig(config *DockerClientConfig) (*docker.Client, error) {
	if config.Tlsverify {
		return docker.NewTLSClient(config.Host, config.Tlscert, config.Tlskey, config.Tlscacert)
	}
	return docker.NewClient(config.Host)
}

// GetBridgeIp gets the ip address of docker network bridge
// it is useful when you want to loose couple containers and not have tightly link them
// container A may publish port 8125 to host network and container B may access this port through
// a bridge ip address; it's a hacky solution, any better way to obtain bridge ip without ssh access
// to host machine is welcome
//
// Here we create a dummy container and look at .NetworkSettings.Gateway value
//
// https://github.com/docker/docker/issues/1143
// https://github.com/docker/docker/issues/11247
//
func GetBridgeIp(client *docker.Client) (ip string, err error) {
	// Ensure empty image existing
	_, err = client.InspectImage(emptyImageName)
	if err != nil && err.Error() == "no such image" {
		log.Infof("Pulling image %s to obtain network bridge address", emptyImageName)
		if err := PullDockerImage(client, imagename.New(emptyImageName), &docker.AuthConfiguration{}); err != nil {
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

func PullDockerImage(client *docker.Client, image *imagename.ImageName, auth *docker.AuthConfiguration) error {
	pipeReader, pipeWriter := io.Pipe()

	pullOpts := docker.PullImageOptions{
		Repository:    image.NameWithRegistry(),
		Registry:      image.Registry,
		Tag:           image.Tag,
		OutputStream:  pipeWriter,
		RawJSONStream: true,
	}

	errch := make(chan error, 1)

	go func() {
		err := client.PullImage(pullOpts, *auth)

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
		return fmt.Errorf("Failed to process json stream for image: %s, error: %s", image, err)
	}

	if err := <-errch; err != nil {
		return fmt.Errorf("Failed to pull image %s, error: %s", image, err)
	}

	return nil
}
