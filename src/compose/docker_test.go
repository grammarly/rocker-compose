package compose

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/fsouza/go-dockerclient"
	"github.com/stretchr/testify/assert"
)

func TestNewDockerClient(t *testing.T) {
	cli, err := NewDockerClient()
	if err != nil {
		t.Fatal(err)
	}

	info, err := cli.Info()
	if err != nil {
		t.Fatal(err)
	}

	assert.IsType(t, &docker.Env{}, info)
}

func TestEntrypointOverride(t *testing.T) {
	cli, err := NewDockerClient()
	if err != nil {
		t.Fatal(err)
	}

	container, err := cli.CreateContainer(docker.CreateContainerOptions{
		Config: &docker.Config{
			Image:        "test-entrypoint-override",
			Entrypoint:   []string{},
			Cmd:          []string{"/bin/ls"},
			AttachStdout: true,
			AttachStderr: true,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := cli.RemoveContainer(docker.RemoveContainerOptions{ID: container.ID, Force: true}); err != nil {
			t.Fatal(err)
		}
	}()

	success := make(chan struct{})
	var buf bytes.Buffer

	attachOpts := docker.AttachToContainerOptions{
		Container:    container.ID,
		OutputStream: &buf,
		ErrorStream:  &buf,
		Stream:       true,
		Stdout:       true,
		Stderr:       true,
		Success:      success,
	}
	go func() {
		if err := cli.AttachToContainer(attachOpts); err != nil {
			t.Fatal(fmt.Errorf("Attach container error: %s", err))
		}
	}()

	success <- <-success

	if err := cli.StartContainer(container.ID, &docker.HostConfig{}); err != nil {
		t.Fatal(fmt.Errorf("Failed to start container, error: %s", err))
	}

	statusCode, err := cli.WaitContainer(container.ID)
	if err != nil {
		t.Fatal(fmt.Errorf("Wait container error: %s", err))
	}

	println(buf.String())

	if statusCode != 0 {
		t.Fatal(fmt.Errorf("Failed to run container, exit with code %d", statusCode))
	}
}
