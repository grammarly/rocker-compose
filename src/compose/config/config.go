/*-
 * Copyright 2015 Grammarly, Inc.
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

// Package config holds functionality for processing compose.yml manifests,
// templating, converting manifests to docker api run spec and comparing them against
// each other.
//
// Comparing mechanism plays  the key role for rocker-compose idempotency features. We implement both
// parsing and serializing for each property and whole manifests, which allows us to store
// configuration in a label of a container and makes easier detecting changes.
package config

import (
	"fmt"
	"io"
	"math"
	"os"
	"path"
	"path/filepath"
	"regexp"
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
	Image           *string        `yaml:"image,omitempty"`             //
	Net             *Net           `yaml:"net,omitempty"`               //
	Pid             *string        `yaml:"pid,omitempty"`               //
	Uts             *string        `yaml:"uts,omitempty"`               //
	State           *ConfigState   `yaml:"state,omitempty"`             // "running" or "created" or "ran"
	Dns             Strings        `yaml:"dns,omitempty"`               //
	AddHost         Strings        `yaml:"add_host,omitempty"`          //
	Restart         *RestartPolicy `yaml:"restart,omitempty"`           //
	Memory          *ConfigMemory  `yaml:"memory,omitempty"`            //
	MemorySwap      *ConfigMemory  `yaml:"memory_swap,omitempty"`       //
	CpuShares       *int64         `yaml:"cpu_shares,omitempty"`        //
	CpusetCpus      *string        `yaml:"cpuset_cpus,omitempty"`       //
	OomKillDisable  *bool          `yaml:"oom_kill_disable,omitempty"`  // e.g. docker run --oom-kill-disable TODO: pull request to go-dockerclient
	Ulimits         []ConfigUlimit `yaml:"ulimits,omitempty"`           // search by "Ulimits" here https://goo.gl/IxbZck
	Privileged      *bool          `yaml:"privileged,omitempty"`        //
	Cmd             Cmd            `yaml:"cmd,omitempty"`               //
	Entrypoint      Strings        `yaml:"entrypoint,omitempty"`        //
	Expose          Strings        `yaml:"expose,omitempty"`            //
	Ports           Ports          `yaml:"ports,omitempty"`             //
	PublishAllPorts *bool          `yaml:"publish_all_ports,omitempty"` //
	Labels          StringMap      `yaml:"labels,omitempty"`            //
	Env             StringMap      `yaml:"env,omitempty"`               //
	VolumesFrom     ContainerNames `yaml:"volumes_from,omitempty"`      //
	Volumes         Strings        `yaml:"volumes,omitempty"`           //
	Links           Links          `yaml:"links,omitempty"`             //
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
	ExtraHosts  Strings   `yaml:"extra_hosts,omitempty"`
	WorkingDir  *string   `yaml:"working_dir,omitempty"`
	Environment StringMap `yaml:"environment,omitempty"`

	// Extra properties that is not known by rocker-compose
	Extra map[string]interface{} `yaml:"extra,omitempty"`

	lastCompareField string
}

// ContainerName represents the pair of namespace and container name.
// It is used in all places that refers to container by name, such as:
// containers in manifests, volumes_from, etc.
type ContainerName struct {
	Namespace string
	Name      string
}

// Link is same as ContainerName with addition of Alias property, which
// specifies associated container alias
type Link struct {
	Namespace string
	Name      string
	Alias     string
}

// ConfigUlimit describes ulimit specification for the manifest file
type ConfigUlimit struct {
	Name string
	Soft int64
	Hard int64
}

// ConfigMemory is memory in bytes that is used for Memory and MemorySwap
// properties of the container spec. It is parsed from string (e.g. "64M")
// to int64 bytes as a uniform representation.
type ConfigMemory int64

// RestartPolicy represents "restart" property of the container spec. Possible
// values are: no | always | on-failure,N (where N is number of times it is allowed to fail)
// Default value is "always". Despite Docker's default value is "no", we found that more often
// we want to have "always" and people constantly forget to put it.
type RestartPolicy struct {
	Name              string
	MaximumRetryCount int
}

// PortBinding represents a single port binding spec, which is used in "ports" property.
// format: ip:hostPort:containerPort | ip::containerPort | hostPort:containerPort | containerPort
type PortBinding struct {
	Port     string
	HostIp   string
	HostPort string
}

// ConfigState represents "state" property from the manifest.
// Possible values are: running | created | ran
type ConfigState string

// Net is "net" property, which can also refer to some container
type Net struct {
	Type      string // bridge|none|container|host
	Container ContainerName
}

// StringMap implements yaml [un]serializable map[string]string
// is used for "labels" and "env" properties. See yaml.go for more info.
type StringMap map[string]string

// ContainerNames is a collection of container references
type ContainerNames []ContainerName

// Ports is a collection of port bindings
type Ports []PortBinding

// Links is a collection of container links
type Links []Link

// Cmd implements yaml [un]serializable "cmd" property of the container spec.
// See yaml.go for more info.
type Cmd []string

// Strings implements yaml [un]serializable list of strings.
// See yaml.go for more info.
type Strings []string

// NewFromFile reads and parses config from a file.
// If given filename is not absolute path, it resolves absolute name from the current
// working directory. See ReadConfig/4 for reading and parsing details.
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

// ReadConfig reads and parses the config from io.Reader stream.
// Before parsing it processes config through a template engine implemented in template.go.
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

	// empty namespace is a backward compatible docker-compose format
	// we will try to guess the namespace my parent directory name
	if config.Namespace == "" {
		parentDir := filepath.Base(basedir)
		config.Namespace = regexp.MustCompile("[^a-z0-9\\-\\_]").ReplaceAllString(parentDir, "")
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
	for _, v := range getYamlFields() {
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
		if container.ExtraHosts != nil {
			if container.AddHost == nil {
				container.AddHost = container.ExtraHosts
			}
			container.ExtraHosts = nil
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

		img := imagename.NewFromString(*container.Image)

		if !img.IsStrict() && !img.HasVersionRange() && !img.All() {
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

// NewContainerName produce ContainerName object
func NewContainerName(namespace, name string) *ContainerName {
	return &ContainerName{namespace, name}
}

// NewContainerNameFromString parses a string to a ContainerName object
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

// NewLinkFromString parses a string to a Link object
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

// NewConfigMemoryFromString parses a string to a ConfigMemory object
// Examples of string that can be given:
//    "124124" (124124 bytes)
//    "124124b" (same)
//    "1024k"
//    "512m"
//    "2g"
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

// NewConfigMemoryFromInt64 makes a ConfigMemory from int64 value
func NewConfigMemoryFromInt64(value int64) *ConfigMemory {
	if value == 0 {
		return nil
	}
	memory := (ConfigMemory)(value)
	return &memory
}

// NewNetFromString parses a string to a Net object.
// Possible values: bridge|none|container:CONTAINER_NAME|host
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

// String gives a string representation of the container name
func (containerName ContainerName) String() string {
	name := containerName.Name
	if containerName.Namespace != "" {
		name = fmt.Sprintf("%s.%s", containerName.Namespace, name)
	}
	return name
}

// String is same as ContainerName.String() but adds alias
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

// ContainerName makes ContainerName object from a Link
func (link Link) ContainerName() ContainerName {
	return ContainerName{
		Namespace: link.Namespace,
		Name:      link.Name,
	}
}

// DefaultNamespace assigns a namespace for ContainerName it does not have one.
// Returns true if namespace was changed.
func (a *ContainerName) DefaultNamespace(ns string) bool {
	if a.Namespace == "" {
		a.Namespace = ns
		return true
	}
	return false
}

// DefaultNamespace assigns a namespace for Link it does not have one.
// Returns true if namespace was changed.
func (a *Link) DefaultNamespace(ns string) bool {
	if a.Namespace == "" {
		a.Namespace = ns
		return true
	}
	return false
}

// Int64 returns int64 value of the ConfigMemory object
func (m *ConfigMemory) Int64() int64 {
	if m == nil {
		return 0
	}
	return (int64)(*m)
}

// ToDockerApi converts RestartPolicy to a docker.RestartPolicy object
// which is eatable by go-dockerclient.
func (r *RestartPolicy) ToDockerApi() docker.RestartPolicy {
	if r == nil {
		return docker.RestartPolicy{}
	}
	return docker.RestartPolicy{
		Name:              r.Name,
		MaximumRetryCount: r.MaximumRetryCount,
	}
}

// Bool returns true if state is "running" or not specified
func (state *ConfigState) Bool() bool {
	if state != nil {
		return *state == "running"
	}
	return true // "running" or anything else
}

// IsRan returns true if state is "ran"
func (state *ConfigState) IsRan() bool {
	return state != nil && *state == "ran"
}

// String returns string representation of Net object.
func (net *Net) String() string {
	if net == nil {
		return ""
	}
	if net.Type == "container" {
		return net.Type + ":" + net.Container.String()
	}
	return net.Type
}
