package compose

import (
	"bytes"
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
type ensureContainer action
type runContainer action
type removeContainer action
type noAction action

var NoAction = &noAction{}

type stepAction struct {
	actions []Action
	async   bool
}

func NewStepAction(async bool, actions ...Action) Action {
	if len(actions) == 0 {
		return NoAction
	}

	return &stepAction{
		actions: actions,
		async:   async,
	}
}

func NewEnsureContainerAction(c *Container) Action {
	return &ensureContainer{container: c}
}

func NewRunContainerAction(c *Container) Action {
	return &runContainer{container: c}
}

func NewRemoveContainerAction(c *Container) Action {
	return &removeContainer{container: c}
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
	for _, a := range c.actions {
		buffer.WriteString(fmt.Sprintf("Running in concurrency mode = %t: %s\n", c.async, a.String()))
	}
	return buffer.String()
}

func (c *runContainer) Execute(client Client) (err error) {
	err = client.RunContainer(c.container)
	return
}

func (c *runContainer) String() string {
	return fmt.Sprintf("Creating container '%s'", c.container.Name.String())
}

func (r *removeContainer) Execute(client Client) (err error) {
	err = client.RemoveContainer(r.container)
	return
}

func (c *removeContainer) String() string {
	return fmt.Sprintf("Removing container '%s'", c.container.Name.String())
}

func (n *noAction) Execute(client Client) (err error) {
	return
}

func (c *noAction) String() string {
	return "NOOP"
}

func (c *ensureContainer) Execute(client Client) (err error) {
	return client.EnsureContainer(c.container)
}

func (c *ensureContainer) String() string {
	return fmt.Sprintf("Ensuring container '%s'", c.container.Name.String())
}
