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
	"github.com/snkozlov/rocker-compose/src/compose/config"
	"strings"
	"testing"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/fsouza/go-dockerclient"
	"github.com/grammarly/rocker/src/dockerclient"
	"github.com/grammarly/rocker/src/imagename"
	"github.com/grammarly/rocker/src/rocker/test"
	"github.com/grammarly/rocker/src/template"
	"github.com/kr/pretty"
	"github.com/stretchr/testify/assert"
)

func TestNewClient(t *testing.T) {
	cli, err := NewClient(&DockerClient{})
	if err != nil {
		t.Fatal(err)
	}

	assert.IsType(t, &DockerClient{}, cli)
}

func TestClientGetContainers(t *testing.T) {
	// TODO: mock?
	t.Skip()
	log.SetOutput(test.Writer("client: ", t))

	dockerCli, err := dockerclient.New()
	if err != nil {
		t.Fatal(err)
	}

	cli, err := NewClient(&DockerClient{Docker: dockerCli})
	if err != nil {
		t.Fatal(err)
	}

	containers, err := cli.GetContainers(false)
	if err != nil {
		t.Fatal(err)
	}

	assert.IsType(t, []*Container{}, containers)
}

func TestClientRunContainer(t *testing.T) {
	t.Skip()

	dockerCli, err := dockerclient.New()
	if err != nil {
		t.Fatal(err)
	}

	cli, err := NewClient(&DockerClient{Docker: dockerCli})
	if err != nil {
		t.Fatal(err)
	}

	yml := `
namespace: test
containers:
  main:
    image: "busybox:buildroot-2013.08.1"
    labels:
      foo: bar
      xxx: yyy
`

	config, err := config.ReadConfig("test.yml", strings.NewReader(yml), map[string]interface{}{}, map[string]interface{}{}, false)
	if err != nil {
		t.Fatal(err)
	}

	for _, container := range GetContainersFromConfig(config) {
		if err := cli.RunContainer(container); err != nil {
			t.Fatal(err)
		}
	}
}

func TestClientClean(t *testing.T) {
	// This test involves interaction with docker
	// enable it in case you want to test Clean() functionality
	t.Skip()

	dockerCli, err := dockerclient.New()
	if err != nil {
		t.Fatal(err)
	}

	// Create number of images to test

	createdContainers := []string{}
	createdImages := []string{}

	defer func() {
		for _, id := range createdContainers {
			if err := dockerCli.RemoveContainer(docker.RemoveContainerOptions{ID: id, Force: true}); err != nil {
				t.Error(err)
			}
		}
		for _, id := range createdImages {
			if err := dockerCli.RemoveImageExtended(id, docker.RemoveImageOptions{Force: true}); err != nil {
				if err.Error() == "no such image" {
					continue
				}
				t.Error(err)
			}
		}
	}()

	for i := 1; i <= 5; i++ {
		c, err := dockerCli.CreateContainer(docker.CreateContainerOptions{
			Config: &docker.Config{
				Image: "gliderlabs/alpine:3.1",
				Cmd:   []string{"true"},
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		createdContainers = append(createdContainers, c.ID)

		commitOpts := docker.CommitContainerOptions{
			Container:  c.ID,
			Repository: "rocker-compose-test-image-clean",
			Tag:        fmt.Sprintf("%d", i),
		}

		img, err := dockerCli.CommitContainer(commitOpts)
		if err != nil {
			t.Fatal(err)
		}

		createdImages = append(createdImages, img.ID)

		// Make sure images have different timestamps
		time.Sleep(time.Second)
	}

	////////////////////////

	cli, err := NewClient(&DockerClient{Docker: dockerCli, KeepImages: 2})
	if err != nil {
		t.Fatal(err)
	}

	yml := `
namespace: test
containers:
  main:
    image: rocker-compose-test-image-clean:5
`

	config, err := config.ReadConfig("test.yml", strings.NewReader(yml), map[string]interface{}{}, map[string]interface{}{}, false)
	if err != nil {
		t.Fatal(err)
	}

	if err := cli.Clean(config); err != nil {
		t.Fatal(err)
	}

	// test that images left

	all, err := dockerCli.ListImages(docker.ListImagesOptions{})
	if err != nil {
		t.Fatal(err)
	}

	n := 0
	for _, image := range all {
		for _, repoTag := range image.RepoTags {
			imageName := imagename.NewFromString(repoTag)
			if imageName.Name == "rocker-compose-test-image-clean" {
				n++
			}
		}
	}

	assert.Equal(t, 2, n, "Expected images to be cleaned up")

	// test removed images list

	removed := cli.GetRemovedImages()
	assert.Equal(t, 3, len(removed), "Expected to remove a particular number of images")

	assert.EqualValues(t, "rocker-compose-test-image-clean:3", removed[0].String(), "removed wrong image")
	assert.EqualValues(t, "rocker-compose-test-image-clean:2", removed[1].String(), "removed wrong image")
	assert.EqualValues(t, "rocker-compose-test-image-clean:1", removed[2].String(), "removed wrong image")
}

func TestClientResolveVersions(t *testing.T) {
	t.Skip()

	dockerCli, err := dockerclient.New()
	if err != nil {
		t.Fatal(err)
	}

	client, err := NewClient(&DockerClient{
		Docker: dockerCli,
	})
	if err != nil {
		t.Fatal(err)
	}

	containers := []*Container{
		&Container{
			Name:  config.NewContainerName("test", "test"),
			Image: imagename.NewFromString("golang:1.4.*"),
		},
	}

	if err := client.resolveVersions(true, true, template.Vars{}, containers); err != nil {
		t.Fatal(err)
	}

	pretty.Println(containers)
}
