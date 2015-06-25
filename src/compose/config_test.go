package compose

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReadConfigFile(t *testing.T) {
	config, err := ReadConfigFile("testdata/compose.yml")
	if err != nil {
		t.Fatal(err)
	}

	// fmt.Printf("config: %q\n", config)

	// TODO: more config assertions
	assert.Equal(t, "patterns", config.Namespace)
	assert.Equal(t, "dockerhub.grammarly.io/patterns:{{patterns_version}}", config.Containers["main"].Image)
	assert.Equal(t, "dockerhub.grammarly.io/patterns-config:{{patterns_config_version}}", config.Containers["config"].Image)
}

func TestConfigExtend(t *testing.T) {
	config, err := ReadConfigFile("testdata/compose.yml")
	if err != nil {
		t.Fatal(err)
	}

	// TODO: more config assertions
	assert.Equal(t, "patterns", config.Namespace)
	assert.Equal(t, "dockerhub.grammarly.io/patterns:{{patterns_version}}", config.Containers["main2"].Image)

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

func TestConfigIsEqualTo_SimpleValue(t *testing.T) {
	var c1, c2 *ConfigContainer
	c1 = &ConfigContainer{Image: "foo"}
	c2 = &ConfigContainer{Image: "foo"}
	assert.True(t, c1.IsEqualTo(c2), "configs with same value should be equal")

	c1 = &ConfigContainer{Image: "foo"}
	c2 = &ConfigContainer{Image: "bar"}
	assert.False(t, c1.IsEqualTo(c2), "configs with same value should be equal")

	c1 = &ConfigContainer{Image: "foo"}
	c2 = &ConfigContainer{}
	assert.False(t, c1.IsEqualTo(c2), "configs with one value missiong should be not equal")

	c1 = &ConfigContainer{}
	c2 = &ConfigContainer{Image: "bar"}
	assert.False(t, c1.IsEqualTo(c2), "configs with one value missiong should be not equal")
}

func TestConfigIsEqualTo_PointerValue(t *testing.T) {
	var c1, c2 *ConfigContainer
	var a, b int64
	a = 25
	b = 25
	c1 = &ConfigContainer{CpuShares: &a}
	c2 = &ConfigContainer{CpuShares: &b}
	assert.True(t, c1.IsEqualTo(c2), "configs with same pointer value should be equal")

	b = 26
	c1 = &ConfigContainer{CpuShares: &a}
	c2 = &ConfigContainer{CpuShares: &b}
	assert.False(t, c1.IsEqualTo(c2), "configs with different pointer value should be not equal")

	c1 = &ConfigContainer{CpuShares: &a}
	c2 = &ConfigContainer{}
	assert.False(t, c1.IsEqualTo(c2), "configs with one pointer value present and one not should differ")

	c1 = &ConfigContainer{}
	c2 = &ConfigContainer{CpuShares: &b}
	assert.False(t, c1.IsEqualTo(c2), "configs with one pointer value present and one not should differ")
}

func TestConfigIsEqualTo_Slices(t *testing.T) {
	var c1, c2 *ConfigContainer
	c1 = &ConfigContainer{Dns: []string{"8.8.8.8"}}
	c2 = &ConfigContainer{Dns: []string{"8.8.8.8"}}
	assert.True(t, c1.IsEqualTo(c2), "configs with same slice should be equal")

	s := []string{"8.8.8.8"}
	c1 = &ConfigContainer{Dns: s}
	c2 = &ConfigContainer{Dns: s}
	assert.True(t, c1.IsEqualTo(c2), "configs with same slice var be equal")

	c1 = &ConfigContainer{Dns: []string{}}
	c2 = &ConfigContainer{}
	assert.True(t, c1.IsEqualTo(c2), "configs with same one slice absent and empty slice should be equal")

	c1 = &ConfigContainer{}
	c2 = &ConfigContainer{Dns: []string{}}
	assert.True(t, c1.IsEqualTo(c2), "configs with same one slice absent and empty slice should be equal")

	c1 = &ConfigContainer{Dns: []string{"8.8.8.8", "127.0.0.1"}}
	c2 = &ConfigContainer{Dns: []string{"8.8.8.8"}}
	assert.False(t, c1.IsEqualTo(c2), "configs with same different slice length should be not equal")

	c1 = &ConfigContainer{Dns: []string{"8.8.8.8"}}
	c2 = &ConfigContainer{Dns: []string{"8.8.8.8", "127.0.0.1"}}
	assert.False(t, c1.IsEqualTo(c2), "configs with same different slice length should be not equal")

	c1 = &ConfigContainer{Dns: []string{"8.8.8.8"}}
	c2 = &ConfigContainer{Dns: []string{}}
	assert.False(t, c1.IsEqualTo(c2), "configs with same different slice length should be not equal")

	c1 = &ConfigContainer{Dns: []string{"8.8.8.8"}}
	c2 = &ConfigContainer{Dns: []string{"127.0.0.1"}}
	assert.False(t, c1.IsEqualTo(c2), "configs with same different slice values should be not equal")

	c1 = &ConfigContainer{}
	c2 = &ConfigContainer{Dns: []string{"127.0.0.1"}}
	assert.False(t, c1.IsEqualTo(c2), "configs with same one slice absent should be not equal")

	c1 = &ConfigContainer{Dns: []string{"8.8.8.8"}}
	c2 = &ConfigContainer{}
	assert.False(t, c1.IsEqualTo(c2), "configs with same one slice absent should be not equal")
}

func TestConfigIsEqualTo_Maps(t *testing.T) {
	var c1, c2 *ConfigContainer

	c1 = &ConfigContainer{Labels: map[string]string{"foo": "bar"}}
	c2 = &ConfigContainer{Labels: map[string]string{"foo": "bar"}}
	assert.True(t, c1.IsEqualTo(c2), "configs with same maps should be equal")

	c1 = &ConfigContainer{Labels: map[string]string{}}
	c2 = &ConfigContainer{}
	assert.True(t, c1.IsEqualTo(c2), "configs with same one map absent and empty map should be equal")

	c1 = &ConfigContainer{}
	c2 = &ConfigContainer{Labels: map[string]string{}}
	assert.True(t, c1.IsEqualTo(c2), "configs with same one map absent and empty map should be equal")

	c1 = &ConfigContainer{Labels: map[string]string{"foo": "bar"}}
	c2 = &ConfigContainer{Labels: map[string]string{}}
	assert.False(t, c1.IsEqualTo(c2), "configs with different maps should be not equal")

	c1 = &ConfigContainer{Labels: map[string]string{}}
	c2 = &ConfigContainer{Labels: map[string]string{"foo": "bar"}}
	assert.False(t, c1.IsEqualTo(c2), "configs with different maps should be not equal")

	c1 = &ConfigContainer{Labels: map[string]string{"xxx": "yyy"}}
	c2 = &ConfigContainer{Labels: map[string]string{"foo": "bar"}}
	assert.False(t, c1.IsEqualTo(c2), "configs with different maps of same length should be not equal")

	c1 = &ConfigContainer{Labels: map[string]string{"foo": "bar", "xxx": "yyy"}}
	c2 = &ConfigContainer{Labels: map[string]string{"foo": "bar"}}
	assert.False(t, c1.IsEqualTo(c2), "configs with different maps of same length should be not equal")

	c1 = &ConfigContainer{Labels: map[string]string{"foo": "bar"}}
	c2 = &ConfigContainer{Labels: map[string]string{"foo": "bar", "xxx": "yyy"}}
	assert.False(t, c1.IsEqualTo(c2), "configs with different maps of same length should be not equal")
}
