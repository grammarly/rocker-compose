package config

import (
	"fmt"
	"reflect"
	"testing"
	"unicode"

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

	equal, err := compareReflect("CpuShares", c1, c2)
	if err != nil {
		t.Fatal(err)
	}
	assert.True(t, equal)
}

func TestConfigCompareReflectSlice(t *testing.T) {
	c1 := &Container{Dns: []string{"foo", "bar"}}
	c2 := &Container{Dns: []string{"bar", "foo"}}

	equal, err := compareReflect("Dns", c1, c2)
	if err != nil {
		t.Fatal(err)
	}
	assert.True(t, equal)
}

func TestConfigIsEqualTo(t *testing.T) {
	type (
		check struct {
			shouldEqual bool
			a           interface{}
			b           interface{}
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

		fooString   = "foo"
		barString   = "bar"
		emptyString = ""

		aInt64   int64 = 25
		bInt64   int64 = 25
		cInt64   int64 = 26
		nilInt64 int64 = 0

		aBool = true
		bBool = true
		cBool = false

		aUlimit = ConfigUlimit{"nofile", 1024, 2048}
		bUlimit = ConfigUlimit{"/app", 1024, 2048}

		aPortBinding = PortBinding{Port: "8000"}
		bPortBinding = PortBinding{Port: "9000"}

		aContainerName = ContainerName{"app", "main"}
		bContainerName = ContainerName{"app", "config"}

		aLink = Link{"app", "main", ""}
		bLink = Link{"app", "main", "alias"}
		cLink = Link{"app", "config", ""}

		aConfigCmd = &ConfigCmd{[]string{"foo"}}
		bConfigCmd = &ConfigCmd{[]string{"foo", "bar"}}
		cConfigCmd = &ConfigCmd{[]string{"bar", "foo"}}
		dConfigCmd = &ConfigCmd{}

		aNet = &Net{Type: "host"}
		bNet = &Net{Type: "bridge"}
		cNet = &Net{Type: "container", Container: *NewContainerNameFromString("asd")}
		dNet = &Net{}

		aMap = map[string]string{"foo": "bar"}
		bMap = map[string]string{"xxx": "yyy"}
		cMap = map[string]string{"foo": "bar", "xxx": "yyy"}
		dMap = map[string]string{}
	)

	cases := tests{
		// type: *string
		fieldSpec{
			[]string{"Image", "Pid", "Uts", "CpusetCpus", "Hostname", "Domainname", "User", "Workdir"},
			[]check{
				check{shouldEqual, &fooString, &fooString},
				check{shouldEqual, &emptyString, &emptyString},
				check{shouldNotEqual, &fooString, nil},
				check{shouldNotEqual, nil, &barString},
				check{shouldNotEqual, &fooString, &barString},
				check{shouldNotEqual, &fooString, &emptyString},
				check{shouldNotEqual, &emptyString, &barString},
			},
		},
		// type: *int64
		fieldSpec{
			[]string{"CpuShares", "Memory", "MemorySwap"},
			[]check{
				check{shouldEqual, &aInt64, &aInt64},
				check{shouldEqual, &aInt64, &bInt64},
				check{shouldEqual, &nilInt64, nil},
				check{shouldNotEqual, nil, &aInt64},
				check{shouldNotEqual, &aInt64, nil},
				check{shouldNotEqual, &aInt64, &cInt64},
			},
		},
		// type: *bool
		fieldSpec{
			[]string{"OomKillDisable", "Privileged", "PublishAllPorts"},
			[]check{
				check{shouldEqual, &aBool, &aBool},
				check{shouldEqual, &aBool, &bBool},
				check{shouldEqual, &cBool, nil},
				check{shouldNotEqual, nil, &aBool},
				check{shouldNotEqual, &aBool, nil},
				check{shouldNotEqual, &aBool, &cBool},
			},
		},
		// type: []string
		fieldSpec{
			[]string{"Dns", "AddHost", "Expose", "Volumes"},
			[]check{
				check{shouldEqual, []string{}, []string{}},
				check{shouldEqual, []string{}, nil},
				check{shouldEqual, nil, []string{}},
				check{shouldEqual, []string{"foo"}, []string{"foo"}},
				check{shouldEqual, []string{"foo", "bar"}, []string{"foo", "bar"}},
				check{shouldEqual, []string{"foo", "bar"}, []string{"bar", "foo"}},
				check{shouldNotEqual, []string{"foo"}, nil},
				check{shouldNotEqual, nil, []string{"foo"}},
				check{shouldNotEqual, []string{"foo", "bar"}, []string{"foo"}},
				check{shouldNotEqual, []string{"foo", "bar"}, []string{}},
			},
		},
		// type: []string  --- Entrypoint should be strict order!
		fieldSpec{
			[]string{"Entrypoint"},
			[]check{
				check{shouldEqual, []string{}, []string{}},
				check{shouldEqual, []string{}, nil},
				check{shouldEqual, nil, []string{}},
				check{shouldEqual, []string{"foo"}, []string{"foo"}},
				check{shouldEqual, []string{"foo", "bar"}, []string{"foo", "bar"}},
				check{shouldNotEqual, []string{"foo", "bar"}, []string{"bar", "foo"}},
				check{shouldNotEqual, []string{"foo"}, nil},
				check{shouldNotEqual, nil, []string{"foo"}},
				check{shouldNotEqual, []string{"foo", "bar"}, []string{"foo"}},
				check{shouldNotEqual, []string{"foo", "bar"}, []string{}},
			},
		},
		// type: ConfigCmd
		fieldSpec{
			[]string{"Cmd"},
			[]check{
				check{shouldEqual, dConfigCmd, dConfigCmd},
				check{shouldEqual, dConfigCmd, nil},
				check{shouldEqual, nil, dConfigCmd},
				check{shouldEqual, aConfigCmd, aConfigCmd},
				check{shouldEqual, bConfigCmd, bConfigCmd},
				check{shouldNotEqual, bConfigCmd, cConfigCmd},
				check{shouldNotEqual, aConfigCmd, nil},
				check{shouldNotEqual, nil, aConfigCmd},
				check{shouldNotEqual, bConfigCmd, aConfigCmd},
				check{shouldNotEqual, bConfigCmd, dConfigCmd},
			},
		},
		// type: RestartPolicy
		fieldSpec{
			[]string{"Restart"},
			[]check{
				check{shouldEqual, &RestartPolicy{}, &RestartPolicy{}},
				check{shouldEqual, &RestartPolicy{}, nil},
				check{shouldEqual, nil, &RestartPolicy{}},
				check{shouldEqual, &RestartPolicy{"always", 0}, &RestartPolicy{"always", 0}},
				check{shouldNotEqual, nil, &RestartPolicy{"always", 0}},
				check{shouldNotEqual, &RestartPolicy{"always", 0}, nil},
				check{shouldNotEqual, &RestartPolicy{"always", 0}, &RestartPolicy{}},
				check{shouldNotEqual, &RestartPolicy{}, &RestartPolicy{"always", 0}},
				check{shouldNotEqual, &RestartPolicy{"always", 0}, &RestartPolicy{"no", 0}},
			},
		},
		// type: Net
		fieldSpec{
			[]string{"Net"},
			[]check{
				check{shouldEqual, dNet, dNet},
				check{shouldEqual, dNet, nil},
				check{shouldEqual, nil, dNet},
				check{shouldEqual, aNet, aNet},
				check{shouldNotEqual, nil, aNet},
				check{shouldNotEqual, aNet, nil},
				check{shouldNotEqual, aNet, bNet},
				check{shouldNotEqual, aNet, dNet},
				check{shouldNotEqual, dNet, aNet},
				check{shouldNotEqual, aNet, cNet},
			},
		},
		// type: []ConfigUlimit
		fieldSpec{
			[]string{"Ulimits"},
			[]check{
				check{shouldEqual, []ConfigUlimit{}, []ConfigUlimit{}},
				check{shouldEqual, []ConfigUlimit{}, nil},
				check{shouldEqual, nil, []ConfigUlimit{}},
				check{shouldEqual, []ConfigUlimit{aUlimit}, []ConfigUlimit{aUlimit}},
				check{shouldEqual, []ConfigUlimit{aUlimit, bUlimit}, []ConfigUlimit{aUlimit, bUlimit}},
				check{shouldEqual, []ConfigUlimit{aUlimit, bUlimit}, []ConfigUlimit{bUlimit, aUlimit}},
				check{shouldNotEqual, []ConfigUlimit{aUlimit}, nil},
				check{shouldNotEqual, nil, []ConfigUlimit{aUlimit}},
				check{shouldNotEqual, []ConfigUlimit{aUlimit, bUlimit}, []ConfigUlimit{aUlimit}},
				check{shouldNotEqual, []ConfigUlimit{aUlimit, bUlimit}, []ConfigUlimit{}},
			},
		},
		// type: []PortBinding
		fieldSpec{
			[]string{"Ports"},
			[]check{
				check{shouldEqual, []PortBinding{}, []PortBinding{}},
				check{shouldEqual, []PortBinding{}, nil},
				check{shouldEqual, nil, []PortBinding{}},
				check{shouldEqual, []PortBinding{aPortBinding}, []PortBinding{aPortBinding}},
				check{shouldEqual, []PortBinding{aPortBinding, bPortBinding}, []PortBinding{aPortBinding, bPortBinding}},
				check{shouldEqual, []PortBinding{aPortBinding, bPortBinding}, []PortBinding{bPortBinding, aPortBinding}},
				check{shouldNotEqual, []PortBinding{aPortBinding}, nil},
				check{shouldNotEqual, nil, []PortBinding{aPortBinding}},
				check{shouldNotEqual, []PortBinding{aPortBinding, bPortBinding}, []PortBinding{aPortBinding}},
				check{shouldNotEqual, []PortBinding{aPortBinding, bPortBinding}, []PortBinding{}},
			},
		},
		// type: []ContainerName
		fieldSpec{
			[]string{"VolumesFrom", "WaitFor"},
			[]check{
				check{shouldEqual, []ContainerName{}, []ContainerName{}},
				check{shouldEqual, []ContainerName{}, nil},
				check{shouldEqual, nil, []ContainerName{}},
				check{shouldEqual, []ContainerName{aContainerName}, []ContainerName{aContainerName}},
				check{shouldEqual, []ContainerName{aContainerName, bContainerName}, []ContainerName{aContainerName, bContainerName}},
				check{shouldEqual, []ContainerName{aContainerName, bContainerName}, []ContainerName{bContainerName, aContainerName}},
				check{shouldNotEqual, []ContainerName{aContainerName}, nil},
				check{shouldNotEqual, nil, []ContainerName{aContainerName}},
				check{shouldNotEqual, []ContainerName{aContainerName, bContainerName}, []ContainerName{aContainerName}},
				check{shouldNotEqual, []ContainerName{aContainerName, bContainerName}, []ContainerName{}},
			},
		},
		// type: []Link
		fieldSpec{
			[]string{"Links"},
			[]check{
				check{shouldEqual, []Link{}, []Link{}},
				check{shouldEqual, []Link{}, nil},
				check{shouldEqual, nil, []Link{}},
				check{shouldEqual, []Link{aLink}, []Link{aLink}},
				check{shouldEqual, []Link{aLink, bLink}, []Link{aLink, bLink}},
				check{shouldEqual, []Link{aLink, bLink}, []Link{bLink, aLink}},
				check{shouldNotEqual, []Link{aLink}, nil},
				check{shouldNotEqual, nil, []Link{aLink}},
				check{shouldNotEqual, []Link{aLink}, []Link{bLink}},
				check{shouldNotEqual, []Link{aLink}, []Link{cLink}},
				check{shouldNotEqual, []Link{aLink, bLink}, []Link{aLink}},
				check{shouldNotEqual, []Link{aLink, bLink}, []Link{}},
			},
		},
		// type: map[string]string
		fieldSpec{
			[]string{"Labels", "Env"},
			[]check{
				check{shouldEqual, dMap, dMap},
				check{shouldEqual, dMap, nil},
				check{shouldEqual, nil, dMap},
				check{shouldEqual, aMap, aMap},
				check{shouldEqual, bMap, bMap},
				check{shouldNotEqual, cMap, nil},
				check{shouldNotEqual, nil, cMap},
				check{shouldNotEqual, aMap, bMap},
				check{shouldNotEqual, bMap, aMap},
				check{shouldNotEqual, aMap, cMap},
				check{shouldNotEqual, cMap, aMap},
				check{shouldNotEqual, cMap, bMap},
				check{shouldNotEqual, bMap, cMap},
			},
		},
	}

	for _, spec := range cases {
		for _, fieldName := range spec.fieldNames {
			for _, valuePair := range spec.checks {
				c1 = &Container{}
				c2 = &Container{}

				a := reflect.ValueOf(valuePair.a)
				b := reflect.ValueOf(valuePair.b)

				if reflect.TypeOf(valuePair.a) != nil {
					field := reflect.ValueOf(c1).Elem().FieldByName(fieldName)
					if field.Type() != reflect.TypeOf(valuePair.a) {
						a = a.Convert(field.Type())
					}
					field.Set(a)
				}
				if reflect.TypeOf(valuePair.b) != nil {
					field := reflect.ValueOf(c2).Elem().FieldByName(fieldName)
					if field.Type() != reflect.TypeOf(valuePair.b) {
						b = b.Convert(field.Type())
					}
					field.Set(b)
				}

				compareRule := map[bool]string{
					true:  "should be equal to",
					false: "should not equal",
				}[valuePair.shouldEqual]

				printValueA := "nil"
				if reflect.Indirect(a).IsValid() {
					printValueA = fmt.Sprintf("%+q", reflect.Indirect(a).Interface())
				}

				printValueB := "nil"
				if reflect.Indirect(b).IsValid() {
					printValueB = fmt.Sprintf("%+q", reflect.Indirect(b).Interface())
				}

				// printValueA := fmt.Sprintf("%+q", reflect.Indirect(a).String())
				// printValueB := fmt.Sprintf("%+q", reflect.Indirect(b).Interface())

				message := fmt.Sprintf("Container{%s: %s} %s Container{%s: %s}",
					fieldName, printValueA, compareRule, fieldName, printValueB)

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
	typeOfElem := reflect.ValueOf(&Container{}).Elem().Type()
	for i := 0; i < typeOfElem.NumField(); i++ {
		fieldName := typeOfElem.Field(i).Name
		// Skip some fields
		if unicode.IsLower((rune)(fieldName[0])) {
			continue
		}
		if fieldName == "Extends" || fieldName == "KillTimeout" ||
			fieldName == "NetworkDisabled" || fieldName == "State" || fieldName == "KeepVolumes" {
			continue
		}

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
