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

package config

import (
	"fmt"
	"strconv"
	"strings"
)

// UnmarshalYAML unserialize Config object form YAML
// It supports compatibility with docker-compose YAML spec where containers map is specified
// on the first level. rocker-compose provides extra level for global properties such as 'namespace'
// This function fallbacks to the docker-compose format if 'namespace' key was not found on the
// first level.
func (config *Config) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// compatibiliy with docker-compose format, if namespace is not specified,
	// we think it is docker-compose format
	c := &struct {
		Namespace  *string
		Containers *map[string]*Container
	}{
		&config.Namespace,
		&config.Containers,
	}
	if err := unmarshal(c); err != nil {
		return err
	}
	// parse containers only, if namespace is empty, we will deal with it later
	if *c.Namespace == "" {
		if err := unmarshal(&c.Containers); err != nil {
			return err
		}
	}
	return nil
}

// UnmarshalYAML unserialize ContainerName object from YAML
func (n *ContainerName) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var name string
	if err := unmarshal(&name); err != nil {
		return err
	}
	*n = *NewContainerNameFromString(name)
	return nil
}

// MarshalYAML serialize ContainerName object to YAML
func (n ContainerName) MarshalYAML() (interface{}, error) {
	return n.String(), nil
}

// UnmarshalYAML unserialize Link object from YAML
func (link *Link) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var name string
	if err := unmarshal(&name); err != nil {
		return err
	}
	*link = *NewLinkFromString(name)
	return nil
}

// MarshalYAML serialize ContainerName object to YAML
func (link Link) MarshalYAML() (interface{}, error) {
	return link.String(), nil
}

// UnmarshalYAML unserialize ConfigMemory object from YAML
func (m *Memory) UnmarshalYAML(unmarshal func(interface{}) error) error {
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

// UnmarshalYAML unserialize RestartPolicy object from YAML
func (r *RestartPolicy) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var name string
	if err := unmarshal(&name); err != nil {
		return err
	}
	if name == "" || name == "no" {
		r.Name = "no"
	} else if name == "always" {
		r.Name = "always"
	} else if strings.Index(name, "on-failure") == 0 {
		r.Name = "on-failure"
		parts := strings.SplitN(name, ",", 2)
		if len(parts) == 2 {
			n, err := strconv.ParseInt(parts[1], 10, 16)
			if err != nil {
				return err
			}
			r.MaximumRetryCount = (int)(n)
		}
	}
	return nil
}

// MarshalYAML serialize RestartPolicy object to YAML
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

// UnmarshalYAML unserialize PortBinding object from YAML
func (b *PortBinding) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var value string
	if err := unmarshal(&value); err != nil {
		return err
	}
	split := strings.SplitN(value, ":", 3)
	if len(split) == 3 {
		b.Port = split[2]
		b.HostIP = split[0]
		b.HostPort = split[1]
	} else if len(split) == 2 {
		b.Port = split[1]
		b.HostPort = split[0]
	} else {
		b.Port = split[0]
	}
	if !strings.Contains(b.Port, "/") {
		b.Port = b.Port + "/tcp"
	}
	return nil
}

// MarshalYAML serialize PortBinding object to YAML
func (b PortBinding) MarshalYAML() (interface{}, error) {
	if b.HostIP != "" && b.HostPort != "" {
		return fmt.Sprintf("%s:%s:%s", b.HostIP, b.HostPort, b.Port), nil
	} else if b.HostIP != "" {
		return fmt.Sprintf("%s::%s", b.HostIP, b.Port), nil
	} else if b.HostPort != "" {
		return fmt.Sprintf("%s:%s", b.HostPort, b.Port), nil
	}
	return b.Port, nil
}

// UnmarshalYAML unserialize Cmd object from YAML
// If string is given, then it adds '/bin/sh -c' prefix to a command
func (cmd *Cmd) UnmarshalYAML(unmarshal func(interface{}) error) (err error) {
	parts, err := stringSliceMaybeString([]string{"/bin/sh", "-c"}, unmarshal)
	if err != nil {
		return err
	}
	*cmd = (Cmd)(parts)

	return nil
}

// UnmarshalYAML unserialize Net object from YAML
func (n *Net) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var str string
	if err := unmarshal(&str); err != nil {
		return err
	}
	value, err := NewNetFromString(str)
	if err != nil {
		return err
	}
	*n = *value
	return nil
}

// MarshalYAML serialize Net object to YAML
func (n *Net) MarshalYAML() (interface{}, error) {
	return n.String(), nil
}

// UnmarshalYAML unserialize slice of ContainerName objects from YAML
// Either single value or array can be given. Single 'value' casts to array{'value'}
func (v *ContainerNames) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var (
		parts []ContainerName
		value ContainerName
	)
	if err := unmarshal(&parts); err != nil {
		if err := unmarshal(&value); err != nil {
			return err
		}
		parts = []ContainerName{value}
	}
	*v = (ContainerNames)(parts)

	return nil
}

// UnmarshalYAML unserialize slice of Port objects from YAML
// Either single value or array can be given. Single 'value' casts to array{'value'}
func (v *Ports) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var (
		parts []PortBinding
		value PortBinding
	)
	if err := unmarshal(&parts); err != nil {
		if err := unmarshal(&value); err != nil {
			return err
		}
		parts = []PortBinding{value}
	}
	*v = (Ports)(parts)

	return nil
}

// UnmarshalYAML unserialize slice of Link objects from YAML
// Either single value or array can be given. Single 'value' casts to array{'value'}
func (v *Links) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var (
		parts []Link
		value Link
	)
	if err := unmarshal(&parts); err != nil {
		if err := unmarshal(&value); err != nil {
			return err
		}
		parts = []Link{value}
	}
	*v = (Links)(parts)

	return nil
}

// UnmarshalYAML unserialize slice of Strings from YAML
// Either single value or array can be given. Single 'value' casts to array{'value'}
func (v *Strings) UnmarshalYAML(unmarshal func(interface{}) error) error {
	parts, err := stringSliceMaybeString([]string{}, unmarshal)
	if err != nil {
		return err
	}
	*v = (Strings)(parts)

	return nil
}

// UnmarshalYAML unserialize map[string]string objects from YAML
// Map can be also specified as string "key=val key2=val2"
// and also as array of strings []string{"key=val", "key2=val2"}
func (v *StringMap) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var (
		value map[string]string
		slice []string
		str   string
	)

	// try parse as map[string]string
	if err := unmarshal(&value); err != nil {
		// try parse as []string
		if err := unmarshal(&slice); err != nil {
			// try parse as string
			if err := unmarshal(&str); err != nil {
				return err
			}
			// TODO: more intelligent split?
			slice = strings.Split(str, " ")
		}

		value = map[string]string{}

		// TODO: more intelligent parsing, consider quotes
		for _, pair := range slice {
			kv := strings.SplitN(pair, "=", 2)
			val := "true"
			if len(kv) > 1 {
				val = kv[1]
			}
			value[kv[0]] = val
		}
	}

	*v = (StringMap)(value)

	return nil
}

// stringSliceMaybeString provides a generic YAML parsing functionality for the list of strings
// it can either receive a string or a list of strings. If a single string given, it casts it
// to a list of strings and adds 'prefix'
func stringSliceMaybeString(prefix []string, unmarshal func(interface{}) error) ([]string, error) {
	var (
		parts []string
		value string
	)
	if err := unmarshal(&parts); err != nil {
		if err := unmarshal(&value); err != nil {
			return parts, err
		}
		parts = append(prefix, value)
	}
	return parts, nil
}
