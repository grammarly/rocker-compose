package compose

import (
	"compose/config"
	"fmt"
)

type Diff interface {
	Diff(ns string, expected []*Container, actual []*Container) ([]Action, error)
}

// graph with container dependencies
type graph struct {
	dependencies map[*Container][]*dependency
}

// single dependency (external - means in other namespace)
type dependency struct {
	container *Container
	external  bool
	waitForIt bool
}

func NewDiff() Diff {
	return &graph{
		dependencies: make(map[*Container][]*dependency),
	}
}

func (g *graph) Diff(ns string, expected []*Container, actual []*Container) (res []Action, err error) {
	//filling dependency graph
	err = g.buildDependencyGraph(ns, expected, actual)
	if err != nil {
		res = []Action{}
		return
	}

	//check for cycles in configuration
	if g.hasCycles() {
		err = fmt.Errorf("Dependencies have cycles, check links and volumes-from")
		return
	}

	res = listContainersToRemove(ns, expected, actual)
	res = append(res, g.buildExecutionPlan(actual)...)
	return
}

func (g *graph) buildDependencyGraph(ns string, expected []*Container, actual []*Container) error {
	for _, c := range expected {
		g.dependencies[c] = []*dependency{}
		dependencies, err := resolveDependencies(ns, expected, actual, c)
		if err != nil {
			return err
		}
		g.dependencies[c] = append(g.dependencies[c], dependencies...)
	}
	return nil
}

func resolveDependencies(ns string, expected []*Container, actual []*Container, target *Container) (resolved []*dependency, err error) {
	resolved = []*dependency{}
	toResolve := map[config.ContainerName]*dependency{}

	//VolumesFrom
	for _, cn := range target.Config.VolumesFrom {
		if _, found := toResolve[cn]; !found {
			toResolve[cn] = &dependency{external: cn.Namespace != ns}
		}
	}

	//WaitFor
	for _, cn := range target.Config.WaitFor {
		if d, found := toResolve[cn]; !found {
			toResolve[cn] = &dependency{
				waitForIt: true,
				external:  cn.Namespace != ns,
			}
		} else {
			d.waitForIt = true
		}
	}

	//Links
	for _, cn := range target.Config.Links {
		if _, found := toResolve[cn]; !found {
			toResolve[cn] = &dependency{external: cn.Namespace != ns}
		}
	}

	for name, dep := range toResolve {
		// in case of the same namespace, we should find dependency
		// in given configuration
		var scope []*Container = expected

		if dep.external {
			scope = actual
		}

		if container := find(scope, &name); container != nil {
			dep.container = container
			resolved = append(resolved, dep)
			continue
		}

		err = fmt.Errorf("Cannot resolve dependency %s for %s", name, target)
		return
	}

	return
}

func listContainersToRemove(ns string, expected []*Container, actual []*Container) (res []Action) {
	for _, a := range actual {
		if a.Name.Namespace == ns {
			var found bool
			for _, e := range expected {
				found = found || e.IsSameKind(a)
			}
			if !found {
				res = append(res, NewRemoveContainerAction(a))
			}
		}
	}
	return
}

func (dg *graph) buildExecutionPlan(actual []*Container) (res []Action) {
	visited := map[*Container]bool{}
	restarted := map[*Container]struct{}{}

	// while number of visited deps less than number of
	// dependencies which should be visited - loop
	for len(visited) < len(dg.dependencies) {
		var step []Action = []Action{}

	nextDependency:
		for container, deps := range dg.dependencies {
			// if dependency is already visited - skip it
			if _, contains := visited[container]; contains {
				continue
			}

			var depActions []Action = []Action{}
			var restart bool

			// check transitive dependencies of current dependency
			for _, dependency := range deps {

				// for all external dependencies (in other namespace), ensure that it exists
				if dependency.waitForIt {
					depActions = append(depActions, NewWaitContainerAction(dependency.container))
				} else if dependency.external {
					depActions = append(depActions, NewEnsureContainerExistAction(dependency.container))

					// if any of dependencies not initialized yet, iterate to next one
				}

				if finalized, contains := visited[dependency.container]; !contains || !finalized {
					continue nextDependency
				}

				// if dependency should be restarted - we should restart current one
				_, contains := restarted[dependency.container]
				restart = restart || contains
			}

			// predefine flag / set false to prevent getting into the same operation
			visited[container] = false

			// comparing dependency with current state
			for _, actualContainer := range actual {
				if container.IsSameKind(actualContainer) {
					//in configuration was changed or restart forced by dependency - recreate container
					if !container.IsEqualTo(actualContainer) || restart {
						step = append(step, NewStepAction(false,
							NewStepAction(true, depActions...),
							NewRemoveContainerAction(actualContainer),
							NewRunContainerAction(container),
						))

						// mark container as recreated
						restarted[container] = struct{}{}
						continue nextDependency
					}

					// adding ensure action if applicable
					step = append(step, NewStepAction(true, depActions...))
					continue nextDependency
				}
			}

			// container is not exi
			step = append(step, NewStepAction(false,
				NewStepAction(true, depActions...),
				NewRunContainerAction(container),
			))
		}

		//finalize step
		for container, visit := range visited {
			if !visit {
				visited[container] = true
			}
		}

		// adding step to result (step actions will be run concurrently)
		res = append(res, NewStepAction(true, step...))
	}
	return
}

func find(containers []*Container, name *config.ContainerName) *Container {
	for _, c := range containers {
		if c.Name.IsEqualTo(name) {
			return c
		}
	}
	return nil
}

func (dg *graph) hasCycles() bool {
	for k, _ := range dg.dependencies {
		if dg.hasCycles0([]*Container{k}, k) {
			return true
		}
	}
	return false
}

func (dg *graph) hasCycles0(path []*Container, curr *Container) bool {
	for _, c := range path[:len(path)-1] {
		if c.IsSameKind(curr) {
			return true
		}
	}
	if deps := dg.dependencies[curr]; deps != nil {
		for _, d := range deps {
			if dg.hasCycles0(append(path, d.container), d.container) {
				return true
			}
		}
	}
	return false
}
