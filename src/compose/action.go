package compose

import (
	"bytes"
	"compose/config"
	"fmt"
	"sync"
)

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

type notifyAction struct {
	container *Container
	notify    config.NotifyAction
}

var NoAction = &noAction{}

type stepAction struct {
	actions []Action
	async   bool
}

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

func NewWaitContainerAction(c *Container) Action {
	return &waitContainerAction{container: c}
}

func NewEnsureContainerExistAction(c *Container) Action {
	return &ensureContainerExist{container: c}
}

func NewEnsureContainerStateAction(c *Container) Action {
	return &ensureContainerState{container: c}
}

func NewRunContainerAction(c *Container) Action {
	return &runContainer{container: c}
}

func NewRemoveContainerAction(c *Container) Action {
	return &removeContainer{container: c}
}

func NewNotifyAction(c *Container, n config.NotifyAction) Action {
	return &notifyAction{container: c, notify: n}
}

func (s *stepAction) Execute(client Client) (err error) {
	if s.async {
		err = s.executeAsync(client)
	} else {
		err = s.executeSync(client)
	}
	return
}

func (s *stepAction) executeAsync(client Client) (err error) {
	var wg sync.WaitGroup
	len := len(s.actions)
	errors := make(chan error, len)
	wg.Add(len)
	for _, a := range s.actions {
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

func (s *stepAction) executeSync(client Client) (err error) {
	for _, a := range s.actions {
		if err = a.Execute(client); err != nil {
			return
		}
	}
	return
}

func (c *stepAction) String() string {
	var buffer bytes.Buffer
	buffer.WriteString(fmt.Sprintf("Running in concurrency mode = %t:\n", c.async))
	for _, a := range c.actions {
		buffer.WriteString(fmt.Sprintf("                        - %s\n", a))
	}
	return buffer.String()
}

func (c *runContainer) Execute(client Client) (err error) {
	err = client.RunContainer(c.container)
	return
}

func (c *runContainer) String() string {
	return fmt.Sprintf("Creating container '%s'", c.container.Name)
}

func (r *removeContainer) Execute(client Client) (err error) {
	err = client.RemoveContainer(r.container)
	return
}

func (c *removeContainer) String() string {
	return fmt.Sprintf("Removing container '%s'", c.container.Name)
}

func (r *waitContainerAction) Execute(client Client) (err error) {
	return client.WaitForContainer(r.container)
}

func (c *waitContainerAction) String() string {
	return fmt.Sprintf("Waiting for container '%s'", c.container.Name)
}

func (r *notifyAction) Execute(client Client) (err error) {
	return client.NotifyContainer(r.container, r.notify)
	return nil
}

func (c *notifyAction) String() string {
	return fmt.Sprintf("Notify '%s': %s", c.container.Name, c.notify)
}

func (n *noAction) Execute(client Client) (err error) {
	return
}

func (c *noAction) String() string {
	return "noop"
}

func (c *ensureContainerExist) Execute(client Client) (err error) {
	return client.EnsureContainerExist(c.container)
}

func (c *ensureContainerExist) String() string {
	return fmt.Sprintf("Ensuring container '%s'", c.container.Name)
}

func (c *ensureContainerState) Execute(client Client) (err error) {
	return client.EnsureContainerState(c.container)
}

func (c *ensureContainerState) String() string {
	return fmt.Sprintf("Ensuring container state '%s'", c.container.Name)
}

// TODO: maybe find a better place for this function
func WalkActions(actions []Action, fn func(action Action)) {
	for _, a := range actions {
		if step, ok := a.(*stepAction); ok {
			WalkActions(step.actions, fn)
		} else {
			fn(a)
		}
	}
}
