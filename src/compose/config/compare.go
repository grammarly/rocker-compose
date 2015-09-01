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
	"reflect"
	"sort"

	"github.com/go-yaml/yaml"
)

// LastCompareField returns last equal compared field of IsEqualTo evaluation.
func (a *Container) LastCompareField() string {
	return a.lastCompareField
}

// IsEqualTo compares the container spec against another one.
// It returns false if at least one property is unequal.
func (a *Container) IsEqualTo(b *Container) bool {
	for _, field := range getComparableFields() {
		a.lastCompareField = field
		if equal, _ := compareYaml(field, a, b); !equal {
			// TODO: return err
			return false
		}
	}

	return true
}

// IsEqualTo compares the ContainerName against another one.
// namespace and name should be same.
func (a *ContainerName) IsEqualTo(b *ContainerName) bool {
	return a.IsEqualNs(b) && a.Name == b.Name
}

// IsEqualNs returns true if both containers have same namespace.
func (a *ContainerName) IsEqualNs(b *ContainerName) bool {
	return a.Namespace == b.Namespace
}

// internals

// TODO: here would be nice to say few words about our approach of container specs comparison.

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
		aSorted := newYamlSortable(av)
		sort.Sort(aSorted)
		av = reflect.ValueOf(aSorted)

		bSorted := newYamlSortable(bv)
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

type yamlSortable []interface{}

func newYamlSortable(slice reflect.Value) yamlSortable {
	sortable := yamlSortable{}
	for i := 0; i < slice.Len(); i++ {
		sortable = append(sortable, slice.Index(i).Interface())
	}
	return sortable
}

func (items yamlSortable) Len() int {
	return len(items)
}

func (items yamlSortable) Less(i, j int) bool {
	yml1, _ := yaml.Marshal(items[i])
	yml2, _ := yaml.Marshal(items[j])
	return string(yml1) < string(yml2)
}

func (items yamlSortable) Swap(i, j int) {
	items[i], items[j] = items[j], items[i]
}
