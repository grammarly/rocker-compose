package compose

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestComparatorSameValue(t *testing.T) {
	cmp := NewDiff()
	containers := make([]*Container, 0)
	act, err := cmp.Diff("", containers, containers)
	assert.Empty(t, act)
	assert.Nil(t, err)
}

func TestComparatorDependencyGraph(t *testing.T) {
	cmp := NewDiff()
	containers := []*Container{}
	containers = append(containers,
		newContainer("test", "1", ContainerName{"test", "2"}, ContainerName{"test", "3"}),
		newContainer("test", "2", ContainerName{"test", "4"}),
		newContainer("test", "3", ContainerName{"test", "4"}),
		newContainer("test", "4"),
	)
	actions, _ := cmp.Diff("test", containers, []*Container{})
	runner := NewDryRunner()
	runner.Run(actions)
}

func newContainer(namespace string, name string, dependencies ...ContainerName) *Container {
	return &Container{
		Name: &ContainerName{namespace, name},
		Config: &ConfigContainer{
			VolumesFrom: dependencies,
		}}
}

