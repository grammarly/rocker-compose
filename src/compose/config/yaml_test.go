package config

import (
	"fmt"
	"strings"
	"testing"

	"github.com/go-yaml/yaml"
	"github.com/stretchr/testify/assert"
)

type yamlTestCases struct {
	assertions map[string]string
}

func (a *yamlTestCases) run(t *testing.T) error {
	for inYaml, outYaml := range a.assertions {
		v := &Container{}
		if err := yaml.Unmarshal([]byte(inYaml), v); err != nil {
			return fmt.Errorf("Failed unmarshal for test %q, error: %s", inYaml, err)
		}
		data, err := yaml.Marshal(v)
		if err != nil {
			return fmt.Errorf("Failed marshal for test %q, error: %s", inYaml, err)
		}
		assert.Equal(t, outYaml, strings.TrimSpace(string(data)))
	}
	return nil
}

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

func TestYamlMemory(t *testing.T) {
	test := &yamlTestCases{
		map[string]string{
			"memory: 0":    "memory: 0",
			"memory: 300":  "memory: 300",
			"memory: 300b": "memory: 300",
			"memory: 1k":   "memory: 1024",
			"memory: 1m":   "memory: 1048576",
			"memory: 1g":   "memory: 1073741824",
		},
	}
	if err := test.run(t); err != nil {
		t.Fatal(err)
	}
}

func TestYamlRestartPolicy(t *testing.T) {
	test := &yamlTestCases{
		map[string]string{
			"restart: no":           "restart: \"no\"",
			"restart: always":       "restart: always",
			"restart: on-failure,5": "restart: on-failure,5",
			"restart: on-failure":   "restart: on-failure,0",
		},
	}
	if err := test.run(t); err != nil {
		t.Fatal(err)
	}
}

func TestYamlCmd(t *testing.T) {
	test := &yamlTestCases{
		map[string]string{
			"cmd: []": "{}",
			`cmd: ["/bin/sh", "-c", "echo hello"]`: "cmd:\n- /bin/sh\n- -c\n- echo hello",
			`cmd: ["du", "-h"]`:                    "cmd:\n- du\n- -h",
			"cmd: echo lopata":                     "cmd:\n- /bin/sh\n- -c\n- echo lopata",
			"cmd:\n- du":                           "cmd:\n- du",
		},
	}
	if err := test.run(t); err != nil {
		t.Fatal(err)
	}
}

func TestYamlNet(t *testing.T) {
	test := &yamlTestCases{
		map[string]string{
			"net: none":             "net: none",
			"net: host":             "net: host",
			"net: bridge":           "net: bridge",
			"net: container:statsd": "net: container:statsd",
		},
	}
	if err := test.run(t); err != nil {
		t.Fatal(err)
	}
}

func TestYamlVolumesFrom(t *testing.T) {
	test := &yamlTestCases{
		map[string]string{
			"volumes_from:\n- data":          "volumes_from:\n- data",
			"volumes_from: data":             "volumes_from:\n- data",
			"volumes_from:\n- .data":         "volumes_from:\n- data",
			`volumes_from: ["data", "logs"]`: "volumes_from:\n- data\n- logs",
		},
	}
	if err := test.run(t); err != nil {
		t.Fatal(err)
	}
}

func TestYamlDns(t *testing.T) {
	test := &yamlTestCases{
		map[string]string{
			"dns:\n- 8.8.8.8":               "dns:\n- 8.8.8.8",
			"dns: 192.168.1.1":              "dns:\n- 192.168.1.1",
			`dns: ["8.8.8.8", "127.0.0.1"]`: "dns:\n- 8.8.8.8\n- 127.0.0.1",
		},
	}
	if err := test.run(t); err != nil {
		t.Fatal(err)
	}
}

func TestYamlHosts(t *testing.T) {
	test := &yamlTestCases{
		map[string]string{
			"add_host:\n- dns:8.8.8.8":                         "add_host:\n- dns:8.8.8.8",
			"add_host: gateway:192.168.1.1":                    "add_host:\n- gateway:192.168.1.1",
			`add_host: ["dns:8.8.8.8", "localhost:127.0.0.1"]`: "add_host:\n- dns:8.8.8.8\n- localhost:127.0.0.1",
		},
	}
	if err := test.run(t); err != nil {
		t.Fatal(err)
	}
}

func TestYamlVolumes(t *testing.T) {
	test := &yamlTestCases{
		map[string]string{
			"volumes:\n- /data":           "volumes:\n- /data",
			"volumes: /mnt":               "volumes:\n- /mnt",
			`volumes: ["/data", "/logs"]`: "volumes:\n- /data\n- /logs",
		},
	}
	if err := test.run(t); err != nil {
		t.Fatal(err)
	}
}

func TestYamlEntrypoint(t *testing.T) {
	test := &yamlTestCases{
		map[string]string{
			"entrypoint:":                          "{}",
			"entrypoint:\n- 8.8.8.8":               "entrypoint:\n- 8.8.8.8",
			"entrypoint: 192.168.1.1":              "entrypoint:\n- 192.168.1.1",
			`entrypoint: ["8.8.8.8", "127.0.0.1"]`: "entrypoint:\n- 8.8.8.8\n- 127.0.0.1",
		},
	}
	if err := test.run(t); err != nil {
		t.Fatal(err)
	}
}

func TestYamlPorts(t *testing.T) {
	test := &yamlTestCases{
		map[string]string{
			"ports:\n- 8080":          "ports:\n- 8080/tcp",
			"ports: 8090":             "ports:\n- 8090/tcp",
			`ports: ["8080", "8090"]`: "ports:\n- 8080/tcp\n- 8090/tcp",
		},
	}
	if err := test.run(t); err != nil {
		t.Fatal(err)
	}
}

func TestYamlLinks(t *testing.T) {
	test := &yamlTestCases{
		map[string]string{
			"links:\n- statsd":        "links:\n- statsd:statsd",
			"links: mysql:db":         "links:\n- mysql:db",
			`links: ["statsd", "db"]`: "links:\n- statsd:statsd\n- db:db",
		},
	}
	if err := test.run(t); err != nil {
		t.Fatal(err)
	}
}

func TestYamlWaitFor(t *testing.T) {
	test := &yamlTestCases{
		map[string]string{
			"wait_for:\n- data":          "wait_for:\n- data",
			"wait_for: data":             "wait_for:\n- data",
			"wait_for:\n- .data":         "wait_for:\n- data",
			`wait_for: ["data", "logs"]`: "wait_for:\n- data\n- logs",
		},
	}
	if err := test.run(t); err != nil {
		t.Fatal(err)
	}
}

func TestYamlEnv(t *testing.T) {
	test := &yamlTestCases{
		map[string]string{
			"env:": "{}",
			"env:\n  REDIS_HOST: redis": "env:\n  REDIS_HOST: redis",
			"env: DB_HOST=db":           "env:\n  DB_HOST: db",
			"env: NO_METRICS":           "env:\n  NO_METRICS: \"true\"",
		},
	}
	if err := test.run(t); err != nil {
		t.Fatal(err)
	}
}

func TestYamlLabels(t *testing.T) {
	test := &yamlTestCases{
		map[string]string{
			"labels:":                      "{}",
			"labels:\n  REDIS_HOST: redis": "labels:\n  REDIS_HOST: redis",
			"labels: DB_HOST=db":           "labels:\n  DB_HOST: db",
			"labels: NO_METRICS":           "labels:\n  NO_METRICS: \"true\"",
		},
	}
	if err := test.run(t); err != nil {
		t.Fatal(err)
	}
}
