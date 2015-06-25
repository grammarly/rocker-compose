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
	Expose          []PortBinding     ``
	PublishAllPorts *bool             ``
	Labels          map[string]string ``
	VolumesFrom     []ContainerName   `` // TODO: may be referred to another compose namespace
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

func (config *Config) GetContainers() []*Container {
	containers := make([]*Container, len(config.Containers))
	i := 0
	for name, containerConfig := range config.Containers {
		containerName := NewContainerName(config.Namespace, name)
		containers[i] = NewContainerFromConfig(containerName, containerConfig)
		i++
	}
	return containers
}

func NewConfigFromDocker(apiContainer *docker.Container) *ConfigContainer {
	return &ConfigContainer{}
}

func (config *ConfigContainer) ToApiConfig() *docker.Config {
	return &docker.Config{}
}

func (a *ConfigContainer) IsEqualTo(b *ConfigContainer) bool {
	// Compare simple values
	if a.Image != b.Image ||
		a.Net != b.Net ||
		a.Pid != b.Pid ||
		a.Uts != b.Uts ||
		a.Restart != b.Restart ||
		a.Memory != b.Memory ||
		a.MemorySwap != b.MemorySwap ||
		a.CpusetCpus != b.CpusetCpus {
		return false
	}

	// Compare pointer values
	if !comparePointerInt64(a.CpuShares, b.CpuShares) ||
		!comparePointerBool(a.OomKillDisable, b.OomKillDisable) ||
		!comparePointerBool(a.Privileged, b.Privileged) ||
		!comparePointerBool(a.PublishAllPorts, b.PublishAllPorts) ||
		!comparePointerInt(a.KillTimeout, b.KillTimeout) {
		return false
	}

	// Compare slices and maps by length first
	if len(a.Dns) != len(b.Dns) ||
		len(a.AddHost) != len(b.AddHost) ||
		len(a.Ulimits) != len(b.Ulimits) ||
		len(a.Cmd) != len(b.Cmd) ||
		len(a.Entrypoint) != len(b.Entrypoint) ||
		len(a.Expose) != len(b.Expose) ||
		len(a.Labels) != len(b.Labels) ||
		len(a.VolumesFrom) != len(b.VolumesFrom) ||
		len(a.Volumes) != len(b.Volumes) {
		return false
	}

	// Compare simple slices
	for i := 0; i < len(a.Dns); i++ {
		if a.Dns[i] != b.Dns[i] {
			return false
		}
	}
	for i := 0; i < len(a.AddHost); i++ {
		if a.AddHost[i] != b.AddHost[i] {
			return false
		}
	}
	for i := 0; i < len(a.Cmd); i++ {
		if a.Cmd[i] != b.Cmd[i] {
			return false
		}
	}
	for i := 0; i < len(a.Entrypoint); i++ {
		if a.Entrypoint[i] != b.Entrypoint[i] {
			return false
		}
	}
	for i := 0; i < len(a.Expose); i++ {
		if a.Expose[i] != b.Expose[i] {
			return false
		}
	}
	for i := 0; i < len(a.VolumesFrom); i++ {
		if a.VolumesFrom[i] != b.VolumesFrom[i] {
			return false
		}
	}
	for i := 0; i < len(a.Volumes); i++ {
		if a.Volumes[i] != b.Volumes[i] {
			return false
		}
	}

	// Compare pointer slices
	for i := 0; i < len(a.Ulimits); i++ {
		if *a.Ulimits[i] != *b.Ulimits[i] {
			return false
		}
	}

	// Compare maps
	for k, v := range a.Labels {
		if v != b.Labels[k] {
			return false
		}
	}

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

// Helper functions to compare pointer values used by ContainerConfig.IsEqualTo function

func comparePointerInt64(a, b *int64) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func comparePointerInt(a, b *int) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func comparePointerBool(a, b *bool) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}
