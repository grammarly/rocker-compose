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
	Image           string            ``
	Extends         string            ``
	Net             string            ``
	Pid             string            ``
	Uts             string            `` // TODO: find in docker remote api
	Dns             []string          ``
	AddHost         []string          `yaml:"add_host"`
	Restart         string            ``
	Memory          ConfigMemory      ``
	MemorySwap      ConfigMemorySwap  `yaml:"memory_swap"`
	CpuShares       *int64            `yaml:"cpu_shares"`
	CpusetCpus      string            `yaml:"cpuset_cpus"`
	OomKillDisable  *bool             `yaml:"oom_kill_disable"` // TODO: pull request to go-dockerclient
	Ulimits         []*ConfigUlimit   ``
	Privileged      *bool             ``
	Cmd             []string          ``
	Entrypoint      []string          ``
	Expose          []*PortBinding    ``
	PublishAllPorts *bool             ``
	Labels          map[string]string ``
	VolumesFrom     []string          `` // TODO: may be referred to another compose namespace
	Volumes         []string          ``
	KillTimeout     *int              `yaml:"kill_timeout"`
}

type ConfigUlimit struct {
	Name string
	Soft int64
	Hard int64
}

type PortBinding string

type ConfigMemory string

type ConfigMemorySwap string

func NewConfigFromApiConfig(apiConfig *docker.Config) *ConfigContainer {
	return &ConfigContainer{}
}

func (config *ConfigContainer) ToApiConfig() *docker.Config {
	return &docker.Config{}
}

func (config *ConfigContainer) IsEqualTo(config2 *ConfigContainer) bool {
	return true
}

func (container *ConfigContainer) ExtendFrom(parent *ConfigContainer) {
	if container.Image == "" {
		container.Image = parent.Image
	}
	if container.Net == "" {
		container.Net = parent.Net
	}
	if container.Pid == "" {
		container.Pid = parent.Pid
	}
	if container.Uts == "" {
		container.Uts = parent.Uts
	}
	if container.Dns == nil {
		container.Dns = parent.Dns
	}
	if container.AddHost == nil {
		container.AddHost = parent.AddHost
	}
	if container.Restart == "" {
		container.Restart = parent.Restart
	}
	if container.Memory == "" {
		container.Memory = parent.Memory
	}
	if container.MemorySwap == "" {
		container.MemorySwap = parent.MemorySwap
	}
	if container.MemorySwap == "" {
		container.MemorySwap = parent.MemorySwap
	}
	if container.CpuShares == nil {
		container.CpuShares = parent.CpuShares
	}
	if container.CpusetCpus == "" {
		container.CpusetCpus = parent.CpusetCpus
	}
	if container.OomKillDisable == nil {
		container.OomKillDisable = parent.OomKillDisable
	}
	if container.Ulimits == nil {
		container.Ulimits = parent.Ulimits
	}
	if container.Privileged == nil {
		container.Privileged = parent.Privileged
	}
	if container.Cmd == nil {
		container.Cmd = parent.Cmd
	}
	if container.Entrypoint == nil {
		container.Entrypoint = parent.Entrypoint
	}
	if container.Expose == nil {
		container.Expose = parent.Expose
	}
	if container.PublishAllPorts == nil {
		container.PublishAllPorts = parent.PublishAllPorts
	}
	// Extend labels
	newLabels := make(map[string]string)
	for k, v := range parent.Labels {
		newLabels[k] = v
	}
	for k, v := range container.Labels {
		newLabels[k] = v
	}
	container.Labels = newLabels

	if container.VolumesFrom == nil {
		container.VolumesFrom = parent.VolumesFrom
	}
	if container.Volumes == nil {
		container.Volumes = parent.Volumes
	}
	if container.KillTimeout == nil {
		container.KillTimeout = parent.KillTimeout
	}

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

	// Process extending containers configuration
	for name, container := range config.Containers {
		if container.Extends != "" {
			if container.Extends == name {
				return nil, fmt.Errorf("Container %s: cannot extend from itself", name)
			}
			if _, ok := config.Containers[container.Extends]; !ok {
				return nil, fmt.Errorf("Container %s: cannot find container %s to extend from", name, container.Extends)
			}
			// TODO: build dependency graph by extends hierarchy to allow multiple inheritance
			if config.Containers[container.Extends].Extends != "" {
				return nil, fmt.Errorf("Container %s: cannot extend from %s: multiple inheritance is not allowed yet",
					name, container.Extends)
			}
			container.ExtendFrom(config.Containers[container.Extends])
		}
	}

	return config, nil
}
