package compose

import (
	"fmt"
	"strings"
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
	Pull      bool
	Remove    bool
	Wait      time.Duration
	Auth      *AuthConfig
}

type Compose struct {
	Manifest *Config
	DryRun   bool
	Attach   bool
	Pull     bool
	Remove   bool
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
		Pull:     config.Pull,
		Wait:     config.Wait,
		Remove:   config.Remove,
	}

	docker, err := NewDockerClientFromConfig(config.DockerCfg)
	if err != nil {
		return nil, fmt.Errorf("Docker client initialization failed with error '%s' and config:\n%s", err,
			pretty.Sprintf("%# v", config.DockerCfg))
	}

	log.Debugf("Docker config: %# v", pretty.Formatter(config.DockerCfg))

	cliConf := &ClientCfg{
		Docker: docker,
		Global: config.Global,
		Attach: config.Attach,
		Wait:   config.Wait,
		Auth:   config.Auth,
	}

	cli, err := NewClient(cliConf)
	if err != nil {
		return nil, fmt.Errorf("Compose client initialization failed with error '%s' and config:\n%s", err,
			pretty.Sprintf("%# v", cliConf))
	}

	compose.client = cli

	return compose, nil
}

func (compose *Compose) RunAction() error {
	if compose.Pull {
		if err := compose.PullAction(); err != nil {
			return err
		}
	}
	actual, err := compose.client.GetContainers()
	if err != nil {
		return fmt.Errorf("GetContainers failed with error, error: %s", err)
	}

	expected := []*Container{}

	if !compose.Remove {
		expected = compose.Manifest.GetContainers()
	}

	if err := compose.client.FetchImages(expected); err != nil {
		return fmt.Errorf("Failed to fetch images of given containers, error: %s", err)
	}

	executionPlan, err := NewDiff().Diff(compose.Manifest.Namespace, expected, actual)
	if err != nil {
		return fmt.Errorf("Diff of configuration failed, error: %s", err)
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

	strContainers := []string{}
	for _, container := range expected {
		// TODO: map ids for already existing containers
		// strContainers = append(strContainers, fmt.Sprintf("%s (id: %s)", container.Name, util.TruncateID(container.Id)))
		strContainers = append(strContainers, container.Name.String())
	}

	if len(strContainers) > 0 {
		log.Infof("Running containers: %s", strings.Join(strContainers, ", "))
	} else {
		log.Infof("Nothing is running")
	}

	if compose.Attach {
		if err := compose.client.AttachToContainers(expected); err != nil {
			return fmt.Errorf("Cannot attach to containers, error: %s", err)
		}
	}

	return nil
}

func (compose *Compose) PullAction() error {
	if err := compose.client.PullAll(compose.Manifest); err != nil {
		return fmt.Errorf("Failed to pull all images, error: %s", err)
	}

	return nil
}
