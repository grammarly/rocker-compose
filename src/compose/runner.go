package compose
import "fmt"

type Runner interface {
	Run([]Action) error
}

type dryRunner struct {
}

func NewDryRunner() Runner {
	return &dryRunner{}
}

func (r *dryRunner) Run(actions []Action) error {
	for _, a := range actions {
		fmt.Printf("[DRY] Running: %s\n", a.String())
	}
	return nil
}