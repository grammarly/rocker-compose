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
	Image           string            ``                         // e.g. docker run <IMAGE>
	Extends         string            ``                         // can extend from other container spec referring by name
	Net             string            ``                         // e.g. docker run --net
	Pid             string            ``                         // e.g. docker run --pid
	Uts             string            ``                         // NOT WORKING, TODO: find in docker remote api
	State           ConfigState       ``                         // "running" or "created"
	Dns             []string          ``                         // e.g. docker run --dns
	AddHost         []string          `yaml:"add_host"`          // e.g. docker run --add-host
	Restart         RestartPolicy     ``                         // e.g. docker run --restart
	Memory          ConfigMemory      ``                         // e.g. docker run --memory
	MemorySwap      ConfigMemory      `yaml:"memory_swap"`       // e.g. docker run --swap
	CpuShares       *int64            `yaml:"cpu_shares"`        // e.g. docker run --cpu-shares
	CpusetCpus      string            `yaml:"cpuset_cpus"`       // e.g. docker run --cpuset-cpus
	OomKillDisable  *bool             `yaml:"oom_kill_disable"`  // e.g. docker run --oom-kill-disable TODO: pull request to go-dockerclient
	Ulimits         []ConfigUlimit    ``                         // search by "Ulimits" here https://goo.gl/IxbZck
	Privileged      *bool             ``                         // e.g. docker run --privileged
	Cmd             []string          ``                         // e.g. docker run <IMAGE> <CMD>
	Entrypoint      []string          ``                         // e.g. docker run --entrypoint
	Expose          []string          ``                         // e.g. docker run --expose
	Ports           []PortBinding     ``                         // e.g. docker run --expose
	PublishAllPorts *bool             `yaml:"publish_all_ports"` // e.g. docker run -P
	Labels          map[string]string ``                         // e.g. docker run --label
	Env             map[string]string ``                         //
	VolumesFrom     []ContainerName   `yaml:"volumes_from"`      // TODO: may be referred to another compose namespace
	Volumes         []string          ``                         //
	Links           []ContainerName   ``                         // TODO: may be referred to another compose namespace
	KillTimeout     *int              `yaml:"kill_timeout"`      //
	Hostname        string            ``                         //
	Domainname      string            ``                         //
	User            string            ``                         //
	Workdir         string            ``                         //
	NetworkDisabled *bool             `yaml:"network_disabled"`  //
	KeepVolumes     *bool             `yaml:"keep_volumes"`      //

	lastCompareField string
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

func (a *ConfigContainer) LastCompareField() string {
	return a.lastCompareField
}

