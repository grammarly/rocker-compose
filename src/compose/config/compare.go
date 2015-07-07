package config

func (a *Container) LastCompareField() string {
	return a.lastCompareField
}

func (a *Container) IsEqualTo(b *Container) bool {
	// Compare simple values

	a.lastCompareField = "Image"
	if !comparePointerString(a.Image, b.Image) {
		return false
	}

	a.lastCompareField = "Net"
	if !comparePointerNet(a.Net, b.Net) {
		return false
	}

	a.lastCompareField = "Pid"
	if !comparePointerString(a.Pid, b.Pid) {
		return false
	}

	a.lastCompareField = "Uts"
	if !comparePointerString(a.Uts, b.Uts) {
		return false
	}

	a.lastCompareField = "CpusetCpus"
	if !comparePointerString(a.CpusetCpus, b.CpusetCpus) {
		return false
	}

	a.lastCompareField = "Hostname"
	if !comparePointerString(a.Hostname, b.Hostname) {
		return false
	}

	a.lastCompareField = "Domainname"
	if !comparePointerString(a.Domainname, b.Domainname) {
		return false
	}

	a.lastCompareField = "User"
	if !comparePointerString(a.User, b.User) {
		return false
	}

	a.lastCompareField = "Workdir"
	if !comparePointerString(a.Workdir, b.Workdir) {
		return false
	}

	// Compare RestartPolicy
	a.lastCompareField = "Restart"
	if !comparePointerRestart(a.Restart, b.Restart) {
		return false
	}

	// Comparable objects

	a.lastCompareField = "State"
	if !a.State.IsEqualTo(b.State) {
		return false
	}

	a.lastCompareField = "Memory"
	if !comparePointerMemory(a.Memory, b.Memory) {
		return false
	}

	a.lastCompareField = "MemorySwap"
	if !comparePointerMemory(a.MemorySwap, b.MemorySwap) {
		return false
	}

	// Compare pointer values

	a.lastCompareField = "CpuShares"
	if !comparePointerInt64(a.CpuShares, b.CpuShares) {
		return false
	}

	a.lastCompareField = "OomKillDisable"
	if !comparePointerBool(a.OomKillDisable, b.OomKillDisable) {
		return false
	}

	a.lastCompareField = "Privileged"
	if !comparePointerBool(a.Privileged, b.Privileged) {
		return false
	}

	a.lastCompareField = "PublishAllPorts"
	if !comparePointerBool(a.PublishAllPorts, b.PublishAllPorts) {
		return false
	}

	a.lastCompareField = "NetworkDisabled"
	if !comparePointerBool(a.NetworkDisabled, b.NetworkDisabled) {
		return false
	}

	a.lastCompareField = "KeepVolumes"
	if !comparePointerBool(a.KeepVolumes, b.KeepVolumes) {
		return false
	}

	// Compare slices

	a.lastCompareField = "Dns"
	if !compareSliceString(a.Dns, b.Dns) {
		return false
	}

	a.lastCompareField = "AddHost"
	if !compareSliceString(a.AddHost, b.AddHost) {
		return false
	}

	a.lastCompareField = "Cmd"
	if !a.Cmd.IsEqualTo(b.Cmd) {
		return false
	}

	// TODO: consider order!
	a.lastCompareField = "Entrypoint"
	if !compareSliceString(a.Entrypoint, b.Entrypoint) {
		return false
	}

	a.lastCompareField = "Expose"
	if !compareSliceString(a.Expose, b.Expose) {
		return false
	}

	a.lastCompareField = "Volumes"
	if !compareSliceString(a.Volumes, b.Volumes) {
		return false
	}

	a.lastCompareField = "Ulimits"
	if !compareSliceUlimit(a.Ulimits, b.Ulimits) {
		return false
	}

	a.lastCompareField = "Ports"
	if !compareSlicePortBinding(a.Ports, b.Ports) {
		return false
	}

	a.lastCompareField = "VolumesFrom"
	if !compareSliceContainerName(a.VolumesFrom, b.VolumesFrom) {
		return false
	}

	a.lastCompareField = "Links"
	if !compareSliceContainerName(a.Links, b.Links) {
		return false
	}

	a.lastCompareField = "WaitFor"
	if !compareSliceContainerName(a.WaitFor, b.WaitFor) {
		return false
	}

	// Compare maps
	a.lastCompareField = "Labels"
	if !compareStringMap(a.Labels, b.Labels) {
		return false
	}

	a.lastCompareField = "Env"
	if !compareStringMap(a.Env, b.Env) {
		return false
	}

	return true
}

