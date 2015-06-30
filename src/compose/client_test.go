package compose

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewClient(t *testing.T) {
	cli, err := NewClient(&ClientCfg{})
	if err != nil {
		t.Fatal(err)
	}

	assert.IsType(t, &ClientCfg{}, cli)
}

func TestClientGetContainers(t *testing.T) {
	dockerCli, err := NewDockerClient()
	if err != nil {
		t.Fatal(err)
	}

	cli, err := NewClient(&ClientCfg{Docker: dockerCli, Global: false})
	if err != nil {
		t.Fatal(err)
	}

	containers, err := cli.GetContainers()
	if err != nil {
		t.Fatal(err)
	}

	assert.IsType(t, []*Container{}, containers)

	// assert.IsType(t, &docker.Env{}, info)
	// fmt.Printf("Containers: %+q\n", containers)
	// pretty.Println(containers)
	// for _, container := range containers {
	// 	data, err := yaml.Marshal(container.Config)
	// 	if err != nil {
	// 		t.Fatal(err)
	// 	}
	// 	println(string(data))
	// }

}
