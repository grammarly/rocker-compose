package compose

import (
	"fmt"
)

type Diff interface {
	Diff(ns string, expected []*Container, actual []*Container) ([]Action, error)
}

type comparator struct {}

type dependencyGraph struct {
	dependencies map[*Container][]*dependency
}

type dependency struct {
	container *Container
	external  bool
}

func NewDiff() Diff {
	return &comparator{}
}

func (c *comparator) Diff(ns string, expected []*Container, actual []*Container) (res []Action, err error) {
	var depGraph *dependencyGraph
	depGraph, err = buildDependencyGraph(ns, expected, actual)
	if err != nil {
		res = []Action{}
		return
	}

	if depGraph.hasCycles() {
		err = fmt.Errorf("Dependencies have cycles")
		return
	}

	res = getContainersToRemove(ns, expected, actual)
	res = append(res, depGraph.buildExecutionPlan(actual)...)

	return
}

func buildDependencyGraph(ns string, expected []*Container, actual []*Container) (*dependencyGraph, error) {
	dg := dependencyGraph{
		dependencies: make(map[*Container][]*dependency),
	}

	for _, c := range expected {
		dg.dependencies[c] = []*dependency{}
		dependencies, err := getDependencies(ns, expected, actual, c)
		if err != nil {
			return nil, err
		}
		dg.dependencies[c] = append(dg.dependencies[c], dependencies...)
	}

	return &dg, nil
}

func getDependencies(ns string, expected []*Container, actual []*Container, target *Container) (resolved []*dependency, err error) {
	resolved = []*dependency{}
	toResolve := append(target.Config.VolumesFrom, target.Config.Links...)
	for _, dep := range toResolve {
		if dep.Namespace == ns {
			if d := find(expected, &dep); d != nil {
				resolved = append(resolved, &dependency{container: d, external: false})
			} else {
				err = fmt.Errorf("Cannot find internal dependency at config %s", dep.String())
			}
		} else {
			if d := find(actual, &dep); d != nil {
				resolved = append(resolved, &dependency{container: d, external: true})
			} else {
				err = fmt.Errorf("Cannot find extenal dependency %s at target system", dep.String())
			}
		}
	}
	return
}

func getContainersToRemove(ns string, expected []*Container, actual []*Container) (res []Action) {
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

func (dg *dependencyGraph) buildExecutionPlan(actual []*Container) (res []Action) {
	visited := map[*Container]bool{}
	restarted := map[*Container]struct {}{}

	for len(visited) < len(dg.dependencies) {
		var step []Action = []Action{}

		nextDep:
		for container, deps := range dg.dependencies {
			if _, contains := visited[container]; contains {
				continue
			}

			var ensures []Action = []Action{}
			var restart bool

			for _, dependency := range deps {
				if dependency.external {
					ensures = append(ensures, NewEnsureContainerAction(dependency.container))
				} else if finalized, contains := visited[dependency.container]; !contains || !finalized {
					continue nextDep
				}
				_, contains := restarted[dependency.container]
				restart = restart || contains
			}

			visited[container] = false

			for _, actualContainer := range actual {
				if container.IsSameKind(actualContainer) {
					if !container.IsEqualTo(actualContainer) || restart {
						step = append(step, NewStepAction(false,
							NewStepAction(true, ensures...),
							NewRemoveContainerAction(actualContainer),
							NewRunContainerAction(container),
						))
						restarted[container] = struct {}{}
						continue nextDep
					}
					step = append(step, NewStepAction(true, ensures...))
					continue nextDep
				}
			}

			step = append(step, NewStepAction(false,
				NewStepAction(true, ensures...),
				NewRunContainerAction(container),
			))
		}

		//finalize step
		for k, v := range visited {
			if !v {
				visited[k] = true
			}
		}
		res = append(res, NewStepAction(true, step...))
	}
	return
}


func find(containers []*Container, name *ContainerName) *Container {
	for _, c := range containers {
		if c.Name.IsEqualTo(name) {
			return c
		}
	}
	return nil
}

func (dg *dependencyGraph) hasCycles() bool {
	for k, _ := range dg.dependencies {
		if dg.hasCycles0([]*Container{k}, k) {
			return true
		}
	}
	return false
}

func (dg *dependencyGraph) hasCycles0(path []*Container, curr *Container) bool {
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