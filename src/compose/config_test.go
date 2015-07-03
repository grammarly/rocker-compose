package compose

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
	"unicode"

	"github.com/stretchr/testify/assert"
)

var (
	configTestVars = map[string]interface{}{
		"version": map[string]string{
			"patterns": "1.9.2",
		},
	}
)

func TestReadConfigFile(t *testing.T) {
	config, err := ReadConfigFile("testdata/compose.yml", configTestVars)
	if err != nil {
		t.Fatal(err)
	}

	// fmt.Printf("config: %q\n", config)

	// TODO: more config assertions
	assert.Equal(t, "patterns", config.Namespace)
	assert.Equal(t, "dockerhub.grammarly.io/patterns:1.9.2", *config.Containers["main"].Image)
	assert.Equal(t, "dockerhub.grammarly.io/patterns-config:latest", *config.Containers["config"].Image)
}

func TestConfigMemoryInt64(t *testing.T) {
	assertions := map[string]int64{
		"-1":   -1,
		"0":    0,
		"100":  100,
		"100x": 100,
		"100b": 100,
		"100k": 102400,
		"100m": 104857600,
		"100g": 107374182400,
	}
	for input, expected := range assertions {
		actual, err := NewConfigMemoryFromString(input)
		if err != nil {
			t.Fatal(err)
		}
		assert.EqualValues(t, expected, *actual)
	}
}

func TestConfigExtend(t *testing.T) {
	config, err := ReadConfigFile("testdata/compose.yml", configTestVars)
	if err != nil {
		t.Fatal(err)
	}

	// TODO: more config assertions
	assert.Equal(t, "patterns", config.Namespace)
	assert.Equal(t, "dockerhub.grammarly.io/patterns:1.9.2", *config.Containers["main2"].Image)

	// should be inherited
	assert.Equal(t, []string{"8.8.8.8"}, config.Containers["main2"].Dns)
	// should be overriden
	assert.Equal(t, []string{"capi.grammarly.com:127.0.0.2"}, config.Containers["main2"].AddHost)

	// should be inherited
	assert.EqualValues(t, 512, *config.Containers["main2"].CpuShares)

	// should inherit and merge labels
	assert.Equal(t, 3, len(config.Containers["main2"].Labels))
	assert.Equal(t, "pattern", config.Containers["main2"].Labels["service"])
	assert.Equal(t, "2", config.Containers["main2"].Labels["num"])
	assert.Equal(t, "replica", config.Containers["main2"].Labels["type"])

	// should not affect parent labels
	assert.Equal(t, 2, len(config.Containers["main"].Labels))
	assert.Equal(t, "pattern", config.Containers["main"].Labels["service"])
	assert.Equal(t, "1", config.Containers["main"].Labels["num"])

	// should be overriden
	assert.EqualValues(t, 200, *config.Containers["main2"].KillTimeout)
}

func TestConfigIsEqualTo_Empty(t *testing.T) {
	var c1, c2 *ConfigContainer
	c1 = &ConfigContainer{}
	c2 = &ConfigContainer{}
	assert.True(t, c1.IsEqualTo(c2), "empty configs should be equal")
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
		c1, c2 *ConfigContainer

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

		aContainerName = ContainerName{"app", "main", ""}
		bContainerName = ContainerName{"app", "config", ""}

		aConfigCmd = &ConfigCmd{[]string{"foo"}}
		bConfigCmd = &ConfigCmd{[]string{"foo", "bar"}}
		cConfigCmd = &ConfigCmd{[]string{"bar", "foo"}}
		dConfigCmd = &ConfigCmd{}

		aMap = map[string]string{"foo": "bar"}
		bMap = map[string]string{"xxx": "yyy"}
		cMap = map[string]string{"foo": "bar", "xxx": "yyy"}
		dMap = map[string]string{}
	)

	cases := tests{
		// type: *string
		fieldSpec{
			[]string{"Image", "Net", "Pid", "Uts", "State", "CpusetCpus", "Hostname", "Domainname", "User", "Workdir"},
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
			[]string{"OomKillDisable", "Privileged", "PublishAllPorts", "NetworkDisabled", "KeepVolumes"},
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
			[]string{"Dns", "AddHost", "Entrypoint", "Expose", "Volumes"},
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
			[]string{"VolumesFrom", "Links", "WaitFor"},
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
				c1 = &ConfigContainer{}
				c2 = &ConfigContainer{}

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

				printValueA := fmt.Sprintf("%+q", valuePair.a)
				printValueB := fmt.Sprintf("%+q", valuePair.b)

				message := fmt.Sprintf("ConfigContainer{%s: %s} %s ConfigContainer{%s: %s}",
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
	typeOfElem := reflect.ValueOf(&ConfigContainer{}).Elem().Type()
	for i := 0; i < typeOfElem.NumField(); i++ {
		fieldName := typeOfElem.Field(i).Name
		// Skip some fields
		if unicode.IsLower((rune)(fieldName[0])) {
			continue
		}
		if fieldName == "Extends" || fieldName == "KillTimeout" {
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

func TestConfigGetContainers(t *testing.T) {
	config, err := ReadConfigFile("testdata/compose.yml", configTestVars)
	if err != nil {
		t.Fatal(err)
	}

	containers := config.GetContainers()

	assert.Equal(t, 5, len(containers), "bad containers number from config")
}

func TestConfigCmdString(t *testing.T) {
	configStr := `namespace: test
containers:
  whoami:
    image: ubuntu
    cmd: whoami`

	config, err := ReadConfig("test", strings.NewReader(configStr), configTestVars)
	if err != nil {
		t.Fatal(err)
	}

	assert.NotNil(t, config.Containers["whoami"].Cmd)
	assert.Equal(t, []string{"/bin/sh", "-c", "whoami"}, config.Containers["whoami"].Cmd.Parts)
}