func (a *ContainerName) IsEqualTo(b *ContainerName) bool {
	return a.IsEqualNs(b) && a.Name == b.Name
}

func (a *ContainerName) IsEqualNs(b *ContainerName) bool {
	return a.Namespace == b.Namespace
}

func (a *ConfigState) IsEqualTo(b *ConfigState) bool {
	if a == nil {
		return b == a || *b == ""
	}
	if b == nil {
		return a == b || *a == ""
	}
	return *a == *b
}

func (a *ConfigCmd) IsEqualTo(b *ConfigCmd) bool {
	if a == nil {
		return b == a || b.IsEqualTo(&ConfigCmd{})
	}
	if b == nil {
		return a == b || a.IsEqualTo(&ConfigCmd{})
	}
	if len(a.Parts) != len(b.Parts) {
		return false
	}
	for i := 0; i < len(a.Parts); i++ {
		if a.Parts[i] != b.Parts[i] {
			return false
		}
	}
	return true
}

// Helper functions to compare pointer values used by ContainerConfig.IsEqualTo function

func comparePointerString(a, b *string) bool {
	if a == nil {
		return b == a || *b == ""
	}
	if b == nil {
		return a == b || *a == ""
	}
	return *a == *b
}

func comparePointerInt64(a, b *int64) bool {
	if a == nil {
		return b == a || *b == 0
	}
	if b == nil {
		return a == b || *a == 0
	}
	return *a == *b
}

func comparePointerInt(a, b *int) bool {
	if a == nil {
		return b == a || *b == 0
	}
	if b == nil {
		return a == b || *a == 0
	}
	return *a == *b
}

func comparePointerBool(a, b *bool) bool {
	if a == nil {
		return b == a || *b == false
	}
	if b == nil {
		return a == b || *a == false
	}
	return *a == *b
}

func comparePointerRestart(a, b *RestartPolicy) bool {
	if a == nil {
		return b == a || *b == RestartPolicy{}
	}
	if b == nil {
		return a == b || *a == RestartPolicy{}
	}
	return *a == *b
}

func comparePointerNet(a, b *Net) bool {
	if a == nil {
		return b == a || *b == Net{}
	}
	if b == nil {
		return a == b || *a == Net{}
	}
	return *a == *b
}

func comparePointerMemory(a, b *ConfigMemory) bool {
	if a == nil {
		return b == a || *b == 0
	}
	if b == nil {
		return a == b || *a == 0
	}
	return *a == *b
}

// Here we duplicate functions changing only argument types
// sadly, there is no way to do it better in Go

func compareSliceString(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	found := true
	for i := 0; i < len(a); i++ {
		elFound := false
		for k := 0; k < len(b); k++ {
			if a[i] == b[k] {
				elFound = true
				break
			}
		}
		if !elFound {
			found = false
			break
		}
	}
	return found
}

func compareSliceUlimit(a, b []ConfigUlimit) bool {
	if len(a) != len(b) {
		return false
	}
	found := true
	for i := 0; i < len(a); i++ {
		elFound := false
		for k := 0; k < len(b); k++ {
			if a[i] == b[k] {
				elFound = true
				break
			}
		}
		if !elFound {
			found = false
			break
		}
	}
	return found
}

func compareSlicePortBinding(a, b []PortBinding) bool {
	if len(a) != len(b) {
		return false
	}
	found := true
	for i := 0; i < len(a); i++ {
		elFound := false
		for k := 0; k < len(b); k++ {
			if a[i] == b[k] {
				elFound = true
				break
			}
		}
		if !elFound {
			found = false
			break
		}
	}
	return found
}

func compareSliceContainerName(a, b []ContainerName) bool {
	if len(a) != len(b) {
		return false
	}
	found := true
	for i := 0; i < len(a); i++ {
		elFound := false
		for k := 0; k < len(b); k++ {
			if a[i] == b[k] {
				elFound = true
				break
			}
		}
		if !elFound {
			found = false
			break
		}
	}
	return found
}

func compareStringMap(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if v != b[k] {
			return false
		}
	}
	return true
}
