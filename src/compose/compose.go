package compose

import (
	"fmt"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/kr/pretty"
)

type ComposeConfig struct {
	Manifest  *Config
	DockerCfg *DockerClientConfig
	Global    bool
	Force     bool
	DryRun    bool
	Attach    bool
	Wait      time.Duration
}

type Compose struct {
	Manifest *Config
	DryRun   bool
	Attach   bool
	Wait     time.Duration

	client             Client
	chErrors           chan error
	attachedContainers map[string]struct{}
}

func New(config *ComposeConfig) (*Compose, error) {
	compose := &Compose{
		Manifest: config.Manifest,
		DryRun:   config.DryRun,
		Attach:   config.Attach,
		Wait:     config.Wait,
	}

	docker, err := NewDockerClientFromConfig(config.DockerCfg)
	if err != nil {
		return nil, fmt.Errorf("Docker client initialization failed with error '%s' and config:\n%s", err,
			pretty.Sprintf("%# v", config.DockerCfg))
	}

	log.Debugf("Docker config: \n%s", pretty.Sprintf("%# v", config.DockerCfg))

	cliConf := &ClientCfg{
		Docker: docker,
		Global: config.Global,
		Attach: config.Attach,
		Wait:   config.Wait,
	}

	cli, err := NewClient(cliConf)
	if err != nil {
		return nil, fmt.Errorf("Compose client initialization failed with error '%s' and config:\n%s", err,
			pretty.Sprintf("%# v", cliConf))
	}

	compose.client = cli

	return compose, nil
}

func (compose *Compose) Run() error {
	actual, err := compose.client.GetContainers()
	if err != nil {
		return fmt.Errorf("GetContainers failed with error '%s'", err)
	}

	expected := compose.Manifest.GetContainers()

	executionPlan, err := NewDiff().Diff(compose.Manifest.Namespace, expected, actual)
	if err != nil {
		return fmt.Errorf("Diff of configuration failed due to error '%s'", err)
	}

	var runner Runner
	if compose.DryRun {
		runner = NewDryRunner()
	} else {
		runner = NewDockerClientRunner(compose.client)
	}

	if err := runner.Run(executionPlan); err != nil {
		return fmt.Errorf("Execution failed with, error: %s", err)
	}

	if compose.Attach {
		if err := compose.client.AttachToContainers(expected); err != nil {
			return fmt.Errorf("Cannot attach to containers, error: %s", err)
		}
	}

	return nil
}
