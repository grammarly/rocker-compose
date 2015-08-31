/*-
 * Copyright 2014 Grammarly, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package config

import (
	"fmt"
	"io"
	"math"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/grammarly/rocker/src/rocker/imagename"

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
	Extends         string         `yaml:"extends,omitempty"`           // can extend from other container spec referring by name
	Image           *string        `yaml:"image,omitempty"`             // e.g. docker run <IMAGE>
	Net             *Net           `yaml:"net,omitempty"`               // e.g. docker run --net
	Pid             *string        `yaml:"pid,omitempty"`               // e.g. docker run --pid
	Uts             *string        `yaml:"uts,omitempty"`               // NOT WORKING, TODO: find in docker remote api
	State           *ConfigState   `yaml:"state,omitempty"`             // "running" or "created"
	Dns             Strings        `yaml:"dns,omitempty"`               // e.g. docker run --dns
	AddHost         Strings        `yaml:"add_host,omitempty"`          // e.g. docker run --add-host
	Restart         *RestartPolicy `yaml:"restart,omitempty"`           // e.g. docker run --restart
	Memory          *ConfigMemory  `yaml:"memory,omitempty"`            // e.g. docker run --memory
	MemorySwap      *ConfigMemory  `yaml:"memory_swap,omitempty"`       // e.g. docker run --swap
	CpuShares       *int64         `yaml:"cpu_shares,omitempty"`        // e.g. docker run --cpu-shares
	CpusetCpus      *string        `yaml:"cpuset_cpus,omitempty"`       // e.g. docker run --cpuset-cpus
	OomKillDisable  *bool          `yaml:"oom_kill_disable,omitempty"`  // e.g. docker run --oom-kill-disable TODO: pull request to go-dockerclient
	Ulimits         []ConfigUlimit `yaml:"ulimits,omitempty"`           // search by "Ulimits" here https://goo.gl/IxbZck
	Privileged      *bool          `yaml:"privileged,omitempty"`        // e.g. docker run --privileged
	Cmd             Cmd            `yaml:"cmd,omitempty"`               // e.g. docker run <IMAGE> <CMD>
	Entrypoint      Strings        `yaml:"entrypoint,omitempty"`        // e.g. docker run --entrypoint
	Expose          Strings        `yaml:"expose,omitempty"`            // e.g. docker run --expose
	Ports           Ports          `yaml:"ports,omitempty"`             // e.g. docker run --expose
	PublishAllPorts *bool          `yaml:"publish_all_ports,omitempty"` // e.g. docker run -P
	Labels          StringMap      `yaml:"labels,omitempty"`            // e.g. docker run --label
	Env             StringMap      `yaml:"env,omitempty"`               //
	VolumesFrom     ContainerNames `yaml:"volumes_from,omitempty"`      // TODO: may be referred to another compose namespace
	Volumes         Strings        `yaml:"volumes,omitempty"`           //
	Links           Links          `yaml:"links,omitempty"`             // TODO: may be referred to another compose namespace
	WaitFor         ContainerNames `yaml:"wait_for,omitempty"`          //
	KillTimeout     *uint          `yaml:"kill_timeout,omitempty"`      //
	Hostname        *string        `yaml:"hostname,omitempty"`          //
	Domainname      *string        `yaml:"domainname,omitempty"`        //
	User            *string        `yaml:"user,omitempty"`              //
	Workdir         *string        `yaml:"workdir,omitempty"`           //
	NetworkDisabled *bool          `yaml:"network_disabled,omitempty"`  // TODO: do we need this?
	KeepVolumes     *bool          `yaml:"keep_volumes,omitempty"`      //

	// Aliases, for compatibility with docker-compose and `docker run`
	Command     Cmd       `yaml:"command,omitempty"`
	Link        Links     `yaml:"link,omitempty"`
	Label       StringMap `yaml:"label,omitempty"`
	Hosts       Strings   `yaml:"hosts,omitempty"`
	WorkingDir  *string   `yaml:"working_dir,omitempty"`
	Environment StringMap `yaml:"environment,omitempty"`

	// Extra properties that is not known by rocker-compose
	Extra map[string]interface{} `yaml:"extra,omitempty"`

	lastCompareField string
}

type ContainerName struct {
	Namespace string
	Name      string
}

type Link struct {
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

type Net struct {
	Type      string // bridge|none|container|host
	Container ContainerName
}

type StringMap map[string]string
type ContainerNames []ContainerName
type Ports []PortBinding
type Links []Link

type Cmd []string

// type Volumes []string
// type Dns []string
// type Hosts []string
type Strings []string

func NewFromFile(filename string, vars map[string]interface{}, funcs map[string]interface{}) (*Config, error) {
	if !path.IsAbs(filename) {
		wd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("Cannot get absolute path to %s due to error %s", filename, err)
		}
		filename = path.Join(wd, filename)
	}

	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return nil, fmt.Errorf("No such file or directory: %s", filename)
	}

	fd, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("Failed to open config file %s, error: %s", filename, err)
	}
	defer fd.Close()

	config, err := ReadConfig(filename, fd, vars, funcs)
	if err != nil {
		return nil, err
	}

	return config, nil
}

func ReadConfig(configName string, reader io.Reader, vars map[string]interface{}, funcs map[string]interface{}) (*Config, error) {
	config := &Config{}

	basedir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("Failed to get working dir, error: %s", err)
	}

	if configName == "-" {
		configName = "<STDIN>"
	} else {
		// if file given, process volume paths relative to the manifest file
		basedir = filepath.Dir(configName)
	}

	data, err := ProcessConfigTemplate(configName, reader, vars, funcs)
	if err != nil {
		return nil, fmt.Errorf("Failed to process config template, error: %s", err)
	}

	if err := yaml.Unmarshal(data.Bytes(), config); err != nil {
		return nil, fmt.Errorf("Failed to parse YAML config, error: %s", err)
	}

	// Read extra data
	type ConfigExtra struct {
		Containers map[string]map[string]interface{}
	}
	extra := &ConfigExtra{}
	if err := yaml.Unmarshal(data.Bytes(), extra); err != nil {
		return nil, fmt.Errorf("Failed to parse YAML config extra properties, error: %s", err)
	}

	// Initialize YAML keys
	// Index yaml fields for better search
	yamlFields := make(map[string]bool)
	for _, v := range GetYamlFields() {
		yamlFields[v] = true
	}

	// Process aliases on the first run, have to do it before extends
	// because Golang randomizes maps, sometimes inherited containers
	// process earlier then dependencies; also do initial validation
	for name, container := range config.Containers {
		if container == nil {
			return nil, fmt.Errorf("Invalid specification for container `%s` in %s", name, configName)
		}
		// Handle aliases
		if container.Command != nil {
			if container.Cmd == nil {
				container.Cmd = container.Command
			}
			container.Command = nil
		}
		if container.Link != nil {
			if container.Links == nil {
				container.Links = container.Link
			}
			container.Link = nil
		}
		if container.Label != nil {
			if container.Labels == nil {
				container.Labels = container.Label
			}
			container.Label = nil
		}
		if container.Hosts != nil {
			if container.AddHost == nil {
				container.AddHost = container.Hosts
			}
			container.Hosts = nil
		}
		if container.WorkingDir != nil {
			if container.Workdir == nil {
				container.Workdir = container.WorkingDir
			}
			container.WorkingDir = nil
		}
		if container.Environment != nil {
			if container.Env == nil {
				container.Env = container.Environment
			}
			container.Environment = nil
		}

		// Process extra data
		extraFields := map[string]interface{}{}
		for key, val := range extra.Containers[name] {
			if !yamlFields[key] {
				extraFields[key] = val
			}
		}
		if len(extraFields) > 0 {
			container.Extra = extraFields
		}

		// pretty.Println(name, container.Extra)
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

		// Validate image
		if container.Image == nil {
			return nil, fmt.Errorf("Image should be specified for container: %s", name)
		}
		if !imagename.New(*container.Image).HasTag() {
			return nil, fmt.Errorf("Image `%s` for container `%s`: image without tag is not allowed",
				*container.Image, name)
		}

		// Set namespace for all containers inside
		for k := range container.VolumesFrom {
			container.VolumesFrom[k].DefaultNamespace(config.Namespace)
		}
		for k := range container.Links {
			container.Links[k].DefaultNamespace(config.Namespace)
		}
		for k := range container.WaitFor {
			container.WaitFor[k].DefaultNamespace(config.Namespace)
		}
		if container.Net != nil {
			if container.Net.Type == "container" {
				container.Net.Container.DefaultNamespace(config.Namespace)
			}
		}

		// Fix exposed ports
		for k, port := range container.Expose {
			if !strings.Contains(port, "/") {
				container.Expose[k] = port + "/tcp"
			}
		}

		// Process relative paths in volumes
		for i, volume := range container.Volumes {
			split := strings.SplitN(volume, ":", 2)
			if len(split) == 1 {
				continue
			}
			if strings.HasPrefix(split[0], "~") {
				split[0] = strings.Replace(split[0], "~", os.Getenv("HOME"), 1)
			}
			if !path.IsAbs(split[0]) {
				split[0] = path.Join(basedir, split[0])
			}
			container.Volumes[i] = strings.Join(split, ":")
		}
	}

	return config, nil
}

// Other minor types functions

// Constructors

func NewContainerName(namespace, name string) *ContainerName {
	return &ContainerName{namespace, name}
}

// format: name | namespace.name
func NewContainerNameFromString(str string) *ContainerName {
	containerName := &ContainerName{}
	str = strings.TrimPrefix(str, "/") // TODO: investigate why Docker adds prefix slash to container names
	split := strings.Split(str, ".")

	containerName.Name = split[len(split)-1]
	if len(split) > 1 {
		containerName.Namespace = strings.Join(split[:len(split)-1], ".")
	}

	return containerName
}

// format: name | namespace.name | name:alias | namespace.name:alias
func NewLinkFromString(str string) *Link {
	link := &Link{}
	split := strings.SplitN(str, ":", 2)

	containerName := NewContainerNameFromString(split[0])
	link.Name = containerName.Name
	link.Namespace = containerName.Namespace

	if len(split) > 1 {
		link.Alias = split[1]
	} else {
		link.Alias = link.Name
	}

	// convert underscores to dashes, because alias is used in hostnames
	link.Alias = strings.Replace(link.Alias, "_", "-", -1)

	return link
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

func NewNetFromString(str string) (*Net, error) {
	n := &Net{}
	split := strings.SplitN(str, ":", 2)
	n.Type = split[0]
	if n.Type == "container" {
		if len(split) < 2 {
			return nil, fmt.Errorf("Missing container id or name for net param: %s", str)
		}
		n.Container = *NewContainerNameFromString(split[1])
	} else if n.Type != "none" && n.Type != "host" && n.Type != "bridge" {
		return nil, fmt.Errorf("Unknown net type: %s", str)
	}
	return n, nil
}

// Methods

func (containerName ContainerName) String() string {
	name := containerName.Name
	if containerName.Namespace != "" {
		name = fmt.Sprintf("%s.%s", containerName.Namespace, name)
	}
	return name
}

// Same as String() but makes alias if not specified
func (link Link) String() string {
	if link.Alias == "" && link.Name == "" {
		return ""
	}
	name := link.Name
	if link.Namespace != "" {
		name = fmt.Sprintf("%s.%s", link.Namespace, name)
	}
	alias := link.Alias
	if alias == "" {
		alias = link.Name
	}
	return fmt.Sprintf("%s:%s", name, alias)
}

func (link Link) ContainerName() ContainerName {
	return ContainerName{
		Namespace: link.Namespace,
		Name:      link.Name,
	}
}

func (a *ContainerName) DefaultNamespace(ns string) bool {
	if a.Namespace == "" {
		a.Namespace = ns
		return true
	}
	return false
}

func (a *Link) DefaultNamespace(ns string) bool {
	if a.Namespace == "" {
		a.Namespace = ns
		return true
	}
	return false
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

func (net *Net) String() string {
	if net == nil {
		return ""
	}
	if net.Type == "container" {
		return net.Type + ":" + net.Container.String()
	}
	return net.Type
}
