package compose

import (
	"fmt"
	"strings"

	"github.com/fsouza/go-dockerclient"
)

func NewContainerFromDocker(dockerContainer *docker.Container) *Container {
	return &Container{
		Id:      dockerContainer.ID,
		Image:   NewImageNameFromString(dockerContainer.Config.Image),
		Name:    NewContainerNameFromString(dockerContainer.Name),
		Created: dockerContainer.Created,
		State: &ContainerState{
			Running:    dockerContainer.State.Running,
			Paused:     dockerContainer.State.Paused,
			Restarting: dockerContainer.State.Restarting,
			OOMKilled:  dockerContainer.State.OOMKilled,
			Pid:        dockerContainer.State.Pid,
			ExitCode:   dockerContainer.State.ExitCode,
			Error:      dockerContainer.State.Error,
			StartedAt:  dockerContainer.State.StartedAt,
			FinishedAt: dockerContainer.State.FinishedAt,
		},
		Config:    NewContainerConfigFromDocker(dockerContainer),
		container: dockerContainer,
	}
}

func NewContainerConfigFromDocker(apiContainer *docker.Container) *ConfigContainer {
	container := &ConfigContainer{
		Cmd:        apiContainer.Config.Cmd,
		Entrypoint: apiContainer.Config.Entrypoint,
		State:      NewConfigStateFromBool(apiContainer.State.Running),
		Dns:        apiContainer.HostConfig.DNS,
		AddHost:    apiContainer.HostConfig.ExtraHosts,
		Memory:     NewConfigMemoryFromInt64(apiContainer.HostConfig.Memory),
		MemorySwap: NewConfigMemoryFromInt64(apiContainer.HostConfig.MemorySwap),
	}

	if apiContainer.Config.Hostname != "" {
		container.Hostname = &apiContainer.Config.Hostname
	}
	if apiContainer.Config.Domainname != "" {
		container.Domainname = &apiContainer.Config.Domainname
	}
	if apiContainer.Config.User != "" {
		container.User = &apiContainer.Config.User
	}
	if apiContainer.Config.WorkingDir != "" {
		container.Workdir = &apiContainer.Config.WorkingDir
	}
	if apiContainer.Config.Image != "" {
		container.Image = &apiContainer.Config.Image
	}
	if apiContainer.Config.NetworkDisabled != false {
		container.NetworkDisabled = &apiContainer.Config.NetworkDisabled
	}
	if apiContainer.HostConfig.Privileged != false {
		container.Privileged = &apiContainer.HostConfig.Privileged
	}
	if apiContainer.HostConfig.PublishAllPorts != false {
		container.PublishAllPorts = &apiContainer.HostConfig.PublishAllPorts
	}
	if apiContainer.HostConfig.NetworkMode != "" {
		container.Net = &apiContainer.HostConfig.NetworkMode
	}
	if apiContainer.HostConfig.PidMode != "" {
		container.Pid = &apiContainer.HostConfig.PidMode
	}
	if apiContainer.HostConfig.CPUShares != 0 {
		container.CpuShares = &apiContainer.HostConfig.CPUShares
	}
	if apiContainer.HostConfig.CPUSet != "" {
		container.CpusetCpus = &apiContainer.HostConfig.CPUSet
	}
	if apiContainer.HostConfig.RestartPolicy != (docker.RestartPolicy{}) {
		container.Restart = &RestartPolicy{
			Name:              apiContainer.HostConfig.RestartPolicy.Name,
			MaximumRetryCount: apiContainer.HostConfig.RestartPolicy.MaximumRetryCount,
		}
	}

	if len(apiContainer.Config.ExposedPorts) > 0 {
		container.Expose = []string{}
		for port, _ := range apiContainer.Config.ExposedPorts {
			container.Expose = append(container.Expose, string(port))
		}
	}

	if len(apiContainer.Config.Env) > 0 {
		container.Env = map[string]string{}
		for _, env := range apiContainer.Config.Env {
			split := strings.SplitN(env, "=", 2)
			container.Env[split[0]] = split[1]
		}
	}

	if len(apiContainer.Config.Volumes) > 0 {
		container.Volumes = []string{}
		for volume, _ := range apiContainer.Config.Volumes {
			container.Volumes = append(container.Volumes, volume)
		}
	}

	if len(apiContainer.HostConfig.Binds) > 0 {
		if container.Volumes == nil {
			container.Volumes = []string{}
		}
		for _, bind := range apiContainer.HostConfig.Binds {
			container.Volumes = append(container.Volumes, bind)
		}
	}

	if len(apiContainer.HostConfig.PortBindings) > 0 {
		container.Ports = []PortBinding{}
		for port, bindings := range apiContainer.HostConfig.PortBindings {
			for _, binding := range bindings {
				container.Ports = append(container.Ports, PortBinding{
					Port:     string(port),
					HostIp:   binding.HostIP,
					HostPort: binding.HostPort,
				})
			}
		}
	}

	if len(apiContainer.HostConfig.Links) > 0 {
		container.Links = []ContainerName{}
		for _, name := range apiContainer.HostConfig.Links {
			container.Links = append(container.Links, *NewContainerNameFromString(name))
		}
	}

	if len(apiContainer.HostConfig.VolumesFrom) > 0 {
		container.VolumesFrom = []ContainerName{}
		for _, name := range apiContainer.HostConfig.VolumesFrom {
			container.VolumesFrom = append(container.VolumesFrom, *NewContainerNameFromString(name))
		}
	}

	if len(apiContainer.HostConfig.Ulimits) > 0 {
		container.Ulimits = []ConfigUlimit{}
		for _, ulimit := range apiContainer.HostConfig.Ulimits {
			container.Ulimits = append(container.Ulimits, ConfigUlimit{
				Name: ulimit.Name,
				Soft: ulimit.Soft,
				Hard: ulimit.Hard,
			})
		}
	}

	if apiContainer.Config.Labels != nil {
		filteredLabels := map[string]string{}
		for k, v := range apiContainer.Config.Labels {
			if !strings.HasPrefix(k, "rocker-compose-") {
				filteredLabels[k] = v
			}
		}
		if len(filteredLabels) > 0 {
			container.Labels = filteredLabels
		}
	}

	return container
}

