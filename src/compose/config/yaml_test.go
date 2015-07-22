package config

import (
	"strings"
	"testing"

	"github.com/go-yaml/yaml"
	"github.com/stretchr/testify/assert"
)

func TestYamlContainerName(t *testing.T) {
	assertions := map[string]string{
		"platform.statsd":  "platform.statsd",
		"/platform.statsd": "platform.statsd",
		"statsd":           "statsd",
		".statsd":          "statsd",
		"":                 "\"\"",
	}

	for inYaml, outYaml := range assertions {
		v := ContainerName{}
		if err := yaml.Unmarshal([]byte(inYaml), &v); err != nil {
			t.Fatal(err)
		}
		data, err := yaml.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, outYaml, strings.TrimSpace(string(data)))
	}
}

func TestYamlLink(t *testing.T) {
	assertions := map[string]string{
		"platform.statsd":          "platform.statsd:statsd",
		"platform.statsd:metrics":  "platform.statsd:metrics",
		"/platform.statsd":         "platform.statsd:statsd",
		"/platform.statsd:metrics": "platform.statsd:metrics",
		"statsd":                   "statsd:statsd",
		"statsd:metrics":           "statsd:metrics",
		".statsd":                  "statsd:statsd",
		".statsd:metrics":          "statsd:metrics",
		"":                         "\"\"",
	}

	for inYaml, outYaml := range assertions {
		v := Link{}
		if err := yaml.Unmarshal([]byte(inYaml), &v); err != nil {
			t.Fatal(err)
		}
		data, err := yaml.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, outYaml, strings.TrimSpace(string(data)))
	}
}

func TestYamlMemory(t *testing.T) {
	assertions := map[string]string{
		"":     "0",
		"0":    "0",
		"300":  "300",
		"300b": "300",
		"1k":   "1024",
		"1m":   "1048576",
		"1g":   "1073741824",
	}

	for inYaml, outYaml := range assertions {
		var v ConfigMemory
		if err := yaml.Unmarshal([]byte(inYaml), &v); err != nil {
			t.Fatal(err)
		}
		data, err := yaml.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, outYaml, strings.TrimSpace(string(data)))
	}
}

func TestYamlRestartPolicy(t *testing.T) {
	assertions := map[string]string{
		"":             "\"no\"",
		"no":           "\"no\"",
		"always":       "always",
		"on-failure,5": "on-failure,5",
		"on-failure":   "on-failure,0",
	}

	for inYaml, outYaml := range assertions {
		v := &RestartPolicy{}
		if err := yaml.Unmarshal([]byte(inYaml), v); err != nil {
			t.Fatal(err)
		}
		data, err := yaml.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, outYaml, strings.TrimSpace(string(data)))
	}
}

func TestYamlPortBinding(t *testing.T) {
	assertions := map[string]string{
		"":                      "\"\"",
		"8000":                  "8000/tcp",
		"8125/udp":              "8125/udp",
		"8081:8000":             "8081:8000/tcp",
		"8126:8125/udp":         "8126:8125/udp",
		"0.0.0.0::5959":         "0.0.0.0::5959/tcp",
		"0.0.0.0:8081:80":       "0.0.0.0:8081:80/tcp",
		"0.0.0.0:8126:8125/udp": "0.0.0.0:8126:8125/udp",
	}

	for inYaml, outYaml := range assertions {
		v := PortBinding{}
		if err := yaml.Unmarshal([]byte(inYaml), &v); err != nil {
			t.Fatal(err)
		}
		data, err := yaml.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, outYaml, strings.TrimSpace(string(data)))
	}
}

func TestYamlCmd(t *testing.T) {
	assertions := map[string]string{
		"":   "[]",
		"[]": "[]",
		`["/bin/sh", "-c", "echo hello"]`: "- /bin/sh\n- -c\n- echo hello",
		`["du", "-h"]`:                    "- du\n- -h",
		"echo lopata":                     "- /bin/sh\n- -c\n- echo lopata",
		"- du":                            "- du",
	}

	for inYaml, outYaml := range assertions {
		v := &Cmd{}
		if err := yaml.Unmarshal([]byte(inYaml), v); err != nil {
			t.Fatal(err)
		}
		data, err := yaml.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, outYaml, strings.TrimSpace(string(data)))
	}
}

func TestYamlNet(t *testing.T) {
	assertions := map[string]string{
		"":                 "\"\"",
		"none":             "none",
		"host":             "host",
		"bridge":           "bridge",
		"container:statsd": "container:statsd",
	}

	for inYaml, outYaml := range assertions {
		v := &Net{}
		if err := yaml.Unmarshal([]byte(inYaml), v); err != nil {
			t.Fatal(err)
		}
		data, err := yaml.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, outYaml, strings.TrimSpace(string(data)))
	}
}

func TestYamlVolumesFrom(t *testing.T) {
	assertions := map[string]string{
		"":                 "[]",
		"- data":           "- data",
		"data":             "- data",
		"- .data":          "- data",
		`["data", "logs"]`: "- data\n- logs",
	}

	for inYaml, outYaml := range assertions {
		v := &VolumesFrom{}
		if err := yaml.Unmarshal([]byte(inYaml), v); err != nil {
			t.Fatal(err)
		}
		data, err := yaml.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, outYaml, strings.TrimSpace(string(data)))
	}
}

func TestYamlVolumes(t *testing.T) {
	assertions := map[string]string{
		"":                   "[]",
		"- /data":            "- /data",
		"/data":              "- /data",
		`["/data", "/logs"]`: "- /data\n- /logs",
	}

	for inYaml, outYaml := range assertions {
		v := &Volumes{}
		if err := yaml.Unmarshal([]byte(inYaml), v); err != nil {
			t.Fatal(err)
		}
		data, err := yaml.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, outYaml, strings.TrimSpace(string(data)))
	}
}

func TestYamlDns(t *testing.T) {
	assertions := map[string]string{
		"":                         "[]",
		"- 8.8.8.8":                "- 8.8.8.8",
		"192.168.1.1":              "- 192.168.1.1",
		`["8.8.8.8", "127.0.0.1"]`: "- 8.8.8.8\n- 127.0.0.1",
	}

	for inYaml, outYaml := range assertions {
		v := &Volumes{}
		if err := yaml.Unmarshal([]byte(inYaml), v); err != nil {
			t.Fatal(err)
		}
		data, err := yaml.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, outYaml, strings.TrimSpace(string(data)))
	}
}
