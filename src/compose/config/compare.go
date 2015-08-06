package config

import (
	"reflect"
	"sort"
	"unicode"

	"github.com/go-yaml/yaml"
)

var CompareSkipFields = []string{
	"Extends",
	"KillTimeout",
	"NetworkDisabled",
	"State",
	"KeepVolumes",

	// aliases
	"Command",
	"Link",
	"Label",
	"Hosts",
	"WorkingDir",
	"Environment",
}

func GetComparableFields() []string {
	fields := []string{}

	typeOfElem := reflect.ValueOf(&Container{}).Elem().Type()
	for i := 0; i < typeOfElem.NumField(); i++ {
		fieldName := typeOfElem.Field(i).Name
		// Skip some fields
		if unicode.IsLower((rune)(fieldName[0])) {
			continue
		}

		skip := false
		for _, f := range CompareSkipFields {
			if f == fieldName {
				skip = true
				break
			}
		}

		if !skip {
			fields = append(fields, fieldName)
		}
	}

	return fields
}

func (a *Container) LastCompareField() string {
	return a.lastCompareField
}

func (a *Container) IsEqualTo(b *Container) bool {
	for _, field := range GetComparableFields() {
		a.lastCompareField = field
		if equal, _ := compareYaml(field, a, b); !equal {
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

func compareYaml(name string, a, b *Container) (bool, error) {
	av := reflect.Indirect(reflect.ValueOf(a)).FieldByName(name)
	bv := reflect.Indirect(reflect.ValueOf(b)).FieldByName(name)

	isSlice := av.Type().Kind() == reflect.Slice
	isMap := av.Type().Kind() == reflect.Map

	// empty values and nil pointer should be considered equal
	if av.IsNil() && !isSlice && !isMap {
		av = reflect.New(av.Type().Elem())
	}
	if bv.IsNil() && !isSlice && !isMap {
		bv = reflect.New(bv.Type().Elem())
	}

	// sort lists which should not consider different order to be a change
	if isSlice && name != "Entrypoint" && name != "Cmd" {
		aSorted := NewYamlSortable(av)
		sort.Sort(aSorted)
		av = reflect.ValueOf(aSorted)

		bSorted := NewYamlSortable(bv)
		sort.Sort(bSorted)
		bv = reflect.ValueOf(bSorted)
	}

	yml1, err := yaml.Marshal(av.Interface())
	if err != nil {
		return false, err
	}
	yml2, err := yaml.Marshal(bv.Interface())
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