func (a *ConfigContainer) IsEqualTo(b *ConfigContainer) bool {
	// Compare simple values

	a.lastCompareField = "Image"
	if a.Image != b.Image {
		return false
	}

	a.lastCompareField = "Net"
	if a.Net != b.Net {
		return false
	}

	a.lastCompareField = "Pid"
	if a.Pid != b.Pid {
		return false
	}

	a.lastCompareField = "Uts"
	if a.Uts != b.Uts {
		return false
	}

	a.lastCompareField = "Restart"
	if a.Restart != b.Restart {
		return false
	}

	a.lastCompareField = "CpusetCpus"
	if a.CpusetCpus != b.CpusetCpus {
		return false
	}

	a.lastCompareField = "Hostname"
	if a.Hostname != b.Hostname {
		return false
	}

	a.lastCompareField = "Domainname"
	if a.Domainname != b.Domainname {
		return false
	}

	a.lastCompareField = "User"
	if a.User != b.User {
		return false
	}

	a.lastCompareField = "Workdir"
	if a.Workdir != b.Workdir {
		return false
	}

	// Comparable objects

	a.lastCompareField = "State"
	if !a.State.IsEqualTo(b.State) {
		return false
	}

	a.lastCompareField = "Memory"
	if !a.Memory.IsEqualTo(b.Memory) {
		return false
	}

	a.lastCompareField = "MemorySwap"
	if !a.MemorySwap.IsEqualTo(b.MemorySwap) {
		return false
	}

	// Compare pointer values

	a.lastCompareField = "CpuShares"
	if !comparePointerInt64(a.CpuShares, b.CpuShares) {
		return false
	}

	a.lastCompareField = "OomKillDisable"
	if !comparePointerBool(a.OomKillDisable, b.OomKillDisable) {
		return false
	}

	a.lastCompareField = "Privileged"
	if !comparePointerBool(a.Privileged, b.Privileged) {
		return false
	}

	a.lastCompareField = "PublishAllPorts"
	if !comparePointerBool(a.PublishAllPorts, b.PublishAllPorts) {
		return false
	}

	a.lastCompareField = "NetworkDisabled"
	if !comparePointerBool(a.NetworkDisabled, b.NetworkDisabled) {
		return false
	}

	a.lastCompareField = "KeepVolumes"
	if !comparePointerBool(a.KeepVolumes, b.KeepVolumes) {
		return false
	}

	// Compare slices

	a.lastCompareField = "Dns"
	if !compareSliceString(a.Dns, b.Dns) {
		return false
	}

	a.lastCompareField = "AddHost"
	if !compareSliceString(a.AddHost, b.AddHost) {
		return false
	}

	a.lastCompareField = "Cmd"
	if !compareSliceString(a.Cmd, b.Cmd) {
		return false
	}

	a.lastCompareField = "Entrypoint"
	if !compareSliceString(a.Entrypoint, b.Entrypoint) {
		return false
	}

	a.lastCompareField = "Expose"
	if !compareSliceString(a.Expose, b.Expose) {
		return false
	}

	a.lastCompareField = "Volumes"
	if !compareSliceString(a.Volumes, b.Volumes) {
		return false
	}

	a.lastCompareField = "Ulimits"
	if !compareSliceUlimit(a.Ulimits, b.Ulimits) {
		return false
	}

	a.lastCompareField = "Ports"
	if !compareSlicePortBinding(a.Ports, b.Ports) {
		return false
	}

	a.lastCompareField = "VolumesFrom"
	if !compareSliceContainerName(a.VolumesFrom, b.VolumesFrom) {
		return false
	}

	a.lastCompareField = "Links"
	if !compareSliceContainerName(a.Links, b.Links) {
		return false
	}

	// Compare maps
	a.lastCompareField = "Labels"
	if len(a.Labels) != len(b.Labels) {
		return false
	}
	for k, v := range a.Labels {
		if v != b.Labels[k] {
			return false
		}
	}

	a.lastCompareField = "Env"
	if len(a.Env) != len(b.Env) {
		return false
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
	if container.State == (ConfigState)("") {
		container.State = parent.State
	}
	if container.Dns == nil {
		container.Dns = parent.Dns
	}
	if container.AddHost == nil {
		container.AddHost = parent.AddHost
	}
	if container.Restart == (RestartPolicy{}) {
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
	if container.KeepVolumes == nil {
		container.KeepVolumes = parent.KeepVolumes
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

func NewConfigMemoryFromInt64(value int64) ConfigMemory {
	return (ConfigMemory)(fmt.Sprintf("%db", value))
}

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

func (a ConfigMemory) IsEqualTo(b ConfigMemory) bool {
	return a.Int64() == b.Int64()
}

type RestartPolicy struct {
	Name              string
	MaximumRetryCount int
}

func (r *RestartPolicy) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var name string
	if err := unmarshal(&name); err != nil {
		return err
	}
	if name == "" {
		r.Name = "no"
	} else if name == "always" {
		r.Name = "always"
	} else if strings.Index(name, "on-failure") == 0 {
		r.Name = "on-failure"
		parts := strings.SplitN(name, ",", 2)
		n, err := strconv.ParseInt(parts[1], 10, 16)
		if err != nil {
			return err
		}
		r.MaximumRetryCount = (int)(n)
	}
	return nil
}

func (r RestartPolicy) ToDockerApi() docker.RestartPolicy {
	return docker.RestartPolicy{
		Name:              r.Name,
		MaximumRetryCount: r.MaximumRetryCount,
	}
}

type PortBinding struct {
	Port     string
	HostIp   string
	HostPort string
}

func (b *PortBinding) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var value string
	if err := unmarshal(&value); err != nil {
		return err
	}
	split := strings.SplitN(value, ":", 3)
	if len(split) == 3 {
		b.Port = split[2]
		b.HostIp = split[0]
		b.HostPort = split[1]
	} else if len(split) == 2 {
		b.Port = split[1]
		b.HostPort = split[0]
	} else {
		b.Port = split[0]
	}
	return nil
}

type ConfigState string

func NewConfigStateFromBool(value bool) ConfigState {
	if value {
		return (ConfigState)("running")
	} else {
		return (ConfigState)("created")
	}
}

func (a ConfigState) IsEqualTo(b ConfigState) bool {
	if a == "" {
		a = "running"
	}
	if b == "" {
		b = "running"
	}
	return a == b
}

func (state ConfigState) RunningBool() bool {
	if state == "running" {
		return true
	} else if state == "created" {
		return false
	}
	return true
}

// Helper functions to compare pointer values used by ContainerConfig.IsEqualTo function

func comparePointerInt64(a, b *int64) bool {
	if a == nil {
		return b == a || *b == 0
	}
	if b == nil {
		return a == b || *a == 0
	}
	return *a == *b
}

func comparePointerInt(a, b *int) bool {
	if a == nil {
		return b == a || *b == 0
	}
	if b == nil {
		return a == b || *a == 0
	}
	return *a == *b
}

func comparePointerBool(a, b *bool) bool {
	if a == nil {
		return b == a || *b == false
	}
	if b == nil {
		return a == b || *a == false
	}
	return *a == *b
}

// Here we duplicate functions changing only argument types
// sadly, there is no way to do it better in Go

func compareSliceString(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	found := true
	for i := 0; i < len(a); i++ {
		elFound := false
		for k := 0; k < len(b); k++ {
			if a[i] == b[k] {
				elFound = true
				break
			}
		}
		if !elFound {
			found = false
			break
		}
	}
	return found
}

func compareSliceUlimit(a, b []ConfigUlimit) bool {
	if len(a) != len(b) {
		return false
	}
	found := true
	for i := 0; i < len(a); i++ {
		elFound := false
		for k := 0; k < len(b); k++ {
			if a[i] == b[k] {
				elFound = true
				break
			}
		}
		if !elFound {
			found = false
			break
		}
	}
	return found
}

func compareSlicePortBinding(a, b []PortBinding) bool {
	if len(a) != len(b) {
		return false
	}
	found := true
	for i := 0; i < len(a); i++ {
		elFound := false
		for k := 0; k < len(b); k++ {
			if a[i] == b[k] {
				elFound = true
				break
			}
		}
		if !elFound {
			found = false
			break
		}
	}
	return found
}

func compareSliceContainerName(a, b []ContainerName) bool {
	if len(a) != len(b) {
		return false
	}
	found := true
	for i := 0; i < len(a); i++ {
		elFound := false
		for k := 0; k < len(b); k++ {
			if a[i] == b[k] {
				elFound = true
				break
			}
		}
		if !elFound {
			found = false
			break
		}
	}
	return found
}
