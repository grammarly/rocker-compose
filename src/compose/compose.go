/*-
 * Copyright 2015 Grammarly, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Package compose is the main rocker-compose facade. It provides functions
// to execute various rocker-compose tasks based on the given manifest.
//
// Rocker-compose is a docker composition tool with idempotency features for deploying applications that consist of multiple containers.
package compose

import (
	"fmt"
	"github.com/snkozlov/rocker-compose/src/compose/ansible"
	"github.com/snkozlov/rocker-compose/src/compose/config"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/fsouza/go-dockerclient"
	"github.com/grammarly/rocker/src/template"
	"github.com/kr/pretty"
)

// Config is a configuration object which is passed to compose.New()
// for creating the new Compose instance.
type Config struct {
	Manifest   *config.Config
	Docker     *docker.Client
	Force      bool
	DryRun     bool
	Attach     bool
	Pull       bool
	Remove     bool
	Recover    bool
	Wait       time.Duration
	Auth       *docker.AuthConfigurations
	KeepImages int
}

// Compose is the main object that executes actions and holds runtime information.
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

// New makes a new Compose object
func New(config *Config) (*Compose, error) {
	compose := &Compose{
		Manifest: config.Manifest,
		DryRun:   config.DryRun,
		Attach:   config.Attach,
		Pull:     config.Pull,
		Wait:     config.Wait,
		Remove:   config.Remove,
	}

	cliConf := &DockerClient{
		Docker:     config.Docker,
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

// RunAction implements 'rocker-compose run'
func (compose *Compose) RunAction() error {
	// get the actual list of existing containers from docker client
	actual, err := compose.client.GetContainers(compose.Manifest.HasExternalRefs())
	if err != nil {
		return fmt.Errorf("GetContainers failed with error, error: %s", err)
	}

	expected := []*Container{}

	// if --remove was specified, pretend we expect to have an empty list of containers
	if !compose.Remove {
		expected = GetContainersFromConfig(compose.Manifest)
	}

	// if --pull is specified PullAll, otherwise Fetch required
	if compose.Pull {
		if err := compose.client.PullAll(expected, compose.Manifest.Vars); err != nil {
			return err
		}
	} else if err := compose.client.FetchImages(expected, compose.Manifest.Vars); err != nil {
		return fmt.Errorf("Failed to fetch images of given containers, error: %s", err)
	}

	// Assign IDs of existing containers
	for _, actualC := range actual {
		for _, expectedC := range expected {
			if expectedC.IsSameKind(actualC) {
				expectedC.ID = actualC.ID
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

	// if --attach was specified
	if compose.Attach {
		log.Debugf("Attaching to containers...")
		if err := compose.client.AttachToContainers(expected); err != nil {
			return fmt.Errorf("Cannot attach to containers, error: %s", err)
		}
	}

	return nil
}

// RecoverAction implements 'rocker-compose recover'
//
// TODO: It duplicates the code of RunAction a bit. Also, do we need this function at all?
// 			 Docker starts containers of "restart=always" automatically after daemon restart.
func (compose *Compose) RecoverAction() error {
	actual, err := compose.client.GetContainers(false)
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

// PullAction implements 'rocker-compose pull'
func (compose *Compose) PullAction() error {
	containers := GetContainersFromConfig(compose.Manifest)
	if err := compose.client.PullAll(containers, compose.Manifest.Vars); err != nil {
		return fmt.Errorf("Failed to pull all images, error: %s", err)
	}

	return nil
}

// CleanAction implements 'rocker-compose clean'
func (compose *Compose) CleanAction() error {
	if err := compose.client.Clean(compose.Manifest); err != nil {
		return fmt.Errorf("Failed to clean old images, error: %s", err)
	}

	return nil
}

// PinAction implements 'rocker-compose pin'
func (compose *Compose) PinAction(local, hub bool) (template.Vars, error) {
	containers := GetContainersFromConfig(compose.Manifest)
	if err := compose.client.Pin(local, hub, compose.Manifest.Vars, containers); err != nil {
		return nil, fmt.Errorf("Failed to pin, error: %s", err)
	}

	// Populate versions to the variables
	vars := compose.Manifest.Vars
	for _, c := range containers {
		vars[fmt.Sprintf("v_container_%s", c.Name.Name)] = c.Image.GetTag()
	}

	return vars, nil
}

// WritePlan saves various rocker-compose change information to the ansible.Response object
// TODO: should compose know about ansible.Response at all?
//       maybe it should give some data struct back to main?
func (compose *Compose) WritePlan(resp *ansible.Response) *ansible.Response {
	resp.Removed = []ansible.ResponseContainer{}
	resp.Created = []ansible.ResponseContainer{}
	resp.Pulled = []string{}
	resp.Cleaned = []string{}

	WalkActions(compose.executionPlan, func(action Action) {
		if a, ok := action.(*removeContainer); ok {
			resp.Removed = append(resp.Removed, ansible.ResponseContainer{
				ID:   a.container.ID,
				Name: a.container.Name.String(),
			})
		}
		if a, ok := action.(*runContainer); ok {
			resp.Created = append(resp.Created, ansible.ResponseContainer{
				ID:   a.container.ID,
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
