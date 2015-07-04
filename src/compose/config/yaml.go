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

func (cmd *ConfigCmd) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var value string
	if err := unmarshal(&cmd.Parts); err != nil {
		if err := unmarshal(&value); err != nil {
			return err
		}
		cmd.Parts = []string{"/bin/sh", "-c", value}
	}
	return nil
}

func (cmd *ConfigCmd) MarshalYAML() (interface{}, error) {
	if cmd == nil {
		return nil, nil
	}
	return cmd.Parts, nil
}
