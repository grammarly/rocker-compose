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
	"strconv"
	"strings"
)

func (containerName *ContainerName) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var name string
	if err := unmarshal(&name); err != nil {
		return err
	}
	*containerName = *NewContainerNameFromString(name)
	return nil
}

func (containerName ContainerName) MarshalYAML() (interface{}, error) {
	return containerName.String(), nil
}

func (link *Link) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var name string
	if err := unmarshal(&name); err != nil {
		return err
	}
	*link = *NewLinkFromString(name)
	return nil
}

func (link Link) MarshalYAML() (interface{}, error) {
	return link.String(), nil
}

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
	if !strings.Contains(b.Port, "/") {
		b.Port = b.Port + "/tcp"
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

func (cmd *Cmd) UnmarshalYAML(unmarshal func(interface{}) error) (err error) {
	parts, err := stringSliceMaybeString([]string{"/bin/sh", "-c"}, unmarshal)
	if err != nil {
		return err
	}
	*cmd = (Cmd)(parts)

	return nil
}

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

func (net *Net) MarshalYAML() (interface{}, error) {
	return net.String(), nil
}

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

func (v *Strings) UnmarshalYAML(unmarshal func(interface{}) error) error {
	parts, err := stringSliceMaybeString([]string{}, unmarshal)
	if err != nil {
		return err
	}
	*v = (Strings)(parts)

	return nil
}

func (v *StringMap) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var (
		parts map[string]string
		value string
	)

	if err := unmarshal(&parts); err != nil {
		if err := unmarshal(&value); err != nil {
			return err
		}
		parts = map[string]string{}

		// TODO: more intelligent parsing, consider quotes
		for _, pair := range strings.Split(value, " ") {
			kv := strings.SplitN(pair, "=", 2)
			value := "true"
			if len(kv) > 1 {
				value = kv[1]
			}
			parts[kv[0]] = value
		}
	}

	*v = (StringMap)(parts)

	return nil
}

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
