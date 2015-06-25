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
	c1 := &ConfigContainer{}
	c2 := &ConfigContainer{}
	assert.True(t, c1.IsEqualTo(c2), "empty configs should be equal")
}

func TestConfigIsEqualTo_PointerValue(t *testing.T) {
	var a, b int64
	a = 25
	b = 25
	c1 := &ConfigContainer{CpuShares: &a}
	c2 := &ConfigContainer{CpuShares: &b}
	assert.True(t, c1.IsEqualTo(c2), "configs with same pointer value should be equal")

	var c, d int64
	c = 25
	d = 26
	c3 := &ConfigContainer{CpuShares: &c}
	c4 := &ConfigContainer{CpuShares: &d}
	assert.False(t, c3.IsEqualTo(c4), "configs with different pointer value should be not equal")

	var a0 int64
	a0 = 25
	c5 := &ConfigContainer{CpuShares: &a0}
	c6 := &ConfigContainer{}
	assert.False(t, c5.IsEqualTo(c6), "configs with one pointer value present and one not should differ")

	var b0 int64
	b0 = 25
	c7 := &ConfigContainer{}
	c8 := &ConfigContainer{CpuShares: &b0}
	assert.False(t, c7.IsEqualTo(c8), "configs with one pointer value present and one not should differ")
}

// TODO: more EqualTo tests
