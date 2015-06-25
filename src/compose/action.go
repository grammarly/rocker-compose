package compose

type Action interface {
	Execute() error
}

type action struct {
	container *Container
	client    *Client
}

type createContainer action
type removeContainer action

func CreateContainer(client *Client, c *Container) Action {
	return &createContainer{
		container: c,
		client:    client,
	}
}

func RemoveContainer(client *Client, c *Container) Action {
	return &removeContainer{
		container: c,
		client:    client,
	}
}

func (c *createContainer) Execute() (err error) {
	err = c.client.CreateContainer(c.container)
	return
}

func (r *removeContainer) Execute() (err error) {
	err = r.client.RemoveContainer(r.container)
	return
}

