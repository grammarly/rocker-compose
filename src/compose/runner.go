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

package compose

import (
	log "github.com/Sirupsen/logrus"
)

// Runner interface describes a runnable facade which executes given list of actions
type Runner interface {
	Run([]Action) error
}

type dryRunner struct{}

type dockerClientRunner struct {
	client Client
}

// NewDryRunner makes a runner that does not actually execute actions, but prints them
func NewDryRunner() Runner {
	return &dryRunner{}
}

// NewDockerClientRunner makes a runner that uses a DockerClient for executing actions
func NewDockerClientRunner(client Client) Runner {
	return &dockerClientRunner{
		client: client,
	}
}

// Run executes all actions
func (r *dockerClientRunner) Run(actions []Action) (err error) {
	for _, a := range actions {
		if err = a.Execute(r.client); err != nil {
			return
		}
	}
	return
}

// Run prints all actions that were about to execute
func (r *dryRunner) Run(actions []Action) error {
	for _, a := range actions {
		log.Infof("[DRY] Running: %s", a)
	}
	return nil
}
