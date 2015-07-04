package config

import (
	"fmt"
	"io"
	"math"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/fsouza/go-dockerclient"
	"github.com/go-yaml/yaml"
)

// Config represents the data structure which is loaded from compose.yml
type Config struct {
	Namespace  string // All containers names under current compose.yml will be prefixed with this namespace
	Containers map[string]*Container
}

// Container represents a single container spec from compose.yml
type Container struct {
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
	Cmd             *ConfigCmd        `yaml:"cmd,omitempty"`               // e.g. docker run <IMAGE> <CMD>
	Entrypoint      []string          `yaml:"entrypoint,omitempty"`        // e.g. docker run --entrypoint
	Expose          []string          `yaml:"expose,omitempty"`            // e.g. docker run --expose
	Ports           []PortBinding     `yaml:"ports,omitempty"`             // e.g. docker run --expose
	PublishAllPorts *bool             `yaml:"publish_all_ports,omitempty"` // e.g. docker run -P
	Labels          map[string]string `yaml:"labels,omitempty"`            // e.g. docker run --label
	Env             map[string]string `yaml:"env,omitempty"`               //
	VolumesFrom     []ContainerName   `yaml:"volumes_from,omitempty"`      // TODO: may be referred to another compose namespace
	Volumes         []string          `yaml:"volumes,omitempty"`           //
	Links           []ContainerName   `yaml:"links,omitempty"`             // TODO: may be referred to another compose namespace
	WaitFor         []ContainerName   `yaml:"wait_for,omitempty"`          //
	KillTimeout     *uint             `yaml:"kill_timeout,omitempty"`      //
	Hostname        *string           `yaml:"hostname,omitempty"`          //
	Domainname      *string           `yaml:"domainname,omitempty"`        //
	User            *string           `yaml:"user,omitempty"`              //
	Workdir         *string           `yaml:"workdir,omitempty"`           //
	NetworkDisabled *bool             `yaml:"network_disabled,omitempty"`  //
	KeepVolumes     *bool             `yaml:"keep_volumes,omitempty"`      //

	lastCompareField string
}

type ContainerName struct {
	Namespace string
	Name      string
	Alias     string
}

type ConfigUlimit struct {
	Name string
	Soft int64
	Hard int64
}

type ConfigMemory int64

type RestartPolicy struct {
	Name              string
	MaximumRetryCount int
}

// format: ip:hostPort:containerPort | ip::containerPort | hostPort:containerPort | containerPort
type PortBinding struct {
	Port     string
	HostIp   string
	HostPort string
}

type ConfigState string

type ConfigCmd struct {
	Parts []string
}

func NewFromFile(filename string, vars map[string]interface{}) (*Config, error) {
	fd, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("Failed to open config file %s, error: %s", filename, err)
	}
	defer fd.Close()

	config, err := ReadConfig(filename, fd, vars)
	if err != nil {
		return nil, err
	}

	// Process relative paths in volumes
	dir := filepath.Dir(filename)
	if !path.IsAbs(dir) {
		wd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		dir = path.Join(wd, dir)
	}

	for _, container := range config.Containers {
		for i, volume := range container.Volumes {
			split := strings.SplitN(volume, ":", 2)
			if len(split) == 1 {
				continue
			}
			if strings.HasPrefix(split[0], "~") {
				split[0] = strings.Replace(split[0], "~", os.Getenv("HOME"), 1)
			}
			if !path.IsAbs(split[0]) {
				split[0] = path.Join(dir, split[0])
			}
			container.Volumes[i] = strings.Join(split, ":")
		}
	}

	return config, nil
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
		for k, name := range container.WaitFor {
			container.WaitFor[k] = *name.DefaultNamespace(config.Namespace)
		}
	}

	return config, nil
}

// Other minor types functions

// Constructors

func NewContainerName(namespace, name string) *ContainerName {
	return &ContainerName{namespace, name, ""}
}

// format: name | namespace.name | name:alias | namespace.name:alias
func NewContainerNameFromString(str string) *ContainerName {
	containerName := &ContainerName{}
	str = strings.TrimPrefix(str, "/") // TODO: investigate why Docker adds prefix slash to container names
	split := strings.SplitN(str, ".", 2)
	if len(split) > 1 {
		containerName.Namespace = split[0]
		containerName.Name = split[1]
	} else {
		containerName.Name = split[0]
	}
	split2 := strings.SplitN(containerName.Name, ":", 2)
	if len(split2) > 1 {
		containerName.Name = split2[0]
		containerName.Alias = split2[1]
	} else {
		containerName.Name = split2[0]
	}
	return containerName
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

// Methods

func (containerName ContainerName) String() string {
	name := containerName.Name
	if containerName.Namespace != "" {
		name = fmt.Sprintf("%s.%s", containerName.Namespace, name)
	}
	if containerName.Alias != "" {
		name = fmt.Sprintf("%s:%s", name, containerName.Alias)
	}
	return name
}

func (a *ContainerName) DefaultNamespace(ns string) *ContainerName {
	newContainerName := *a // copy object
	if newContainerName.Namespace == "" {
		newContainerName.Namespace = ns
	}
	return &newContainerName
}

func (m *ConfigMemory) Int64() int64 {
	if m == nil {
		return 0
	}
	return (int64)(*m)
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

func (state *ConfigState) Bool() bool {
	if state != nil {
		return *state == "running"
	}
	return true // "running" or anything else
}

func (state *ConfigState) IsRan() bool {
	return state != nil && *state == "ran"
}
