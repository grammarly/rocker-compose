package compose

import (
	"strings"
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
	// println(string(data))
	// }

}

func TestClientRunContainer(t *testing.T) {
	t.Skip()

	dockerCli, err := NewDockerClient()
	if err != nil {
		t.Fatal(err)
	}

	cli, err := NewClient(&ClientCfg{Docker: dockerCli, Global: false})
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

	config, err := ReadConfig("test.yml", strings.NewReader(yml), map[string]interface{}{})
	if err != nil {
		t.Fatal(err)
	}

	for _, container := range config.GetContainers() {
		if err := cli.RunContainer(container); err != nil {
			t.Fatal(err)
		}
	}
}
