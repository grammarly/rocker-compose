package compose

import (
	"compose/ansible"
	"compose/config"
	"fmt"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/fsouza/go-dockerclient"
	"github.com/kr/pretty"
)

type ComposeConfig struct {
	Manifest   *config.Config
	Docker     *docker.Client
	Global     bool
	Force      bool
	DryRun     bool
	Attach     bool
	Pull       bool
	Remove     bool
	Recover    bool
	Wait       time.Duration
	Auth       *AuthConfig
	KeepImages int
}

type Compose struct {
	Manifest *config.Config
	DryRun   bool
	Attach   bool
	Pull     bool
	Remove   bool
	Wait     time.Duration

	client             Client
	chErrors           chan error
	attachedContainers map[string]struct{}
	executionPlan      []Action
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

	cliConf := &ClientCfg{
		Docker:     config.Docker,
		Global:     config.Global,
		Attach:     config.Attach,
		Wait:       config.Wait,
		Auth:       config.Auth,
		KeepImages: config.KeepImages,
		Recover:    config.Recover,
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
		expected = GetContainersFromConfig(compose.Manifest)
	}

	if err := compose.client.FetchImages(expected); err != nil {
		return fmt.Errorf("Failed to fetch images of given containers, error: %s", err)
	}

	// Assign IDs of existing containers
	for _, actualC := range actual {
		for _, expectedC := range expected {
			if expectedC.IsSameKind(actualC) {
				expectedC.Id = actualC.Id
			}
		}
	}

	executionPlan, err := NewDiff(compose.Manifest.Namespace).Diff(expected, actual)
	if err != nil {
		return fmt.Errorf("Diff of configuration failed, error: %s", err)
	}
	compose.executionPlan = executionPlan

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
		log.Infof("OK, containers are running: %s", strings.Join(strContainers, ", "))
	} else {
		log.Infof("Nothing is running")
	}

	if compose.Attach {
		log.Debugf("Attaching to containers...")
		if err := compose.client.AttachToContainers(expected); err != nil {
			return fmt.Errorf("Cannot attach to containers, error: %s", err)
		}
	}

	return nil
}

func (compose *Compose) RecoverAction() error {
	actual, err := compose.client.GetContainers()
	if err != nil {
		return fmt.Errorf("GetContainers failed with error, error: %s", err)
	}

	// collect expected containers list based on actual state
	// but use expected state
	expected := []*Container{}
	for _, c := range actual {
		expectedC := *c // actually copy the struct
		expectedC.State = &ContainerState{
			Running: c.Config.State.Bool(),
		}
		expected = append(expected, &expectedC)
	}

	executionPlan, err := NewDiff("").Diff(expected, actual)
	if err != nil {
		return fmt.Errorf("Diff of configuration failed, error: %s", err)
	}
	compose.executionPlan = executionPlan

	var runner Runner
	if compose.DryRun {
		runner = NewDryRunner()
	} else {
		runner = NewDockerClientRunner(compose.client)
	}

	// pretty.Println(executionPlan)

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
		log.Infof("OK, containers are running: %s", strings.Join(strContainers, ", "))
	} else {
		log.Infof("Nothing is running")
	}

	return nil
}

func (compose *Compose) PullAction() error {
	if err := compose.client.PullAll(compose.Manifest); err != nil {
		return fmt.Errorf("Failed to pull all images, error: %s", err)
	}

	return nil
}

func (compose *Compose) CleanAction() error {
	if err := compose.client.Clean(compose.Manifest); err != nil {
		return fmt.Errorf("Failed to clean old images, error: %s", err)
	}

	return nil
}

// TODO: should compose know about ansible.Response at all?
// maybe it should give some data struct back to main?
func (compose *Compose) WritePlan(resp *ansible.Response) *ansible.Response {
	resp.Removed = []ansible.ResponseContainer{}
	resp.Created = []ansible.ResponseContainer{}
	resp.Pulled = []string{}
	resp.Cleaned = []string{}

	WalkActions(compose.executionPlan, func(action Action) {
		if a, ok := action.(*removeContainer); ok {
			resp.Removed = append(resp.Removed, ansible.ResponseContainer{
				Id:   a.container.Id,
				Name: a.container.Name.String(),
			})
		}
		if a, ok := action.(*runContainer); ok {
			resp.Created = append(resp.Created, ansible.ResponseContainer{
				Id:   a.container.Id,
				Name: a.container.Name.String(),
			})
		}
	})

	// TODO: images are pulled but may not be changed
	for _, imageName := range compose.client.GetPulledImages() {
		resp.Pulled = append(resp.Pulled, imageName.String())
	}

	for _, imageName := range compose.client.GetRemovedImages() {
		resp.Cleaned = append(resp.Cleaned, imageName.String())
	}

	resp.Changed = len(resp.Removed)+len(resp.Created)+len(resp.Pulled) > 0
	return resp
}
