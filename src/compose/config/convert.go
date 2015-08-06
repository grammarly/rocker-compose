package config

import (
	"fmt"
	"strings"

	"github.com/fsouza/go-dockerclient"
	"github.com/go-yaml/yaml"
)

type ErrNotRockerCompose struct {
	ContainerId string
}

func (err ErrNotRockerCompose) Error() string {
	return fmt.Sprintf("Expecting container %.12s to have label 'rocker-compose-config' to parse it", err.ContainerId)
}

func NewFromDocker(apiContainer *docker.Container) (*Container, error) {
	yamlData, ok := apiContainer.Config.Labels["rocker-compose-config"]
	if !ok {
		return nil, ErrNotRockerCompose{apiContainer.ID}
	}

	container := &Container{}

	if err := yaml.Unmarshal([]byte(yamlData), container); err != nil {
		return nil, fmt.Errorf("Failed to parse YAML config for container %s, error: %s", apiContainer.Name, err)
	}

	if container.Labels != nil {
		for k, _ := range container.Labels {
			if strings.HasPrefix(k, "rocker-compose-") {
				delete(container.Labels, k)
			}
		}
	}

	return container, nil
}

func (config *Container) GetApiConfig() *docker.Config {
	// Copy simple values
	apiConfig := &docker.Config{
		Entrypoint: config.Entrypoint,
		Labels:     config.Labels,
	}
	if config.Cmd != nil {
		apiConfig.Cmd = config.Cmd
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
	if len(config.Expose) > 0 || len(config.Ports) > 0 {
		apiConfig.ExposedPorts = map[docker.Port]struct{}{}
		for _, portBinding := range config.Expose {
			port := (docker.Port)(portBinding)
			apiConfig.ExposedPorts[port] = struct{}{}
		}
		// expose publised ports as well
		for _, configPort := range config.Ports {
			port := (docker.Port)(configPort.Port)
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

func (config *Container) GetApiHostConfig() *docker.HostConfig {
	// TODO: CapAdd, CapDrop, LxcConf, Devices, LogConfig, ReadonlyRootfs,
	//       SecurityOpt, CgroupParent, CPUQuota, CPUPeriod
	// TODO: where Memory and MemorySwap should go?
	hostConfig := &docker.HostConfig{
		DNS:           config.Dns,
		ExtraHosts:    config.AddHost,
		RestartPolicy: config.Restart.ToDockerApi(),
		Memory:        config.Memory.Int64(),
		MemorySwap:    config.MemorySwap.Int64(),
		NetworkMode:   config.Net.String(),
	}

	// if state is "running", then restart policy sould be "always" by default
	if config.State.Bool() && config.Restart == nil {
		hostConfig.RestartPolicy = (&RestartPolicy{"always", 0}).ToDockerApi()
	}

	if config.Pid != nil {
		hostConfig.PidMode = *config.Pid
	}
	if config.Uts != nil {
		hostConfig.UTSMode = *config.Uts
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
