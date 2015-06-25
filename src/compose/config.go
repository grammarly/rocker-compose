package compose

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/go-yaml/yaml"
)

// TODO: more config fields!
type Config struct {
	Containers map[string]*ContainerConfig
}

type ContainerConfig struct {
	Image string
}

func ReadConfigFile(filename string) (*Config, error) {
	fd, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("Failed to open config file %s, error: %s", filename, err)
	}
	defer fd.Close()

	return ReadConfig(fd)
}

func ReadConfig(reader io.Reader) (*Config, error) {
	config := &Config{}

	data, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("Failed read config, error: %s", err)
	}

	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("Failed to parse YAML config, error: %s", err)
	}

	return config, nil
}
