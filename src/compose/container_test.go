package compose

import (
	"testing"

	"github.com/fsouza/go-dockerclient"
	"github.com/stretchr/testify/assert"
)

var (
	containerTestVars = map[string]interface{}{
		"version": map[string]string{
			"patterns": "1.9.2",
		},
	}
)

// type T struct {
// 	Names []*Name
// }

// type Name struct {
// 	Value string
// }

// func (name *Name) MarshalYAML() (interface{}, error) {
// 	return name.Value, nil
// }

func TestCreateContainerOptions(t *testing.T) {
	// val := &T{
	// 	Names: []*Name{&Name{"asd"}},
	// }
	// data, err := yaml.Marshal(val)
	// if err != nil {
	// 	t.Fatal(err)
	// }

	// pretty.Println(string(data))

	config, err := ReadConfigFile("testdata/compose.yml", containerTestVars)
	if err != nil {
		t.Fatal(err)
	}

	container := NewContainerFromConfig(NewContainerName("patterns", "main"), config.Containers["main"])

	opts, err := container.CreateContainerOptions()
	if err != nil {
		t.Fatal(err)
	}

	// pretty.Println(opts.Config.Labels["rocker-compose-config"])

	assert.IsType(t, &docker.CreateContainerOptions{}, opts)
}
