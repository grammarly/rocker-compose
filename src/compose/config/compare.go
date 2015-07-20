package config

import (
	"reflect"
	"sort"

	"github.com/go-yaml/yaml"
)

func (a *Container) LastCompareField() string {
	return a.lastCompareField
}

func (a *Container) IsEqualTo(b *Container) bool {
	compareFields := []string{
		"Image", "Net", "Pid", "Uts", "Dns",
		"AddHost", "Restart", "Memory", "MemorySwap",
		"CpuShares", "CpusetCpus", "OomKillDisable",
		"Ulimits", "Privileged", "Cmd", "Entrypoint",
		"Expose", "Ports", "PublishAllPorts",
		"Labels", "Env", "VolumesFrom", "Volumes",
		"Links", "WaitFor", "Hostname", "Domainname",
		"User", "Workdir",
	}

	for _, field := range compareFields {
		a.lastCompareField = field
		if equal, _ := compareReflect(field, a, b); !equal {
			// TODO: return err
			return false
		}
	}

	return true
}

func (a *ContainerName) IsEqualTo(b *ContainerName) bool {
	return a.IsEqualNs(b) && a.Name == b.Name
}

func (a *ContainerName) IsEqualNs(b *ContainerName) bool {
	return a.Namespace == b.Namespace
}

// internals

func compareReflect(name string, a, b *Container) (bool, error) {
	av := reflect.Indirect(reflect.ValueOf(a)).FieldByName(name)
	bv := reflect.Indirect(reflect.ValueOf(b)).FieldByName(name)
	a1 := reflect.ValueOf(&Container{})
	b1 := reflect.ValueOf(&Container{})

	// empty values and nil pointer should be considered equal
	if av.IsNil() && av.Type().Kind() != reflect.Slice && av.Type().Kind() != reflect.Map {
		av = reflect.New(av.Type().Elem())
	}
	if bv.IsNil() && bv.Type().Kind() != reflect.Slice && bv.Type().Kind() != reflect.Map {
		bv = reflect.New(bv.Type().Elem())
	}

	aField := a1.Elem().FieldByName(name)
	bField := b1.Elem().FieldByName(name)

	aField.Set(av)
	bField.Set(bv)

	// TODO: remove Entrypoint from here!
	// sort lists which should not consider different order to be a change
	if name == "Dns" || name == "AddHost" || name == "Links" ||
		name == "Expose" || name == "Volumes" || name == "Ulimits" ||
		name == "Ports" || name == "VolumesFrom" || name == "WaitFor" {

		aSorted := NewYamlSortable(aField)
		sort.Sort(aSorted)
		a1 = reflect.ValueOf(aSorted)

		bSorted := NewYamlSortable(bField)
		sort.Sort(bSorted)
		b1 = reflect.ValueOf(bSorted)
	}

	yml1, err := yaml.Marshal(a1.Interface())
	if err != nil {
		return false, err
	}
	yml2, err := yaml.Marshal(b1.Interface())
	if err != nil {
		return false, err
	}

	return string(yml1) == string(yml2), nil
}

type YamlSortable []interface{}

func NewYamlSortable(slice reflect.Value) YamlSortable {
	sortable := YamlSortable{}
	for i := 0; i < slice.Len(); i++ {
		sortable = append(sortable, slice.Index(i).Interface())
	}
	return sortable
}

func (items YamlSortable) Len() int {
	return len(items)
}

func (items YamlSortable) Less(i, j int) bool {
	yml1, _ := yaml.Marshal(items[i])
	yml2, _ := yaml.Marshal(items[j])
	return string(yml1) < string(yml2)
}

func (items YamlSortable) Swap(i, j int) {
	items[i], items[j] = items[j], items[i]
}
