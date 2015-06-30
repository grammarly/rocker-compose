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
	Extends         string            `yaml:"extends,omitempty"`           // can extend from other container spec referring by name
	Image           *string           `yaml:"image,omitempty"`             // e.g. docker run <IMAGE>
	Net             *string           `yaml:"net,omitempty"`               // e.g. docker run --net
	Pid             *string           `yaml:"pid,omitempty"`               // e.g. docker run --pid
	Uts             *string           `yaml:"uts,omitempty"`               // NOT WORKING, TODO: find in docker remote api
	State           *ConfigState      `yaml:"state,omitempty"`             // "running" or "created"
	Dns             []string          `yaml:"dns,omitempty"`               // e.g. docker run --dns
	AddHost         []string          `yaml:"add_host,omitempty"`          // e.g. docker run --add-host
	Restart         *RestartPolicy    `yaml:"restart,omitempty"`           // e.g. docker run --restart
	Memory          *ConfigMemory     `yaml:"memory,omitempty"`            // e.g. docker run --memory
	MemorySwap      *ConfigMemory     `yaml:"memory_swap,omitempty"`       // e.g. docker run --swap
	CpuShares       *int64            `yaml:"cpu_shares,omitempty"`        // e.g. docker run --cpu-shares
	CpusetCpus      *string           `yaml:"cpuset_cpus,omitempty"`       // e.g. docker run --cpuset-cpus
	OomKillDisable  *bool             `yaml:"oom_kill_disable,omitempty"`  // e.g. docker run --oom-kill-disable TODO: pull request to go-dockerclient
	Ulimits         []ConfigUlimit    `yaml:"ulimits,omitempty"`           // search by "Ulimits" here https://goo.gl/IxbZck
	Privileged      *bool             `yaml:"privileged,omitempty"`        // e.g. docker run --privileged
	Cmd             []string          `yaml:"cmd,omitempty"`               // e.g. docker run <IMAGE> <CMD>
	Entrypoint      []string          `yaml:"entrypoint,omitempty"`        // e.g. docker run --entrypoint
	Expose          []string          `yaml:"expose,omitempty"`            // e.g. docker run --expose
	Ports           []PortBinding     `yaml:"ports,omitempty"`             // e.g. docker run --expose
	PublishAllPorts *bool             `yaml:"publish_all_ports,omitempty"` // e.g. docker run -P
	Labels          map[string]string `yaml:"labels,omitempty"`            // e.g. docker run --label
	Env             map[string]string `yaml:"env,omitempty"`               //
	VolumesFrom     []ContainerName   `yaml:"volumes_from,omitempty"`      // TODO: may be referred to another compose namespace
	Volumes         []string          `yaml:"volumes,omitempty"`           //
	Links           []ContainerName   `yaml:"links,omitempty"`             // TODO: may be referred to another compose namespace
	KillTimeout     *uint             `yaml:"kill_timeout,omitempty"`      //
	Hostname        *string           `yaml:"hostname,omitempty"`          //
	Domainname      *string           `yaml:"domainname,omitempty"`        //
	User            *string           `yaml:"user,omitempty"`              //
	Workdir         *string           `yaml:"workdir,omitempty"`           //
	NetworkDisabled *bool             `yaml:"network_disabled,omitempty"`  //
	KeepVolumes     *bool             `yaml:"keep_volumes,omitempty"`      //

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
	if !comparePointerString(a.Image, b.Image) {
		return false
	}

	a.lastCompareField = "Net"
	if !comparePointerString(a.Net, b.Net) {
		return false
	}

	a.lastCompareField = "Pid"
	if !comparePointerString(a.Pid, b.Pid) {
		return false
	}

	a.lastCompareField = "Uts"
	if !comparePointerString(a.Uts, b.Uts) {
		return false
	}

	a.lastCompareField = "CpusetCpus"
	if !comparePointerString(a.CpusetCpus, b.CpusetCpus) {
		return false
	}

	a.lastCompareField = "Hostname"
	if !comparePointerString(a.Hostname, b.Hostname) {
		return false
	}

	a.lastCompareField = "Domainname"
	if !comparePointerString(a.Domainname, b.Domainname) {
		return false
	}

	a.lastCompareField = "User"
	if !comparePointerString(a.User, b.User) {
		return false
	}

	a.lastCompareField = "Workdir"
	if !comparePointerString(a.Workdir, b.Workdir) {
		return false
	}

	// Compare RestartPolicy
	a.lastCompareField = "Restart"
	if !comparePointerRestart(a.Restart, b.Restart) {
		return false
	}

	// Comparable objects

	a.lastCompareField = "State"
	if !a.State.IsEqualTo(b.State) {
		return false
	}

	a.lastCompareField = "Memory"
	if !comparePointerMemory(a.Memory, b.Memory) {
		return false
	}

	a.lastCompareField = "MemorySwap"
	if !comparePointerMemory(a.MemorySwap, b.MemorySwap) {
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
	if !compareStringMap(a.Labels, b.Labels) {
		return false
	}

	a.lastCompareField = "Env"
	if !compareStringMap(a.Env, b.Env) {
		return false
	}

	return true
}

