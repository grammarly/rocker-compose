package compose

import (
	"os"
	"util"

	"github.com/fsouza/go-dockerclient"
)

type DockerClientConfig struct {
	Host      string
	Tlsverify bool
	Tlscacert string
	Tlscert   string
	Tlskey    string
}

func NewDockerClientConfig() *DockerClientConfig {
	certPath := util.StringOr(os.Getenv("DOCKER_CERT_PATH"), "~/.docker")
	return &DockerClientConfig{
		Host:      os.Getenv("DOCKER_HOST"),
		Tlsverify: os.Getenv("DOCKER_TLS_VERIFY") == "1" || os.Getenv("DOCKER_TLS_VERIFY") == "yes",
		Tlscacert: certPath + "/ca.pem",
		Tlscert:   certPath + "/cert.pem",
		Tlskey:    certPath + "/key.pem",
	}
}

func NewDockerClient() (*docker.Client, error) {
	config := NewDockerClientConfig()
	if config.Tlsverify {
		return docker.NewTLSClient(config.Host, config.Tlscert, config.Tlskey, config.Tlscacert)
	}
	return docker.NewClient(config.Host)
}
