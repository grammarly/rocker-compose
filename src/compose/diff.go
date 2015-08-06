package compose

import (
	"compose/config"
	"fmt"
	"strings"
)

type Diff interface {
	Diff(expected []*Container, actual []*Container) ([]Action, error)
}

// graph with container dependencies
type graph struct {
	ns            string
	dependencies  map[*Container][]*dependency
	notifications map[*Container][]*dependency
}

// single dependency (external - means not in our namespace)
type dependency struct {
	container *Container
	external  bool
	waitForIt bool
	notify    config.NotifyAction
}

func NewDiff(ns string) Diff {
	return &graph{
		ns:            ns,
		dependencies:  make(map[*Container][]*dependency),
		notifications: make(map[*Container][]*dependency),
	}
}

func (g *graph) Diff(expected []*Container, actual []*Container) (res []Action, err error) {
	//filling dependency graph
	err = g.buildDependencyGraph(expected, actual)
	if err != nil {
		res = []Action{}
		return
	}

	//check for cycles in configuration
	if g.hasCycles() {
		// TODO: more descriptive error message
		err = fmt.Errorf("Dependencies have cycles, check links, volumes-from and notify")
		return
	}

	res = listContainersToRemove(g.ns, expected, actual)
	res = append(res, g.buildExecutionPlan(actual)...)
	return
}

func (g *graph) buildDependencyGraph(expected []*Container, actual []*Container) error {
	for _, c := range expected {
		if err := g.resolveDependencies(expected, actual, c); err != nil {
			return err
		}
	}
	fmt.Printf("deps: %+v\n", g.dependencies)
	// pretty.Println(g.dependencies[expected[0]])
	// pretty.Println(g.dependencies[expected[1]])
	return nil
}

func (g *graph) resolveDependencies(expected []*Container, actual []*Container, target *Container) (err error) {
	toResolve := map[config.ContainerName]*dependency{}
	toNotify := map[config.ContainerName]*dependency{}
	toNotify2 := map[config.ContainerName]*dependency{}

	if g.dependencies[target] == nil {
		g.dependencies[target] = []*dependency{}
	}

	//VolumesFrom
	for _, cn := range target.Config.VolumesFrom {
		if _, found := toResolve[cn]; !found {
			toResolve[cn] = &dependency{external: cn.Namespace != g.ns}
		}
	}

	//WaitFor
	for _, cn := range target.Config.WaitFor {
		if d, found := toResolve[cn]; !found {
			toResolve[cn] = &dependency{
				waitForIt: true,
				external:  cn.Namespace != g.ns,
			}
		} else {
			d.waitForIt = true
		}
	}

	//Links
	for _, cn := range target.Config.Links {
		cn := cn.ContainerName()
		if _, found := toResolve[cn]; !found {
			toResolve[cn] = &dependency{external: cn.Namespace != g.ns}
		}
	}

	//Net
	if target.Config.Net != nil && target.Config.Net.Type == "container" {
		cn := target.Config.Net.Container
		if _, found := toResolve[cn]; !found {
			toResolve[cn] = &dependency{external: cn.Namespace != g.ns}
		}
	}

	//Notifications
	for _, n := range target.Config.Notify {
		cn := *n.ContainerName()
		if _, found := toNotify[cn]; !found {
			switch n.(type) {
			case *config.NotifyActionRestart, *config.NotifyActionRecreate:
				toNotify[cn] = &dependency{
					notify:   n,
					external: cn.Namespace != g.ns,
				}

			default:
				toNotify2[cn] = &dependency{
					notify:   n,
					external: cn.Namespace != g.ns,
				}
			}
		}
	}

	for name, dep := range toResolve {
		// in case of the same namespace, we should find dependency
		// in given configuration
		var scope []*Container = expected

		if dep.external {
			scope = actual
		}

		container := find(scope, &name)
		if container == nil {
			return fmt.Errorf("Cannot resolve dependency %s for %s", name, target)
		}

		dep.container = container
		g.dependencies[target] = append(g.dependencies[target], dep)
	}

	for name, dep := range toNotify {
		// in case of the same namespace, we should find dependency
		// in given configuration
		var scope []*Container = expected

		if dep.external {
			scope = actual
		}

		container := find(scope, &name)
		if container == nil {
			return fmt.Errorf("Cannot resolve dependency %s for %s", name, target)
		}

		dep.container = target
		g.dependencies[container] = append(g.dependencies[container], dep)
	}

	for name, dep := range toNotify2 {
		// in case of the same namespace, we should find dependency
		// in given configuration
		var scope []*Container = expected

		if dep.external {
			scope = actual
		}

		container := find(scope, &name)
		if container == nil {
			return fmt.Errorf("Cannot resolve dependency %s for %s", name, target)
		}

		dep.container = container
		g.notifications[target] = append(g.notifications[target], dep)
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
				}

				if finalized, contains := visited[dependency.container]; !dependency.external && (!contains || !finalized) {
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
						restartActions := []Action{
							NewStepAction(true, depActions...),
							NewRemoveContainerAction(actualContainer),
							NewRunContainerAction(container),
						}

						// in recovery mode we have to ensure containers are started
						if container.Name.Namespace != dg.ns {
							restartActions = []Action{
								NewStepAction(true, depActions...),
								NewEnsureContainerStateAction(container),
							}
						}

						// pretty.Println(dg.notifications[container])
						for _, dep := range dg.notifications[container] {
							restartActions = append(restartActions, NewNotifyAction(dep.container, dep.notify))
						}

						step = append(step, NewStepAction(false, restartActions...))

						// mark container as recreated
						restarted[container] = struct{}{}
						continue nextDependency
					}

					// adding ensure action if applicable
					step = append(step, NewStepAction(true, depActions...))
					continue nextDependency
				}
			}

			createActions := []Action{
				NewStepAction(true, depActions...),
				NewRunContainerAction(container),
			}
			for _, dep := range dg.notifications[container] {
				createActions = append(createActions, NewNotifyAction(dep.container, dep.notify))
			}

			// container is not exists
			step = append(step, NewStepAction(false, createActions...))
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
	fmt.Printf("---- hasCycles0 %s\n", curr.Name)
	for _, c := range path {
		fmt.Printf("PATH %s\n", c.Name)
	}
	for _, c := range path[:len(path)-1] {
		if c.IsSameKind(curr) {
			fmt.Printf("return true\n")
			return true
		}
	}
	if deps := dg.dependencies[curr]; deps != nil {
		depsStr := []string{}
		for _, d := range deps {
			depsStr = append(depsStr, d.container.Name.String())
		}
		fmt.Printf("deps of %s: %s\n", curr.Name, strings.Join(depsStr, ", "))
		for _, d := range deps {
			if dg.hasCycles0(append(path, d.container), d.container) {
				return true
			}
		}
	}
	return false
}
