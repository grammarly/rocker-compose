package config

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/go-yaml/yaml"
	"github.com/stretchr/testify/assert"
)

func TestConfigIsEqualTo_Empty(t *testing.T) {
	var c1, c2 *Container
	c1 = &Container{}
	c2 = &Container{}
	assert.True(t, c1.IsEqualTo(c2), "empty configs should be equal")
}

func TestConfigCompareReflect(t *testing.T) {
	var aInt64 int64 = 0
	c1 := &Container{CpuShares: &aInt64}
	c2 := &Container{}

	equal, err := compareYaml("CpuShares", c1, c2)
	if err != nil {
		t.Fatal(err)
	}
	assert.True(t, equal)
}

func TestConfigCompareReflectSlice(t *testing.T) {
	c1 := &Container{Entrypoint: []string{"foo", "bar"}}
	c2 := &Container{Entrypoint: []string{"bar", "foo"}}

	equal, err := compareYaml("Entrypoint", c1, c2)
	if err != nil {
		t.Fatal(err)
	}
	assert.False(t, equal)
}

func TestConfigIsEqualTo(t *testing.T) {
	type (
		check struct {
			shouldEqual bool
			a           string
			b           string
		}
		fieldSpec struct {
			fieldNames []string
			checks     []check
		}
		tests []fieldSpec
	)

	var (
		c1, c2 *Container

		shouldEqual    = true
		shouldNotEqual = false
	)

	cases := tests{
		// type: string
		fieldSpec{
			[]string{"Image", "Pid", "Uts", "CpusetCpus", "Hostname", "Domainname", "User", "Workdir"},
			[]check{
				check{shouldEqual, "KEY: foo", "KEY: foo"},
				check{shouldEqual, "", ""},
				check{shouldNotEqual, "KEY: foo", ""},
				check{shouldNotEqual, "", "KEY: bar"},
				check{shouldNotEqual, "KEY: foo", "KEY: bar"},
				check{shouldNotEqual, "KEY: foo", ""},
				check{shouldNotEqual, "", "KEY: bar"},
			},
		},
		// type: numbers
		fieldSpec{
			[]string{"CpuShares"},
			[]check{
				check{shouldEqual, "KEY: 20", "KEY: 20"},
				check{shouldEqual, "", ""},
				check{shouldNotEqual, "KEY: 20", ""},
				check{shouldNotEqual, "", "KEY: 30"},
				check{shouldNotEqual, "KEY: 20", "KEY: 30"},
				check{shouldNotEqual, "KEY: 20", ""},
				check{shouldNotEqual, "", "KEY: 30"},
			},
		},
		// type: booleans
		fieldSpec{
			[]string{"OomKillDisable", "Privileged", "PublishAllPorts"},
			[]check{
				check{shouldEqual, "KEY: true", "KEY: true"},
				check{shouldEqual, "", ""},
				check{shouldEqual, "", "KEY: false"},
				check{shouldNotEqual, "KEY: true", ""},
				check{shouldNotEqual, "KEY: true", "KEY: false"},
				check{shouldNotEqual, "KEY: true", ""},
			},
		},
		// type: Net
		fieldSpec{
			[]string{"Net"},
			[]check{
				check{shouldEqual, "KEY: host", "KEY: host"},
				check{shouldEqual, "", ""},
				check{shouldNotEqual, "KEY: host", ""},
				check{shouldNotEqual, "", "KEY: bridge"},
				check{shouldNotEqual, "KEY: host", "KEY: bridge"},
				check{shouldNotEqual, "KEY: host", ""},
				check{shouldNotEqual, "", "KEY: bridge"},
			},
		},
		// type: ConfigMemory
		fieldSpec{
			[]string{"Memory", "MemorySwap"},
			[]check{
				check{shouldEqual, "KEY: 64m", "KEY: 64m"},
				check{shouldEqual, "KEY: 1024m", "KEY: 1g"},
				check{shouldEqual, "", ""},
				check{shouldNotEqual, "", "KEY: 64m"},
				check{shouldNotEqual, "KEY: 64m", ""},
				check{shouldNotEqual, "KEY: 64m", "KEY: 2g"},
			},
		},
		// type: []string
		fieldSpec{
			[]string{"Dns", "AddHost", "Expose", "Volumes", "VolumesFrom", "Links", "WaitFor", "Ports"},
			[]check{
				check{shouldEqual, "", ""},
				check{shouldEqual, "KEY:\n  - foo", "KEY:\n  - foo"},
				check{shouldEqual, "KEY:\n  - foo\n  - bar", "KEY:\n  - foo\n  - bar"},
				check{shouldEqual, "KEY:\n  - foo\n  - bar", "KEY:\n  - bar\n  - foo"},
				check{shouldNotEqual, "KEY:\n  - foo", ""},
				check{shouldNotEqual, "", "KEY:\n  - foo"},
				check{shouldNotEqual, "KEY:\n  - foo\n  - bar", "KEY:\n  - foo"},
				check{shouldNotEqual, "KEY:\n  - foo\n  - bar", ""},
			},
		},
		// type: []string -- ORDERED
		fieldSpec{
			[]string{"Cmd", "Entrypoint"},
			[]check{
				check{shouldEqual, "", ""},
				check{shouldEqual, "KEY:\n  - foo", "KEY:\n  - foo"},
				check{shouldEqual, "KEY:\n  - foo\n  - bar", "KEY:\n  - foo\n  - bar"},
				check{shouldNotEqual, "KEY:\n  - foo\n  - bar", "KEY:\n  - bar\n  - foo"},
				check{shouldNotEqual, "KEY:\n  - foo", ""},
				check{shouldNotEqual, "", "KEY:\n  - foo"},
				check{shouldNotEqual, "KEY:\n  - foo\n  - bar", "KEY:\n  - foo"},
				check{shouldNotEqual, "KEY:\n  - foo\n  - bar", ""},
			},
		},
		// type: RestartPolicy
		fieldSpec{
			[]string{"Restart"},
			[]check{
				check{shouldEqual, "", ""},
				check{shouldEqual, "KEY: always", "KEY: always"},
				check{shouldNotEqual, "", "KEY: always"},
				check{shouldNotEqual, "KEY: always", ""},
				check{shouldNotEqual, "KEY: always", "KEY: no"},
			},
		},
		// TODO: change ulimit YAML parsing
		// type: []Ulimits
		fieldSpec{
			[]string{"Ulimits"},
			[]check{
				check{shouldEqual, "", ""},
				check{shouldEqual, "KEY:\n  - name: nofile\n    soft: 1024\n    hard: 2048", "KEY:\n  - name: nofile\n    soft: 1024\n    hard: 2048"},
				check{shouldEqual, "KEY:\n  - name: nofile\n    soft: 1024\n    hard: 2048\n  - name: /app\n    soft: 1024\n    hard: 2048", "KEY:\n  - name: nofile\n    soft: 1024\n    hard: 2048\n  - name: /app\n    soft: 1024\n    hard: 2048"},
				check{shouldEqual, "KEY:\n  - name: nofile\n    soft: 1024\n    hard: 2048\n  - name: /app\n    soft: 1024\n    hard: 2048", "KEY:\n  - name: /app\n    soft: 1024\n    hard: 2048\n  - name: nofile\n    soft: 1024\n    hard: 2048\n"},
				check{shouldNotEqual, "KEY:\n  - name: nofile\n    soft: 1024\n    hard: 2048", ""},
				check{shouldNotEqual, "", "KEY:\n  - name: nofile\n    soft: 1024\n    hard: 2048"},
				check{shouldNotEqual, "KEY:\n  - name: nofile\n    soft: 1024\n    hard: 2048\n  - name: /app\n    soft: 1024\n    hard: 2048", "KEY:\n  - name: nofile\n    soft: 1024\n    hard: 2048"},
				check{shouldNotEqual, "KEY:\n  - name: nofile\n    soft: 1024\n    hard: 2048\n  - name: /app\n    soft: 1024\n    hard: 2048", ""},
			},
		},
		// type: map[string]string
		fieldSpec{
			[]string{"Labels", "Env"},
			[]check{
				check{shouldEqual, "", ""},
				check{shouldEqual, "KEY:\n  foo: bar", "KEY:\n  foo: bar"},
				check{shouldEqual, "KEY:\n  xxx: yyy", "KEY:\n  xxx: yyy"},
				check{shouldNotEqual, "KEY:\n  foo: bar\n  xxx: yyy", ""},
				check{shouldNotEqual, "", "KEY:\n  foo: bar\n  xxx: yyy"},
				check{shouldNotEqual, "KEY:\n  foo: bar", "KEY:\n  xxx: yyy"},
				check{shouldNotEqual, "KEY:\n  xxx: yyy", "KEY:\n  foo: bar"},
				check{shouldNotEqual, "KEY:\n  foo: bar", "KEY:\n  foo: bar\n  xxx: yyy"},
				check{shouldNotEqual, "KEY:\n  foo: bar\n  xxx: yyy", "KEY:\n  foo: bar"},
				check{shouldNotEqual, "KEY:\n  foo: bar\n  xxx: yyy", "KEY:\n  xxx: yyy"},
				check{shouldNotEqual, "KEY:\n  xxx: yyy", "KEY:\n  foo: bar\n  xxx: yyy"},
			},
		},
	}

	for _, spec := range cases {
		for _, fieldName := range spec.fieldNames {
			for _, valuePair := range spec.checks {
				c1 = &Container{}
				c2 = &Container{}

				// read yaml field name
				field, _ := reflect.TypeOf(Container{}).FieldByName(fieldName)
				yamlTag := field.Tag.Get("yaml")
				split := strings.SplitN(yamlTag, ",", 2)
				ymlFieldName := split[0]

				av := strings.Replace(valuePair.a, "KEY", ymlFieldName, 1)
				bv := strings.Replace(valuePair.b, "KEY", ymlFieldName, 1)

				if err := yaml.Unmarshal([]byte(av), c1); err != nil {
					t.Fatal(err)
				}
				if err := yaml.Unmarshal([]byte(bv), c2); err != nil {
					t.Fatal(err)
				}

				compareRule := map[bool]string{
					true:  "should be equal to",
					false: "should not equal",
				}[valuePair.shouldEqual]

				message := fmt.Sprintf("Container{%q} %s Container{%q}", av, compareRule, bv)

				t.Logf("check: %s", message)

				if valuePair.shouldEqual {
					assert.True(t, c1.IsEqualTo(c2), message)
				} else {
					assert.False(t, c1.IsEqualTo(c2), message)
				}
			}
		}
	}

	// test that all fields are checked
	for _, fieldName := range GetComparableFields() {
		found := false
		for _, spec := range cases {
			for _, specFieldName := range spec.fieldNames {
				if specFieldName == fieldName {
					found = true
					break
				}
			}
		}
		assert.True(t, found, fmt.Sprintf("missing compare check for field: %s", fieldName))
	}
}
