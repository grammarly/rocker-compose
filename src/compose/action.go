package compose
import (
	"fmt"
	"bytes"
)

type Action interface {
	Execute(client Client) error
	String() string
}

type action struct {
	container *Container
}
type ensureContainer action
type createContainer action
type removeContainer action
type noAction        action

var NoAction = &noAction{}

type stepAction struct {
	actions    []Action
	concurrent bool
}

func NewStepAction(concurrent bool, actions ...Action) Action {
	return &stepAction{
		actions: actions,
		concurrent: concurrent,
	}
}

func NewEnsureContainerAction(c *Container) Action {
	return &ensureContainer{container: c}
}

func NewCreateContainerAction(c *Container) Action {
	return &createContainer{container: c}
}

func NewRemoveContainerAction(c *Container) Action {
	return &removeContainer{container: c}
}

func (s *stepAction) Execute(client Client) (err error) {
	for _, a := range s.actions {
		if s.concurrent {
			go a.Execute(client)
		}else {
			a.Execute(client)
		}
	}
	return nil
}

func (c *stepAction) String() string {
	var buffer bytes.Buffer
	for _, a := range c.actions {
		buffer.WriteString(fmt.Sprintf("Running in concurrency mode = %t: %s\n", c.concurrent, a.String()))
	}
	return buffer.String()
}

func (c *createContainer) Execute(client Client) (err error) {
	err = client.CreateContainer(c.container)
	return
}

func (c *createContainer) String() string {
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

