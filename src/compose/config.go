package compose

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/fsouza/go-dockerclient"
	"github.com/go-yaml/yaml"
)

// TODO: more config fields!
type Config struct {
	Namespace  string
	Containers map[string]*ConfigContainer
}

type ConfigContainer struct {
	Image           string
	Extends         string
	Net             string
	Pid             string
	Uts             string // TODO: find in docker remote api
	Dns             []string
	AddHost         []string `yaml:"add_host"`
	Restart         string
	Memory          string
	MemorySwap      string `yaml:"memory_swap"`
	CpuShares       int64  `yaml:"cpu_shares"`
	CpusetCpus      string `yaml:"cpuset_cpus"`
	OomKillDisable  string `yaml:"oom_kill_disable"` // TODO: pull request to go-dockerclient
	Ulimits         []ConfigUlimit
	Privileged      bool
	Cmd             []string
	Entrypoint      []string
	Expose          []PortBinding
	PublishAllPorts bool
	Labels          map[string]string
	VolumesFrom     []string // TODO: may be referred to another compose namespace
	Volumes         []string
	KillTimeout     int `yaml:"kill_timeout"`
}

type ConfigUlimit struct {
	Name string
	Soft int64
	Hard int64
}

type PortBinding string

func NewConfigFromApiConfig(apiConfig *docker.Config) *ConfigContainer {
	return &ConfigContainer{}
}

func (config *ConfigContainer) ToApiConfig() *docker.Config {
	return &docker.Config{}
}

func (config *ConfigContainer) IsEqualTo(config2 *ConfigContainer) bool {
	return true
}

func (config *ConfigContainer) ExtendFrom(parent *ConfigContainer) {
	return
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

	// TODO: make extends

	return config, nil
}