func (container *ConfigContainer) ExtendFrom(parent *ConfigContainer) {
	if container.Image == nil {
		container.Image = parent.Image
	}
	if container.Net == nil {
		container.Net = parent.Net
	}
	if container.Pid == nil {
		container.Pid = parent.Pid
	}
	if container.Uts == nil {
		container.Uts = parent.Uts
	}
	if container.State == nil {
		container.State = parent.State
	}
	if container.Dns == nil {
		container.Dns = parent.Dns
	}
	if container.AddHost == nil {
		container.AddHost = parent.AddHost
	}
	if container.Restart == nil {
		container.Restart = parent.Restart
	}
	if container.Memory == nil {
		container.Memory = parent.Memory
	}
	if container.MemorySwap == nil {
		container.MemorySwap = parent.MemorySwap
	}
	if container.CpuShares == nil {
		container.CpuShares = parent.CpuShares
	}
	if container.CpusetCpus == nil {
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
	if container.Hostname == nil {
		container.Hostname = parent.Hostname
	}
	if container.Domainname == nil {
		container.Domainname = parent.Domainname
	}
	if container.User == nil {
		container.User = parent.User
	}
	if container.Workdir == nil {
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

type ConfigMemory int64

func (m *ConfigMemory) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var str string
	if err := unmarshal(&str); err != nil {
		return err
	}
	value, err := NewConfigMemoryFromString(str)
	if err != nil {
		return err
	}
	*m = *value

	return nil
}

func (m *ConfigMemory) Int64() int64 {
	if m == nil {
		return 0
	}
	return (int64)(*m)
}

func NewConfigMemoryFromString(str string) (*ConfigMemory, error) {
	var (
		value int64
		t     string
	)

	if str == "" {
		return nil, nil
	}

	if _, err := fmt.Sscanf(str, "%d%s", &value, &t); err != nil {
		if _, err := fmt.Sscanf(str, "%d", &value); err != nil {
			return nil, err
		}
	}
	for idx, ct := range []string{"b", "k", "m", "g"} {
		if ct == strings.ToLower(t) {
			value = value * (int64)(math.Pow(1024, (float64)(idx)))
			break
		}
	}

	memory := (ConfigMemory)(value)
	return &memory, nil
}

func NewConfigMemoryFromInt64(value int64) *ConfigMemory {
	if value == 0 {
		return nil
	}
	memory := (ConfigMemory)(value)
	return &memory
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

func (r *RestartPolicy) MarshalYAML() (interface{}, error) {
	if r == nil || r.Name == "" {
		return "no", nil
	} else if r.Name == "always" {
		return "always", nil
	} else if r.Name == "on-failure" {
		return fmt.Sprintf("on-failure,%d", r.MaximumRetryCount), nil
	}
	return "no", nil
}

func (r *RestartPolicy) ToDockerApi() docker.RestartPolicy {
	if r == nil {
		return docker.RestartPolicy{}
	}
	return docker.RestartPolicy{
		Name:              r.Name,
		MaximumRetryCount: r.MaximumRetryCount,
	}
}

// format: ip:hostPort:containerPort | ip::containerPort | hostPort:containerPort | containerPort
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

func (b PortBinding) MarshalYAML() (interface{}, error) {
	if b.HostIp != "" && b.HostPort != "" {
		return fmt.Sprintf("%s:%s:%s", b.HostIp, b.HostPort, b.Port), nil
	} else if b.HostIp != "" {
		return fmt.Sprintf("%s::%s", b.HostIp, b.Port), nil
	} else if b.HostPort != "" {
		return fmt.Sprintf("%s:%s", b.HostPort, b.Port), nil
	}
	return b.Port, nil
}

type ConfigState string

func NewConfigStateFromBool(running bool) *ConfigState {
	var state ConfigState
	if running {
		state = (ConfigState)("running")
	} else {
		state = (ConfigState)("created")
	}
	return &state
}

func (a *ConfigState) IsEqualTo(b *ConfigState) bool {
	if a == nil {
		return b == a || *b == ""
	}
	if b == nil {
		return a == b || *a == ""
	}
	return *a == *b
}

func (state *ConfigState) RunningBool() bool {
	if state != nil && *state == "created" {
		return false
	}
	return true // "running" or anything else
}

// Helper functions to compare pointer values used by ContainerConfig.IsEqualTo function

func comparePointerString(a, b *string) bool {
	if a == nil {
		return b == a || *b == ""
	}
	if b == nil {
		return a == b || *a == ""
	}
	return *a == *b
}

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

func comparePointerRestart(a, b *RestartPolicy) bool {
	if a == nil {
		return b == a || *b == RestartPolicy{}
	}
	if b == nil {
		return a == b || *a == RestartPolicy{}
	}
	return *a == *b
}

func comparePointerMemory(a, b *ConfigMemory) bool {
	if a == nil {
		return b == a || *b == 0
	}
	if b == nil {
		return a == b || *a == 0
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

func compareStringMap(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if v != b[k] {
			return false
		}
	}
	return true
}
