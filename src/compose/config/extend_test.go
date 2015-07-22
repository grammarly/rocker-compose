package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfigExtend(t *testing.T) {
	config, err := NewFromFile("testdata/compose.yml", configTestVars, map[string]interface{}{})
	if err != nil {
		t.Fatal(err)
	}

	// TODO: more config assertions
	assert.Equal(t, "patterns", config.Namespace)
	assert.Equal(t, "dockerhub.grammarly.io/patterns:1.9.2", *config.Containers["main2"].Image)

	// should be inherited
	assert.Equal(t, Strings{"8.8.8.8"}, config.Containers["main2"].Dns)
	// should be overriden
	assert.Equal(t, Strings{"capi.grammarly.com:127.0.0.2"}, config.Containers["main2"].AddHost)

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
