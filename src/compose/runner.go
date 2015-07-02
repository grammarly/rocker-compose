package compose

import (
	log "github.com/Sirupsen/logrus"
)

type Runner interface {
	Run([]Action) error
}

type dryRunner struct{}

type dockerClientRunner struct {
	client Client
}

func NewDryRunner() Runner {
	return &dryRunner{}
}

func NewDockerClientRunner(client Client) Runner {
	return &dockerClientRunner{
		client: client,
	}
}

func (r *dockerClientRunner) Run(actions []Action) (err error) {
	for _, a := range actions {
		if err = a.Execute(r.client); err != nil {
			return
		}
	}
	return
}

func (r *dryRunner) Run(actions []Action) error {
	for _, a := range actions {
		log.Infof("[DRY] Running: %s", a)
	}
	return nil
}
