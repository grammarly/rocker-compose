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
	if depGraph, err = buildDependencyGraph(ns, expected, actual); err != nil {
		res = []Action{NoAction}
	}else {
		if depGraph.hasCycles() {
			fmt.Println("Graph has cycles")
			err = fmt.Errorf("Graph has cycles")
			return
		}
		var sortedGraph [][]*dependency
		if sortedGraph, err = depGraph.topologicalSort(); err != nil {
			return
		}
		res = findContainersToShutdown(ns, expected, actual)
		res = append(res, convertContainersToActions(sortedGraph, actual)...)
	}
	return
}

func buildDependencyGraph(ns string, expected []*Container, actual []*Container) (*dependencyGraph, error) {
	dg := dependencyGraph{dependencies: make(map[*Container][]*dependency) }
	for _, c := range expected {
		if dependencies, err := getDependencies(ns, expected, actual, c); err != nil {
			return nil, err
		}else if dependencies != nil {
			dg.dependencies[c] = append(dg.dependencies[c], dependencies...)
		}
	}
	return &dg, nil
}

func getDependencies(ns string, expected []*Container, actual []*Container, target *Container) (resolved []*dependency, err error) {
	resolved = []*dependency{}
	toResolve := append(target.Config.VolumesFrom, target.Config.Links...)
	for _, dep := range toResolve {
		if dep.Namespace == ns {
			if d := find(expected, &dep); d != nil {
				resolved = append(resolved, &dependency{container: d, external: false })
			}else {
				err = fmt.Errorf("Cannot find internal dependency at config %s", dep.String())
			}
		} else {
			if d := find(actual, &dep); d != nil {
				resolved = append(resolved, &dependency{container: d, external: true })
			}else {
				err = fmt.Errorf("Cannot find extenal dependency %s at target system", dep.String())
			}
		}
	}
	return
}

func findContainersToShutdown(ns string, expected []*Container, actual []*Container) (res []Action) {
	for _, a := range actual {
		if (a.Name.Namespace == ns) {
			var found bool
			for _, e := range expected {
				found = found || e.IsEqualTo(a)
			}
			if !found {
				res = append(res, NewRemoveContainerAction(a))
			}
		}
	}
	return
}

func convertContainersToActions(containers [][]*dependency, actual []*Container) (res []Action) {
	for _, step := range containers {
		stepActions := []Action{}
		for _, container := range step {
			stepActions = append(stepActions, convertContainerToAction(container, actual))
		}
		if (len(stepActions)>1) {
			res = append(res, NewStepAction(true, stepActions...))
		}else {
			res = append(res, stepActions...)
		}
	}
	return
}

func convertContainerToAction(dep *dependency, actual []*Container) (res Action) {
	for _, actualContainer := range actual {
		if dep.external {
			res = NewEnsureContainerAction(dep.container)
			return
		} else if dep.container.IsSameKind(actualContainer) {
			if (dep.container.IsEqualTo(actualContainer)) {
				res = NoAction
			}else {
				res = NewStepAction(false,
					NewRemoveContainerAction(actualContainer),
					NewCreateContainerAction(dep.container),
				)
			}
			return
		}
	}
	res = NewCreateContainerAction(dep.container)
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
		if dg.hasCycles0([]*Container{k}, k){
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
			if dg.hasCycles0(append(path, d.container), d.container){
				return true
			}
		}
 	}
	return false
}

func (dg *dependencyGraph) topologicalSort() ([][]*dependency, error) {
	res := [][]*dependency{}
	visited := map[string]struct {}{}
	for len(dg.dependencies)>0 {
		cont := []*dependency{}
		processing := []*dependency{}

		for k, v := range dg.dependencies {
			if len(v) == 0 {
				processing = append(processing, &dependency{container: k, external:false})
				delete(dg.dependencies, k)
			}

			for i, c := range v {
				if _, ok := dg.dependencies[c.container]; !ok {
					processing = append(processing, c)
					dg.dependencies[k] = append(v[:i], v[i+1:]...)
				}
			}
		}

		for _, p := range processing {
			if _, ok := visited[p.container.Name.String()]; !ok {
				cont = append(cont, p)
				visited[p.container.Name.String()] = struct {}{}
			}
		}

		if len(cont) != 0 {
			res = append(res, cont)
		}
	}
	return res, nil
}




