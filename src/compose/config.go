package compose

import (
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/fsouza/go-dockerclient"
	"github.com/go-yaml/yaml"
)

// Config represents the data structure which is loaded from compose.yml
type Config struct {
	Namespace  string // All containers names under current compose.yml will be prefixed with this namespace
	Containers map[string]*ConfigContainer
}

// ConfigContainer represents a single container spec from compose.yml
type ConfigContainer struct {
	Image           string            ``                        // e.g. docker run <IMAGE>
	Extends         string            ``                        // can extend from other container spec referring by name
	Net             string            ``                        // e.g. docker run --net
	Pid             string            ``                        // e.g. docker run --pid
	Uts             string            ``                        // NOT WORKING, TODO: find in docker remote api
	State           string            ``                        // "running" or "created"
	Dns             []string          ``                        // e.g. docker run --dns
	AddHost         []string          `yaml:"add_host"`         // e.g. docker run --add-host
	Restart         RestartPolicy     ``                        // e.g. docker run --restart
	Memory          ConfigMemory      ``                        // e.g. docker run --memory
	MemorySwap      ConfigMemory      `yaml:"memory_swap"`      // e.g. docker run --swap
	CpuShares       *int64            `yaml:"cpu_shares"`       // e.g. docker run --cpu-shares
	CpusetCpus      string            `yaml:"cpuset_cpus"`      // e.g. docker run --cpuset-cpus
	OomKillDisable  *bool             `yaml:"oom_kill_disable"` // e.g. docker run --oom-kill-disable TODO: pull request to go-dockerclient
	Ulimits         []ConfigUlimit    ``                        // search by "Ulimits" here https://goo.gl/IxbZck
	Privileged      *bool             ``                        // e.g. docker run --privileged
	Cmd             []string          ``                        // e.g. docker run <IMAGE> <CMD>
	Entrypoint      []string          ``                        // e.g. docker run --entrypoint
	Expose          []string          ``                        // e.g. docker run --expose
	Ports           []PortBinding     ``                        // e.g. docker run --expose
	PublishAllPorts *bool             ``                        // e.g. docker run -P
	Labels          map[string]string ``                        // e.g. docker run --label
	Env             map[string]string ``                        //
	VolumesFrom     []ContainerName   `yaml:"volumes_from"`     // TODO: may be referred to another compose namespace
	Volumes         []string          ``                        //
	Links           []ContainerName   ``                        // TODO: may be referred to another compose namespace
	KillTimeout     *int              `yaml:"kill_timeout"`     //
	Hostname        string            ``                        //
	Domainname      string            ``                        //
	User            string            ``                        //
	Workdir         string            ``                        //
	NetworkDisabled *bool             `yaml:"network_disabled"` //
}

type ConfigUlimit struct {
	Name string
	Soft int64
	Hard int64
}

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

