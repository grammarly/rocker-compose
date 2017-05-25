/*-
 * Copyright 2015 Grammarly, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package compose

import (
	"fmt"
	"github.com/snkozlov/rocker-compose/src/compose/config"
)

// Diff describes a comparison functionality of two container sets: expected and actual
// 'expected' is a list of containers from a spec (compose.yml)
// 'actual' is a list of existing containers in a docker daemon
//
// The Diff function should return the action list that is needed to transition form
// the 'actual' state to the 'expected' one.
type Diff interface {
	Diff(expected []*Container, actual []*Container) ([]Action, error)
}

// graph with container dependencies
type graph struct {
	ns           string
	dependencies map[*Container][]*dependency
}

// single dependency (external - means not in our namespace)
type dependency struct {
	container *Container
	external  bool
	waitForIt bool
}

// NewDiff returns an implementation of Diff object
func NewDiff(ns string) Diff {
	return &graph{
		ns:           ns,
		dependencies: make(map[*Container][]*dependency),
	}
}

// Diff compares 'expected' and 'actual' state by detecting changes and building
// a gependency graph, and returns an action list that is needed to transition
// from 'actual' state to 'expected' one.
func (g *graph) Diff(expected []*Container, actual []*Container) (res []Action, err error) {
	//filling dependency graph
	err = g.buildDependencyGraph(expected, actual)
	if err != nil {
		res = []Action{}
		return
	}

	//check for cycles in configuration
	if g.hasCycles() {
		err = fmt.Errorf("Dependencies have cycles, check links and volumes-from")
		return
	}

	res = listContainersToRemove(g.ns, expected, actual)
	res = append(res, g.buildExecutionPlan(actual)...)
	return
}

func (g *graph) buildDependencyGraph(expected []*Container, actual []*Container) error {
	for _, c := range expected {
		g.dependencies[c] = []*dependency{}
		dependencies, err := resolveDependencies(g.ns, expected, actual, c)
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
	for _, link := range target.Config.Links {
		cn := link.ContainerName
		if _, found := toResolve[cn]; !found {
			toResolve[cn] = &dependency{external: cn.Namespace != ns}
		}
	}

	//Net
	if target.Config.Net != nil && target.Config.Net.Type == "container" {
		cn := target.Config.Net.Container
		if _, found := toResolve[cn]; !found {
			toResolve[cn] = &dependency{external: cn.Namespace != ns}
		}
	}

	for name, dep := range toResolve {
		// in case of the same namespace, we should find dependency
		// in given configuration
		var scope = expected

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

func (g *graph) buildExecutionPlan(actual []*Container) (res []Action) {
	visited := map[*Container]bool{}
	restarted := map[*Container]struct{}{}

	// while number of visited deps less than number of
	// dependencies which should be visited - loop
	for len(visited) < len(g.dependencies) {
		var step = []Action{}

	nextDependency:
		for container, deps := range g.dependencies {
			// if dependency is already visited - skip it
			if _, contains := visited[container]; contains {
				continue
			}

			var depActions = []Action{}
			var restart bool

			// check transitive dependencies of current dependency
			for _, dependency := range deps {

				if finalized, contains := visited[dependency.container]; !dependency.external && (!contains || !finalized) {
					continue nextDependency
				}

				// for all external dependencies (in other namespace), ensure that it exists
				if dependency.waitForIt {
					depActions = append(depActions, NewWaitContainerAction(dependency.container))
				} else if dependency.external {
					depActions = append(depActions, NewEnsureContainerExistAction(dependency.container))
				} else {
					// if dependency should be restarted - we should restart current one
					_, contains := restarted[dependency.container]
					restart = restart || contains
				}
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
						if container.Name.Namespace != g.ns {
							restartActions = []Action{
								NewStepAction(true, depActions...),
								NewEnsureContainerStateAction(container),
							}
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

			// container is not exists
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

func (g *graph) hasCycles() bool {
	for k := range g.dependencies {
		if g.hasCycles0([]*Container{k}, k) {
			return true
		}
	}
	return false
}

func (g *graph) hasCycles0(path []*Container, curr *Container) bool {
	for _, c := range path[:len(path)-1] {
		if c.IsSameKind(curr) {
			return true
		}
	}
	if deps := g.dependencies[curr]; deps != nil {
		for _, d := range deps {
			if g.hasCycles0(append(path, d.container), d.container) {
				return true
			}
		}
	}
	return false
}
