package compose
import "fmt"

type Runner interface {
	Run([]Action) error
}

type dryRunner struct {}

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
			fmt.Printf("[RUNNER] Running: %s\n", a.String())
			return
		}
	}
	return
}

func (r *dryRunner) Run(actions []Action) error {
	for _, a := range actions {
		fmt.Printf("[DRY] Running: %s\n", a.String())
	}
	return nil
}