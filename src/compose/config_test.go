package compose

import (
	"fmt"
	"reflect"
	"testing"

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
	assert.Equal(t, "dockerhub.grammarly.io/patterns:1.9.2", config.Containers["main"].Image)
	assert.Equal(t, "dockerhub.grammarly.io/patterns-config:latest", config.Containers["config"].Image)
}

func TestConfigMemoryInt64(t *testing.T) {
	assert.EqualValues(t, -1, (ConfigMemory)("-1").Int64())
	assert.EqualValues(t, 0, (ConfigMemory)("0").Int64())
	assert.EqualValues(t, 100, (ConfigMemory)("100").Int64())
	assert.EqualValues(t, 100, (ConfigMemory)("100x").Int64())
	assert.EqualValues(t, 100, (ConfigMemory)("100b").Int64())
	assert.EqualValues(t, 102400, (ConfigMemory)("100k").Int64())
	assert.EqualValues(t, 104857600, (ConfigMemory)("100m").Int64())
	assert.EqualValues(t, 107374182400, (ConfigMemory)("100g").Int64())
}

func TestConfigExtend(t *testing.T) {
	config, err := ReadConfigFile("testdata/compose.yml", configTestVars)
	if err != nil {
		t.Fatal(err)
	}

	// TODO: more config assertions
	assert.Equal(t, "patterns", config.Namespace)
	assert.Equal(t, "dockerhub.grammarly.io/patterns:1.9.2", config.Containers["main2"].Image)

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

		aInt int = 25
		bInt int = 25
		cInt int = 26

		aInt64 int64 = 25
		bInt64 int64 = 25
		cInt64 int64 = 26

		aBool = true
		bBool = true
		cBool = false

		aUlimit = ConfigUlimit{"nofile", 1024, 2048}
		bUlimit = ConfigUlimit{"/app", 1024, 2048}

		aPortBinding = (PortBinding)("8000")
		bPortBinding = (PortBinding)("9000")

		aContainerName = ContainerName{"app", "main"}
		bContainerName = ContainerName{"app", "config"}

		aMap = map[string]string{"foo": "bar"}
		bMap = map[string]string{"xxx": "yyy"}
		cMap = map[string]string{"foo": "bar", "xxx": "yyy"}
		dMap = map[string]string{}
	)

	cases := tests{
		// type: string
		fieldSpec{
			[]string{"Image", "Net", "Pid", "Uts", "State", "CpusetCpus", "Hostname", "Domainname", "User", "Workdir"},
			[]check{
				check{shouldEqual, "foo", "foo"},
				check{shouldEqual, "", ""},
				check{shouldNotEqual, "foo", "bar"},
				check{shouldNotEqual, "foo", ""},
				check{shouldNotEqual, "foo", nil},
				check{shouldNotEqual, "", "bar"},
				check{shouldNotEqual, nil, "bar"},
			},
		},
		// type: *int
		fieldSpec{
			[]string{"KillTimeout"},
			[]check{
				check{shouldEqual, &aInt, &aInt},
				check{shouldEqual, &aInt, &bInt},
				check{shouldNotEqual, &aInt, &cInt},
				check{shouldNotEqual, &aInt, nil},
				check{shouldNotEqual, nil, &aInt},
			},
		},
		// type: *int64
		fieldSpec{
			[]string{"CpuShares"},
			[]check{
				check{shouldEqual, &aInt64, &aInt64},
				check{shouldEqual, &aInt64, &bInt64},
				check{shouldNotEqual, &aInt64, &cInt64},
				check{shouldNotEqual, &aInt64, nil},
				check{shouldNotEqual, nil, &aInt64},
			},
		},
		// type: *bool
		fieldSpec{
			[]string{"OomKillDisable", "Privileged", "PublishAllPorts", "NetworkDisabled", "KeepVolumes"},
			[]check{
				check{shouldEqual, &aBool, &aBool},
				check{shouldEqual, &aBool, &bBool},
				check{shouldNotEqual, &aBool, &cBool},
				check{shouldNotEqual, &aBool, nil},
				check{shouldNotEqual, nil, &aBool},
			},
		},
		// type: []string
		fieldSpec{
			[]string{"Dns", "AddHost", "Cmd", "Entrypoint", "Expose", "Volumes"},
			[]check{
				check{shouldEqual, []string{}, []string{}},
				check{shouldEqual, []string{}, nil},
				check{shouldEqual, nil, []string{}},
				check{shouldEqual, []string{"foo"}, []string{"foo"}},
				check{shouldEqual, []string{"foo", "bar"}, []string{"foo", "bar"}},
				check{shouldNotEqual, []string{"foo", "bar"}, []string{"bar", "foo"}},
				check{shouldNotEqual, []string{"foo", "bar"}, []string{"foo"}},
				check{shouldNotEqual, []string{"foo", "bar"}, []string{}},
				check{shouldNotEqual, []string{"foo"}, nil},
				check{shouldNotEqual, nil, []string{"foo"}},
			},
		},
		// type: RestartPolicy
		fieldSpec{
			[]string{"Restart"},
			[]check{
				check{shouldEqual, (RestartPolicy)(""), (RestartPolicy)("")},
				check{shouldEqual, (RestartPolicy)("always"), (RestartPolicy)("always")},
				check{shouldNotEqual, (RestartPolicy)("always"), (RestartPolicy)("")},
				check{shouldNotEqual, (RestartPolicy)(""), (RestartPolicy)("always")},
				check{shouldNotEqual, (RestartPolicy)("always"), (RestartPolicy)("no")},
				check{shouldNotEqual, (RestartPolicy)("always"), nil},
				check{shouldNotEqual, nil, (RestartPolicy)("always")},
			},
		},
		// type: ConfigMemory
		fieldSpec{
			[]string{"Memory", "MemorySwap"},
			[]check{
				check{shouldEqual, (ConfigMemory)(""), (ConfigMemory)("")},
				check{shouldEqual, (ConfigMemory)("100"), (ConfigMemory)("100")},
				check{shouldEqual, (ConfigMemory)(""), nil},
				check{shouldEqual, nil, (ConfigMemory)("")},
				check{shouldNotEqual, (ConfigMemory)("100"), (ConfigMemory)("100b")},
				check{shouldNotEqual, (ConfigMemory)("100"), (ConfigMemory)("")},
				check{shouldNotEqual, (ConfigMemory)(""), (ConfigMemory)("100")},
				check{shouldNotEqual, (ConfigMemory)("100"), (ConfigMemory)("")},
				check{shouldNotEqual, (ConfigMemory)("-1"), (ConfigMemory)("")},
				check{shouldNotEqual, (ConfigMemory)("-1"), (ConfigMemory)("0")},
				check{shouldNotEqual, (ConfigMemory)(""), (ConfigMemory)("0")},
				check{shouldNotEqual, (ConfigMemory)("0"), (ConfigMemory)("")},
				check{shouldNotEqual, (ConfigMemory)("0"), nil},
				check{shouldNotEqual, nil, (ConfigMemory)("0")},
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
				check{shouldNotEqual, []ConfigUlimit{aUlimit, bUlimit}, []ConfigUlimit{bUlimit, aUlimit}},
				check{shouldNotEqual, []ConfigUlimit{aUlimit, bUlimit}, []ConfigUlimit{aUlimit}},
				check{shouldNotEqual, []ConfigUlimit{aUlimit, bUlimit}, []ConfigUlimit{}},
				check{shouldNotEqual, []ConfigUlimit{aUlimit}, nil},
				check{shouldNotEqual, nil, []ConfigUlimit{aUlimit}},
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
				check{shouldNotEqual, []PortBinding{aPortBinding, bPortBinding}, []PortBinding{bPortBinding, aPortBinding}},
				check{shouldNotEqual, []PortBinding{aPortBinding, bPortBinding}, []PortBinding{aPortBinding}},
				check{shouldNotEqual, []PortBinding{aPortBinding, bPortBinding}, []PortBinding{}},
				check{shouldNotEqual, []PortBinding{aPortBinding}, nil},
				check{shouldNotEqual, nil, []PortBinding{aPortBinding}},
			},
		},
		// type: []ContainerName
		fieldSpec{
			[]string{"VolumesFrom", "Links"},
			[]check{
				check{shouldEqual, []ContainerName{}, []ContainerName{}},
				check{shouldEqual, []ContainerName{}, nil},
				check{shouldEqual, nil, []ContainerName{}},
				check{shouldEqual, []ContainerName{aContainerName}, []ContainerName{aContainerName}},
				check{shouldEqual, []ContainerName{aContainerName, bContainerName}, []ContainerName{aContainerName, bContainerName}},
				check{shouldNotEqual, []ContainerName{aContainerName, bContainerName}, []ContainerName{bContainerName, aContainerName}},
				check{shouldNotEqual, []ContainerName{aContainerName, bContainerName}, []ContainerName{aContainerName}},
				check{shouldNotEqual, []ContainerName{aContainerName, bContainerName}, []ContainerName{}},
				check{shouldNotEqual, []ContainerName{aContainerName}, nil},
				check{shouldNotEqual, nil, []ContainerName{aContainerName}},
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
				check{shouldNotEqual, aMap, bMap},
				check{shouldNotEqual, bMap, aMap},
				check{shouldNotEqual, aMap, cMap},
				check{shouldNotEqual, cMap, aMap},
				check{shouldNotEqual, cMap, nil},
				check{shouldNotEqual, nil, cMap},
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
					reflect.ValueOf(c1).Elem().FieldByName(fieldName).Set(a)
				}
				if reflect.TypeOf(valuePair.b) != nil {
					reflect.ValueOf(c2).Elem().FieldByName(fieldName).Set(b)
				}

				compareRule := map[bool]string{
					true:  "should be equal to",
					false: "should not equal",
				}[valuePair.shouldEqual]

				message := fmt.Sprintf("ConfigContainer{%s: %+q} %s ConfigContainer{%s: %+q}",
					fieldName, valuePair.a, compareRule, fieldName, valuePair.b)

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
		// Skip "Extends" field - not compared
		if fieldName == "Extends" {
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

	assert.Equal(t, 4, len(containers), "bad containers number from config")
}