func (a *ConfigContainer) IsEqualTo(b *ConfigContainer) bool {
	// Compare simple values
	if a.Image != b.Image ||
		a.Net != b.Net ||
		a.Pid != b.Pid ||
		a.Uts != b.Uts ||
		a.State != b.State ||
		a.Restart != b.Restart ||
		a.Memory != b.Memory ||
		a.MemorySwap != b.MemorySwap ||
		a.CpusetCpus != b.CpusetCpus ||
		a.Hostname != b.Hostname ||
		a.Domainname != b.Domainname ||
		a.User != b.User ||
		a.Workdir != b.Workdir {
		return false
	}

	// Compare pointer values
	if !comparePointerInt64(a.CpuShares, b.CpuShares) ||
		!comparePointerBool(a.OomKillDisable, b.OomKillDisable) ||
		!comparePointerBool(a.Privileged, b.Privileged) ||
		!comparePointerBool(a.PublishAllPorts, b.PublishAllPorts) ||
		!comparePointerBool(a.NetworkDisabled, b.NetworkDisabled) ||
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
		len(a.Ports) != len(b.Ports) ||
		len(a.Labels) != len(b.Labels) ||
		len(a.Env) != len(b.Env) ||
		len(a.Links) != len(b.Links) ||
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
	for i := 0; i < len(a.Ports); i++ {
		if a.Ports[i] != b.Ports[i] {
			return false
		}
	}
	for i := 0; i < len(a.Links); i++ {
		if a.Links[i] != b.Links[i] {
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
		if a.Ulimits[i] != b.Ulimits[i] {
			return false
		}
	}

	// Compare maps
	for k, v := range a.Labels {
		if v != b.Labels[k] {
			return false
		}
	}
	for k, v := range a.Env {
		if v != b.Env[k] {
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
	if container.State == "" {
		container.State = parent.State
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
	if container.Ports == nil {
		container.Ports = parent.Ports
	}
	if container.PublishAllPorts == nil {
		container.PublishAllPorts = parent.PublishAllPorts
	}
	if container.NetworkDisabled == nil {
		container.NetworkDisabled = parent.NetworkDisabled
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

	// Extend env
	newEnv := make(map[string]string)
	for k, v := range parent.Env {
		newEnv[k] = v
	}
	for k, v := range container.Env {
		newEnv[k] = v
	}
	container.Env = newEnv

	if container.Links == nil {
		container.Links = parent.Links
	}
	if container.VolumesFrom == nil {
		container.VolumesFrom = parent.VolumesFrom
	}
	if container.Volumes == nil {
		container.Volumes = parent.Volumes
	}
	if container.KillTimeout == nil {
		container.KillTimeout = parent.KillTimeout
	}
	if container.Hostname == "" {
		container.Hostname = parent.Hostname
	}
	if container.Domainname == "" {
		container.Domainname = parent.Domainname
	}
	if container.User == "" {
		container.User = parent.User
	}
	if container.Workdir == "" {
		container.Workdir = parent.Workdir
	}

	return
}

func (container *ConfigContainer) StateRunningBool() bool {
	if container.State == "running" || container.State == "" {
		return true
	} else if container.State == "created" {
		return false
	}
	return true
}

func ReadConfigFile(filename string, vars map[string]interface{}) (*Config, error) {
	fd, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("Failed to open config file %s, error: %s", filename, err)
	}
	defer fd.Close()

	return ReadConfig(filename, fd, vars)
}

func ReadConfig(name string, reader io.Reader, vars map[string]interface{}) (*Config, error) {
	config := &Config{}

	data, err := ProcessConfigTemplate(name, reader, vars)
	if err != nil {
		return nil, fmt.Errorf("Failed to process config template, error: %s", err)
	}

	if err := yaml.Unmarshal(data.Bytes(), config); err != nil {
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

		// Set namespace for all containers inside
		for k, name := range container.VolumesFrom {
			container.VolumesFrom[k] = *name.DefaultNamespace(config.Namespace)
		}
		for k, name := range container.Links {
			container.Links[k] = *name.DefaultNamespace(config.Namespace)
		}
	}

	return config, nil
}

// Other minor types

type ConfigMemory string

func (m ConfigMemory) Int64() (value int64) {
	var t string
	_, err := fmt.Sscanf(string(m), "%d%s", &value, &t)
	if err != nil {
		_, err := fmt.Sscanf(string(m), "%d", &value)
		if err != nil {
			return 0
		}
	}
	for idx, ct := range []string{"b", "k", "m", "g"} {
		if ct == strings.ToLower(t) {
			value = value * (int64)(math.Pow(1024, (float64)(idx)))
			break
		}
	}
	return value
}

type RestartPolicy string

func (r RestartPolicy) ToDockerApi() docker.RestartPolicy {
	if r == "" {
		return docker.RestartPolicy{}
	} else if r == "always" {
		return docker.AlwaysRestart()
	} else if strings.Index(string(r), "on-failure") == 0 {
		parts := strings.SplitN(string(r), ",", 2)
		n, err := strconv.ParseInt(parts[1], 10, 16)
		if err == nil {
			return docker.RestartOnFailure((int)(n))
		}
	}
	return docker.NeverRestart()
}

type PortBinding string

func (p PortBinding) Parse() (port, hostIp, hostPort string) {
	// format: ip:hostPort:containerPort | ip::containerPort | hostPort:containerPort | containerPort
	split := strings.SplitN(string(p), ":", 3)
	if len(split) == 3 {
		port = split[2]
		hostIp = split[0]
		hostPort = split[1]
	} else if len(split) == 2 {
		port = split[1]
		hostPort = split[0]
	} else {
		port = split[0]
	}
	return port, hostIp, hostPort
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
