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
	"bytes"
	"fmt"
	"sync"
)

// Action interface describes action that can be done by rocker-compose docker client
type Action interface {
	Execute(client Client) error
	String() string
}

type action struct {
	container *Container
}
type ensureContainerExist action
type ensureContainerState action
type runContainer action
type removeContainer action
type noAction action
type waitContainerAction action

// NoAction is an empty action which does nothing
var NoAction = &noAction{}

type stepAction struct {
	actions []Action
	async   bool
}

// NewStepAction makes a "step" wrapper which holds the list of actions that may run in parallel.
// Multiple steps can only run one by one. Steps can be nested.
func NewStepAction(async bool, actions ...Action) Action {
	len := len(actions)
	if len == 0 {
		return NoAction
	}

	if len == 1 {
		return actions[0]
	}

	//filter NoAction elements
	acts := []Action{}
	for _, a := range actions {
		if a != NoAction {
			acts = append(acts, a)
		}
	}

	return &stepAction{
		actions: acts,
		async:   async,
	}
}

// NewWaitContainerAction makes action that waits for container
func NewWaitContainerAction(c *Container) Action {
	return &waitContainerAction{container: c}
}

// NewEnsureContainerExistAction makes action that ensures that container exists
func NewEnsureContainerExistAction(c *Container) Action {
	return &ensureContainerExist{container: c}
}

// NewEnsureContainerStateAction makes action that ensures that container
// state is a desired one
func NewEnsureContainerStateAction(c *Container) Action {
	return &ensureContainerState{container: c}
}

// NewRunContainerAction makes action that runs a container
func NewRunContainerAction(c *Container) Action {
	return &runContainer{container: c}
}

// NewRemoveContainerAction makes action that removes a container
func NewRemoveContainerAction(c *Container) Action {
	return &removeContainer{container: c}
}

// Execute runs the step
func (a *stepAction) Execute(client Client) (err error) {
	if a.async {
		err = a.executeAsync(client)
	} else {
		err = a.executeSync(client)
	}
	return
}

func (a *stepAction) executeAsync(client Client) (err error) {
	var wg sync.WaitGroup
	len := len(a.actions)
	errors := make(chan error, len)
	wg.Add(len)
	for _, a := range a.actions {
		go func(action Action) {
			defer wg.Done()
			if err := action.Execute(client); err != nil {
				errors <- err
			}
		}(a)
	}
	wg.Wait()
	select {
	case err = <-errors:
	default:
	}
	return
}

func (a *stepAction) executeSync(client Client) (err error) {
	for _, a := range a.actions {
		if err = a.Execute(client); err != nil {
			return
		}
	}
	return
}

// String returns the printable string representation of the step.
func (a *stepAction) String() string {
	var buffer bytes.Buffer
	buffer.WriteString(fmt.Sprintf("Running in concurrency mode = %t:\n", a.async))
	for _, a := range a.actions {
		buffer.WriteString(fmt.Sprintf("                        - %s\n", a))
	}
	return buffer.String()
}

// Execute runs a container
func (a *runContainer) Execute(client Client) (err error) {
	err = client.RunContainer(a.container)
	return
}

// String returns the printable string representation of the runContainer action.
func (a *runContainer) String() string {
	return fmt.Sprintf("Creating container '%s'", a.container.Name)
}

// Execute removes a container
func (a *removeContainer) Execute(client Client) (err error) {
	err = client.RemoveContainer(a.container)
	return
}

// String returns the printable string representation of the removeContainer action.
func (a *removeContainer) String() string {
	return fmt.Sprintf("Removing container '%s'", a.container.Name)
}

// Execute waits for a container
func (a *waitContainerAction) Execute(client Client) (err error) {
	return client.WaitForContainer(a.container)
}

// String returns the printable string representation of the waitContainer action.
func (a *waitContainerAction) String() string {
	return fmt.Sprintf("Waiting for container '%s'", a.container.Name)
}

// Execute does nothing
func (a *noAction) Execute(client Client) (err error) {
	return
}

// String returns "noop"
func (a *noAction) String() string {
	return "noop"
}

// Execute ensures container exists
func (c *ensureContainerExist) Execute(client Client) (err error) {
	return client.EnsureContainerExist(c.container)
}

// String returns the printable string representation of the ensureContainerExist action.
func (c *ensureContainerExist) String() string {
	return fmt.Sprintf("Ensuring container '%s'", c.container.Name)
}

// Execute ensures container state is what we want in a spec
func (c *ensureContainerState) Execute(client Client) (err error) {
	return client.EnsureContainerState(c.container)
}

// String returns the printable string representation of the ensureContainerState action.
func (c *ensureContainerState) String() string {
	return fmt.Sprintf("Ensuring container state '%s'", c.container.Name)
}

// WalkActions recursively though all action and applies given function to every action.
func WalkActions(actions []Action, fn func(action Action)) {
	for _, a := range actions {
		if step, ok := a.(*stepAction); ok {
			WalkActions(step.actions, fn)
		} else {
			fn(a)
		}
	}
}
