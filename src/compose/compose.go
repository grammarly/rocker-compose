package compose

import (
	log "github.com/Sirupsen/logrus"
	"github.com/kr/pretty"
)

type ComposeConfig struct {
	Manifest  *Config
	DockerCfg *DockerClientConfig
	Global    bool
	Force     bool
	DryRun    bool
}

func Run(config *ComposeConfig) {
	log.Debugf("Running configuration: \n%s", pretty.Sprintf("%# v", config))

	docker, err := NewDockerClientFromConfig(config.DockerCfg)
	if err != nil {
		log.Errorf("Docker client initialization failed with error '%s' and config:\n%s", err,
			pretty.Sprintf("%# v", config.DockerCfg))
		return
	}

	log.Debugf("Docker config: \n%s", pretty.Sprintf("%# v", config.DockerCfg))

	cliConf := ClientCfg{
		Docker: docker,
		Global: config.Global,
	}

	cli, err := NewClient(&cliConf)
	if err != nil {
		log.Errorf("Compose client initialization failed with error '%s' and config:\n%s", err,
			pretty.Sprintf("%# v", cliConf))
	}

	// log.Debugf("Composer client initialization succeeded with config: \n%s", pretty.Sprintf("%# v", cliConf))

	run(cli, config)
}

func run(client Client, config *ComposeConfig) {
	actual, err := client.GetContainers()
	if err != nil {
		log.Errorf("GetContainers failed with error '%s'", err)
		return
	}

	expected := config.Manifest.GetContainers()

	diff := NewDiff()
	executionPlan, err := diff.Diff(config.Manifest.Namespace,
		expected,
		actual)

	if err != nil {
		log.Errorf("Diff of configuration failed due to error '%s'", err)
		return
	}

	var runner Runner
	if config.DryRun {
		runner = NewDryRunner()
	} else {
		runner = NewDockerClientRunner(client)
	}

	if err := runner.Run(executionPlan); err != nil {
		log.Errorf("Execution failed with '%s'", err)
	}
}