func (config *ConfigContainer) GetApiConfig() *docker.Config {
	// Copy simple values
	apiConfig := &docker.Config{
		Cmd:        config.Cmd,
		Entrypoint: config.Entrypoint,
		Labels:     config.Labels,
	}
	if config.Image != nil {
		apiConfig.Image = *config.Image
	}
	if config.Hostname != nil {
		apiConfig.Hostname = *config.Hostname
	}
	if config.Domainname != nil {
		apiConfig.Domainname = *config.Domainname
	}
	if config.Workdir != nil {
		apiConfig.WorkingDir = *config.Workdir
	}
	if config.User != nil {
		apiConfig.User = *config.User
	}
	if config.Memory != nil {
		apiConfig.Memory = config.Memory.Int64()
	}
	if config.MemorySwap != nil {
		apiConfig.MemorySwap = config.MemorySwap.Int64()
	}
	if config.CpusetCpus != nil {
		apiConfig.CPUSet = *config.CpusetCpus
	}
	if config.CpuShares != nil {
		apiConfig.CPUShares = *config.CpuShares
	}
	if config.NetworkDisabled != nil {
		apiConfig.NetworkDisabled = *config.NetworkDisabled
	}

	// expose
	if config.Expose != nil {
		apiConfig.ExposedPorts = map[docker.Port]struct{}{}
		for _, portBinding := range config.Expose {
			port := (docker.Port)(portBinding)
			apiConfig.ExposedPorts[port] = struct{}{}
		}
	}

	// env
	if config.Env != nil {
		apiConfig.Env = []string{}
		for key, val := range config.Env {
			apiConfig.Env = append(apiConfig.Env, fmt.Sprintf("%s=%s", key, val))
		}
	}

	// volumes
	if config.Volumes != nil {
		hostVolumes := map[string]struct{}{}
		for _, volume := range config.Volumes {
			if !strings.Contains(volume, ":") {
				hostVolumes[volume] = struct{}{}
			}
		}
		if len(hostVolumes) > 0 {
			apiConfig.Volumes = hostVolumes
		}
	}

	// TODO: SecurityOpts, OnBuild ?

	return apiConfig
}

func (config *ConfigContainer) GetApiHostConfig() *docker.HostConfig {
	// TODO: CapAdd, CapDrop, LxcConf, Devices, LogConfig, ReadonlyRootfs,
	//       SecurityOpt, CgroupParent, CPUQuota, CPUPeriod
	// TODO: where Memory and MemorySwap should go?
	hostConfig := &docker.HostConfig{
		DNS:           config.Dns,
		ExtraHosts:    config.AddHost,
		RestartPolicy: config.Restart.ToDockerApi(),
		Memory:        config.Memory.Int64(),
		MemorySwap:    config.MemorySwap.Int64(),
	}

	if config.Net != nil {
		hostConfig.NetworkMode = *config.Net
	}
	if config.Pid != nil {
		hostConfig.PidMode = *config.Pid
	}
	if config.CpusetCpus != nil {
		hostConfig.CPUSet = *config.CpusetCpus
	}

	// Binds
	binds := []string{}
	for _, volume := range config.Volumes {
		if strings.Contains(volume, ":") {
			binds = append(binds, volume)
		}
	}
	if len(binds) > 0 {
		hostConfig.Binds = binds
	}

	// Privileged
	if config.Privileged != nil {
		hostConfig.Privileged = *config.Privileged
	}

	// PublishAllPorts
	if config.PublishAllPorts != nil {
		hostConfig.PublishAllPorts = *config.PublishAllPorts
	}

	// PortBindings
	if len(config.Ports) > 0 {
		hostConfig.PortBindings = map[docker.Port][]docker.PortBinding{}
		for _, configPort := range config.Ports {
			key := (docker.Port)(configPort.Port)
			binding := docker.PortBinding{configPort.HostIp, configPort.HostPort}
			if _, ok := hostConfig.PortBindings[key]; !ok {
				hostConfig.PortBindings[key] = []docker.PortBinding{}
			}
			hostConfig.PortBindings[key] = append(hostConfig.PortBindings[key], binding)
		}
	}

	// Links
	if len(config.Links) > 0 {
		hostConfig.Links = []string{}
		for _, link := range config.Links {
			hostConfig.Links = append(hostConfig.Links, link.String())
		}
	}

	// VolumesFrom
	if len(config.VolumesFrom) > 0 {
		hostConfig.VolumesFrom = []string{}
		for _, volume := range config.VolumesFrom {
			hostConfig.VolumesFrom = append(hostConfig.VolumesFrom, volume.String())
		}
	}

	// Ulimits
	if len(config.Ulimits) > 0 {
		hostConfig.Ulimits = []docker.ULimit{}
		for _, ulimit := range config.Ulimits {
			hostConfig.Ulimits = append(hostConfig.Ulimits, docker.ULimit{
				Name: ulimit.Name,
				Soft: ulimit.Soft,
				Hard: ulimit.Hard,
			})
		}
	}

	return hostConfig
}
