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

// ExtendFrom extends the container spec from a given one
func (container *Container) ExtendFrom(parent *Container) {
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
	if container.WaitFor == nil {
		container.WaitFor = parent.WaitFor
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
	if container.Extra == nil {
		container.Extra = parent.Extra
	}

	return
}
