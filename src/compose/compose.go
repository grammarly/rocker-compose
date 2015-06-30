package compose

import (
	log "github.com/Sirupsen/logrus"
	"github.com/kr/pretty"
)

type ComposeConfig struct {
	Manifest  *Config
	DockerCfg *DockerClientConfig
	Timeout   int
	Global    bool
	Force     bool

}

func Run(config *ComposeConfig) {
	log.Debugf("Running configuration: \n%s", pretty.Sprintf("%# v", config))
	if cli, err := NewClient(&ClientCfg{
		NewDockerClientFromConfig(config.DockerCfg),
		Timeout: config.Timeout,
		Global: config.Global,
	}); err != nil {
		log.Errorf("Docker client initialization failed with error %s and config:\n%s", err,
			pretty.Sprintf("%# v", config.DockerCfg))
	}else {
		log.Debugf("Docker client initialization succeeded with config: \n%s",
			pretty.Sprintf("%# v", config.DockerCfg))
		run(cli)
	}
}

func run(client Client){
	//
}