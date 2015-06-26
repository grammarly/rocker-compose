package compose

import (
	"fmt"
	"strings"

	"github.com/fsouza/go-dockerclient"
)

func NewContainerConfigFromDocker(apiContainer *docker.Container) *ConfigContainer {
	// container := &ConfigContainer{
	//  Image: apiContainer.Image,
	//  Net: apiContainer.Net,
	// }
	return &ConfigContainer{}
}

func (config *ConfigContainer) GetApiConfig() *docker.Config {
	// Copy simple values
	apiConfig := &docker.Config{
		Hostname:   config.Hostname,
		Domainname: config.Domainname,
		User:       config.User,
		Memory:     config.Memory.Int64(),
		MemorySwap: config.MemorySwap.Int64(),
		CPUSet:     config.CpusetCpus,
		// PortSpecs:  config.Ports, // TODO: WTF?
		Cmd:        config.Cmd,
		Image:      config.Image,
		WorkingDir: config.Workdir,
		Entrypoint: config.Entrypoint,
		Labels:     config.Labels,
	}
	// Copy more complex values
	if config.CpuShares != nil {
		apiConfig.CPUShares = *config.CpuShares
	}
	if config.NetworkDisabled != nil {
		apiConfig.NetworkDisabled = *config.NetworkDisabled
	}

	// expose
	if len(config.Expose) > 0 {
		apiConfig.ExposedPorts = map[docker.Port]struct{}{}
		for _, portBinding := range config.Expose {
			port := (docker.Port)(portBinding)
			apiConfig.ExposedPorts[port] = struct{}{}
		}
	}

	// env
	if len(config.Env) > 0 {
		apiConfig.Env = []string{}
		for key, val := range config.Env {
			apiConfig.Env = append(apiConfig.Env, fmt.Sprintf("%s=%s", key, val))
		}
	}

	// volumes
	hostVolumes := map[string]struct{}{}
	for _, volume := range config.Volumes {
		if !strings.Contains(volume, ":") {
			hostVolumes[volume] = struct{}{}
		}
	}
	if len(hostVolumes) > 0 {
		apiConfig.Volumes = hostVolumes
	}

	// TODO: SecurityOpts, OnBuild ?

	return apiConfig
}

func (config *ConfigContainer) GetApiHostConfig() *docker.HostConfig {
	// TODO: CapAdd, CapDrop, LxcConf, Devices, LogConfig, ReadonlyRootfs,
	//       SecurityOpt, CgroupParent, CPUQuota, CPUPeriod
	hostConfig := &docker.HostConfig{
		DNS:           config.Dns,
		ExtraHosts:    config.AddHost,
		NetworkMode:   config.Net,
		PidMode:       config.Pid,
		RestartPolicy: config.Restart.ToDockerApi(),
		Memory:        config.Memory.Int64(),
		MemorySwap:    config.MemorySwap.Int64(),
		CPUSet:        config.CpusetCpus,
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

	// PortBindings
	if len(config.Ports) > 0 {
		hostConfig.PortBindings = map[docker.Port][]docker.PortBinding{}
		for _, configPort := range config.Ports {
			port, hostIp, hostPort := configPort.Parse()
			key := (docker.Port)(port)
			binding := docker.PortBinding{hostIp, hostPort}
			value := []docker.PortBinding{binding}
			hostConfig.PortBindings[key] = value
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
