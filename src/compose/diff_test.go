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
	"testing"

	"github.com/grammarly/rocker/src/imagename"
	"github.com/grammarly/rocker/src/template"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestComparatorSameValue(t *testing.T) {
	cmp := NewDiff("")
	var containers []*Container
	act, err := cmp.Diff(containers, containers)
	assert.Empty(t, act)
	assert.Nil(t, err)
}

func TestDiffCreateAll(t *testing.T) {
	cmp := NewDiff("test")
	containers := []*Container{}
	c1 := newContainer("test", "1", config.ContainerName{Namespace: "test", Name: "2"}, config.ContainerName{Namespace: "test", Name: "3"})
	c2 := newContainer("test", "2", config.ContainerName{Namespace: "test", Name: "4"})
	c3 := newContainer("test", "3", config.ContainerName{Namespace: "test", Name: "4"})
	c4 := newContainer("test", "4")
	containers = append(containers, c1, c2, c3, c4)
	actions, _ := cmp.Diff(containers, []*Container{})
	mock := clientMock{}
	mock.On("RunContainer", c4).Return(nil)
	mock.On("RunContainer", c2).Return(nil)
	mock.On("RunContainer", c3).Return(nil)
	mock.On("RunContainer", c1).Return(nil)

	runner := NewDockerClientRunner(&mock)
	runner.Run(actions)
	mock.AssertExpectations(t)
}

func TestDiffNoDependencies(t *testing.T) {
	cmp := NewDiff("test")
	containers := []*Container{}
	c1 := newContainer("test", "1")
	c2 := newContainer("test", "2")
	c3 := newContainer("test", "3")
	containers = append(containers, c1, c2, c3)
	actions, _ := cmp.Diff(containers, []*Container{})
	mock := clientMock{}
	mock.On("RunContainer", c1).Return(nil)
	mock.On("RunContainer", c2).Return(nil)
	mock.On("RunContainer", c3).Return(nil)
	runner := NewDockerClientRunner(&mock)
	runner.Run(actions)
	mock.AssertExpectations(t)
}

func TestDiffAddingOneContainer(t *testing.T) {
	cmp := NewDiff("test")
	containers := []*Container{}
	c1 := newContainer("test", "1")
	c2 := newContainer("test", "2")
	c3 := newContainer("test", "3")
	containers = append(containers, c1, c2)
	actions, _ := cmp.Diff(containers, []*Container{c1, c3})
	mock := clientMock{}
	mock.On("RunContainer", c2).Return(nil)
	mock.On("RemoveContainer", c3).Return(nil)
	runner := NewDockerClientRunner(&mock)
	runner.Run(actions)
	mock.AssertExpectations(t)
}

func TestDiffExternalDependencies(t *testing.T) {
	cmp := NewDiff("test")
	c1 := newContainer("metrics", "1")
	c2 := newContainer("metrics", "2")
	c3 := newContainer("metrics", "3")
	actions, _ := cmp.Diff([]*Container{}, []*Container{c1, c2, c3})
	mock := clientMock{}
	runner := NewDockerClientRunner(&mock)
	runner.Run(actions)
	mock.AssertExpectations(t)
}

func TestDiffRunningOnce(t *testing.T) {
	var once config.State = "ran"
	cmp := NewDiff("test")
	c1 := newContainer("test", "1")
	c1.Config.State = &once

	mock := clientMock{}

	mock.On("RunContainer", c1).Return(nil)
	mock.Once()

	runner := NewDockerClientRunner(&mock)
	actions, _ := cmp.Diff([]*Container{c1}, []*Container{})
	runner.Run(actions)

	c2 := newContainer("test", "1")
	c2.Config.State = &once
	c2.State.ExitCode = 0

	actions, _ = cmp.Diff([]*Container{c1}, []*Container{c2})
	runner.Run(actions)

	mock.AssertExpectations(t)
}

func TestDiffRunningOnceWithNonZero(t *testing.T) {
	var once config.State = "ran"
	cmp := NewDiff("test")
	c1 := newContainer("test", "1")
	c1.Config.State = &once

	mock := clientMock{}

	mock.On("RunContainer", c1).Return(nil)

	runner := NewDockerClientRunner(&mock)
	actions, _ := cmp.Diff([]*Container{c1}, []*Container{})
	runner.Run(actions)

	c2 := newContainer("test", "1")
	c2.Config.State = &once
	c2.State.ExitCode = 137

	mock.On("RemoveContainer", c2).Return(nil)
	mock.On("RunContainer", c1).Return(nil)

	actions, _ = cmp.Diff([]*Container{c1}, []*Container{c2})
	runner.Run(actions)

	mock.AssertExpectations(t)
}

func TestDiffEnsureFewExternalDependencies(t *testing.T) {
	cmp := NewDiff("test")
	c1 := newContainer("metrics", "1")
	c2 := newContainer("metrics", "2")
	c3 := newContainer("metrics", "3")
	c4 := newContainer("test", "1", config.ContainerName{Namespace: "metrics", Name: "1"},
		config.ContainerName{Namespace: "metrics", Name: "2"}, config.ContainerName{Namespace: "metrics", Name: "3"})
	actions, _ := cmp.Diff([]*Container{c4}, []*Container{c1, c2, c3})
	mock := clientMock{}
	mock.On("EnsureContainerExist", c1).Return(nil)
	mock.On("EnsureContainerExist", c2).Return(nil)
	mock.On("EnsureContainerExist", c3).Return(nil)
	mock.On("RunContainer", c4).Return(nil)
	runner := NewDockerClientRunner(&mock)
	t.Log(runner, actions)
	runner.Run(actions)
	mock.AssertExpectations(t)
}

func TestDiffFailInMiddle(t *testing.T) {
	cmp := NewDiff("test")
	c1 := newContainer("test", "1")
	c2 := newContainer("test", "2")
	c3 := newContainer("test", "3")
	actions, _ := cmp.Diff([]*Container{c1, c2, c3}, []*Container{})
	mock := clientMock{}
	mock.On("RunContainer", c1).Return(nil)
	mock.On("RunContainer", c2).Return(fmt.Errorf("fail"))
	mock.On("RunContainer", c3).Return(nil)
	runner := NewDockerClientRunner(&mock)
	assert.Error(t, runner.Run(actions))
	mock.AssertExpectations(t)
}

func TestDiffFailInDependent(t *testing.T) {
	cmp := NewDiff("test")
	c1 := newContainer("test", "1", config.ContainerName{Namespace: "test", Name: "2"})
	c2 := newContainer("test", "2")
	c3 := newContainer("test", "3", config.ContainerName{Namespace: "test", Name: "2"})
	actions, _ := cmp.Diff([]*Container{c1, c2, c3}, []*Container{})
	mock := clientMock{}
	mock.On("RunContainer", c2).Return(fmt.Errorf("fail"))
	runner := NewDockerClientRunner(&mock)
	assert.Error(t, runner.Run(actions))
	mock.AssertExpectations(t)
}

func TestDiffInDependent(t *testing.T) {
	cmp := NewDiff("test")
	c1 := newContainer("test", "1", config.ContainerName{Namespace: "test", Name: "2"})
	c2 := newContainer("test", "2")
	c2x := newContainer("test", "2")
	c2x.Config.Labels = map[string]string{"test": "test2"}
	actions, _ := cmp.Diff([]*Container{c1, c2x}, []*Container{c1, c2})
	mock := clientMock{}
	mock.On("RemoveContainer", c2).Return(nil)
	mock.On("RunContainer", c2x).Return(nil)
	mock.On("RemoveContainer", c1).Return(nil)
	mock.On("RunContainer", c1).Return(nil)
	runner := NewDockerClientRunner(&mock)
	runner.Run(actions)
	mock.AssertExpectations(t)
}

func TestDiffInDependentNet(t *testing.T) {
	cmp := NewDiff("test")
	c2NetName, err := config.NewNetFromString("container:test.2")
	if err != nil {
		t.Fatal(err)
	}
	c1 := &Container{
		State:  &ContainerState{Running: true},
		Name:   &config.ContainerName{Namespace: "test", Name: "1"},
		Config: &config.Container{Net: c2NetName},
	}
	c2 := &Container{
		State:  &ContainerState{Running: true},
		Name:   &config.ContainerName{Namespace: "test", Name: "2"},
		Config: &config.Container{},
	}
	c2x := &Container{
		State:  &ContainerState{Running: true},
		Name:   &config.ContainerName{Namespace: "test", Name: "2"},
		Config: &config.Container{Labels: map[string]string{"test": "test2"}},
	}
	actions, _ := cmp.Diff([]*Container{c1, c2x}, []*Container{c1, c2})
	mock := clientMock{}
	mock.On("RemoveContainer", c2).Return(nil)
	mock.On("RunContainer", c2x).Return(nil)
	mock.On("RemoveContainer", c1).Return(nil)
	mock.On("RunContainer", c1).Return(nil)
	runner := NewDockerClientRunner(&mock)
	runner.Run(actions)
	mock.AssertExpectations(t)
}

func TestDiffInDependentExternalNet(t *testing.T) {
	cmp := NewDiff("test")
	c2NetName, err := config.NewNetFromString("container:external.2")
	if err != nil {
		t.Fatal(err)
	}
	c1 := &Container{
		State:  &ContainerState{Running: true},
		Name:   &config.ContainerName{Namespace: "test", Name: "1"},
		Config: &config.Container{Net: c2NetName},
	}
	c2 := newContainer("external", "2")
	actions, _ := cmp.Diff([]*Container{c1}, []*Container{c1, c2})
	mock := clientMock{}
	mock.On("EnsureContainerExist", c2).Return(nil)
	runner := NewDockerClientRunner(&mock)
	runner.Run(actions)
	mock.AssertExpectations(t)
}

func TestDiffForCycles(t *testing.T) {
	cmp := NewDiff("test")
	containers := []*Container{}
	c1 := newContainer("test", "1", config.ContainerName{Namespace: "test", Name: "2"})
	c2 := newContainer("test", "2", config.ContainerName{Namespace: "test", Name: "3"})
	c3 := newContainer("test", "3", config.ContainerName{Namespace: "test", Name: "1"})
	containers = append(containers, c1, c2, c3)
	_, err := cmp.Diff(containers, []*Container{c1, c3})
	assert.Error(t, err)
}

func TestDiffDifferentConfig(t *testing.T) {
	cmp := NewDiff("test")
	containers := []*Container{}
	cpusetCpus1 := "0-2"
	cpusetCpus2 := "0-4"
	c1x := &Container{
		State:  &ContainerState{Running: true},
		Name:   &config.ContainerName{Namespace: "test", Name: "1"},
		Config: &config.Container{CpusetCpus: &cpusetCpus1},
	}
	c1y := &Container{
		State:  &ContainerState{Running: true},
		Name:   &config.ContainerName{Namespace: "test", Name: "1"},
		Config: &config.Container{CpusetCpus: &cpusetCpus2},
	}
	containers = append(containers, c1x)
	actions, _ := cmp.Diff(containers, []*Container{c1y})
	mock := clientMock{}
	mock.On("RemoveContainer", c1y).Return(nil)
	mock.On("RunContainer", c1x).Return(nil)
	runner := NewDockerClientRunner(&mock)
	runner.Run(actions)
	mock.AssertExpectations(t)
}

func TestDiffForExternalDependencies(t *testing.T) {
	cmp := NewDiff("test")
	containers := []*Container{}
	c1 := newContainer("test", "1")
	c2 := newContainer("test", "2", config.ContainerName{Namespace: "metrics", Name: "1"})
	m1 := newContainer("metrics", "1")
	containers = append(containers, c1, c2)
	actions, _ := cmp.Diff(containers, []*Container{m1})
	mock := clientMock{}
	mock.On("EnsureContainerExist", m1).Return(nil)
	mock.On("RunContainer", c1).Return(nil)
	mock.On("RunContainer", c2).Return(nil)
	runner := NewDockerClientRunner(&mock)
	runner.Run(actions)
	mock.AssertExpectations(t)
}

func TestDiffCreateRemoving(t *testing.T) {
	cmp := NewDiff("test")
	containers := []*Container{}
	c1 := newContainer("test", "1", config.ContainerName{Namespace: "test", Name: "2"}, config.ContainerName{Namespace: "test", Name: "3"})
	c2 := newContainer("test", "2", config.ContainerName{Namespace: "test", Name: "4"})
	c3 := newContainer("test", "3", config.ContainerName{Namespace: "test", Name: "4"})
	c4 := newContainer("test", "4")
	c5 := newContainer("test", "5")
	containers = append(containers, c1, c2, c3, c4)
	actions, _ := cmp.Diff(containers, []*Container{c5})
	mock := clientMock{}
	mock.On("RemoveContainer", c5).Return(nil)
	mock.On("RunContainer", c4).Return(nil)
	mock.On("RunContainer", c2).Return(nil)
	mock.On("RunContainer", c3).Return(nil)
	mock.On("RunContainer", c1).Return(nil)
	runner := NewDockerClientRunner(&mock)
	runner.Run(actions)
	mock.AssertExpectations(t)
}

func TestDiffCreateSome(t *testing.T) {
	cmp := NewDiff("test")
	containers := []*Container{}
	c1 := newContainer("test", "1", config.ContainerName{Namespace: "test", Name: "2"}, config.ContainerName{Namespace: "test", Name: "3"})
	c2 := newContainer("test", "2", config.ContainerName{Namespace: "test", Name: "4"})
	c3 := newContainer("test", "3", config.ContainerName{Namespace: "test", Name: "4"})
	c4 := newContainer("test", "4")
	containers = append(containers, c1, c2, c3, c4)
	actions, _ := cmp.Diff(containers, []*Container{c1})
	mock := clientMock{}
	mock.On("RunContainer", c4).Return(nil)
	mock.On("RunContainer", c2).Return(nil)
	mock.On("RunContainer", c3).Return(nil)
	runner := NewDockerClientRunner(&mock)
	runner.Run(actions)
	mock.AssertExpectations(t)
}

func TestWaitForStart(t *testing.T) {
	cmp := NewDiff("test")
	c1 := newContainerWaitFor("test", "1", config.ContainerName{Namespace: "test", Name: "2"})
	c2 := newContainer("test", "2")
	actions, _ := cmp.Diff([]*Container{c1, c2}, []*Container{})
	mock := clientMock{}
	mock.On("RunContainer", c2).Return(nil)
	mock.On("WaitForContainer", c2).Return(nil)
	mock.On("RunContainer", c1).Return(nil)
	runner := NewDockerClientRunner(&mock)
	runner.Run(actions)
	mock.AssertExpectations(t)
}

func TestWaitForNotRestart(t *testing.T) {
	cmp := NewDiff("test")
	c1 := newContainerWaitFor("test", "1", config.ContainerName{Namespace: "test", Name: "2"})
	c2 := newContainer("test", "2")
	c2x := newContainer("test", "2")
	c2x.Config.Labels = map[string]string{"test": "test2"}
	actions, _ := cmp.Diff([]*Container{c1, c2x}, []*Container{c1, c2})
	mock := clientMock{}
	mock.On("RemoveContainer", c2).Return(nil)
	mock.On("RunContainer", c2x).Return(nil)
	mock.On("WaitForContainer", c2x).Return(nil)
	runner := NewDockerClientRunner(&mock)
	runner.Run(actions)
	mock.AssertExpectations(t)
}

func TestDiffRecovery(t *testing.T) {
	cmp := NewDiff("")
	c1x := &Container{
		State:  &ContainerState{Running: true},
		Name:   &config.ContainerName{Namespace: "test", Name: "1"},
		Config: &config.Container{},
	}
	c1y := &Container{
		State:  &ContainerState{Running: false},
		Name:   &config.ContainerName{Namespace: "test", Name: "1"},
		Config: &config.Container{},
	}
	actions, _ := cmp.Diff([]*Container{c1x}, []*Container{c1y})
	mock := clientMock{}
	mock.On("EnsureContainerState", c1x).Return(nil)
	runner := NewDockerClientRunner(&mock)
	runner.Run(actions)
	mock.AssertExpectations(t)
}

func newContainer(namespace string, name string, dependencies ...config.ContainerName) *Container {
	return &Container{
		State: &ContainerState{
			Running: true,
		},
		Name: &config.ContainerName{Namespace: namespace, Name: name},
		Config: &config.Container{
			VolumesFrom: dependencies,
		}}
}

func newContainerWaitFor(namespace string, name string, dependencies ...config.ContainerName) *Container {
	return &Container{
		State: &ContainerState{
			Running: true,
		},
		Name: &config.ContainerName{Namespace: namespace, Name: name},
		Config: &config.Container{
			WaitFor: dependencies,
		}}
}

// clientMock implementation

func (m *clientMock) GetContainers(global bool) ([]*Container, error) {
	args := m.Called()
	return nil, args.Error(0)
}

func (m *clientMock) RemoveContainer(container *Container) error {
	args := m.Called(container)
	return args.Error(0)
}

func (m *clientMock) RunContainer(container *Container) error {
	args := m.Called(container)
	return args.Error(0)
}

func (m *clientMock) EnsureContainerExist(container *Container) error {
	args := m.Called(container)
	return args.Error(0)
}

func (m *clientMock) EnsureContainerState(container *Container) error {
	args := m.Called(container)
	return args.Error(0)
}

func (m *clientMock) PullImage(imageName *imagename.ImageName) error {
	args := m.Called(imageName)
	return args.Error(0)
}

func (m *clientMock) PullAll(containers []*Container, vars template.Vars) error {
	args := m.Called(containers, vars)
	return args.Error(0)
}

func (m *clientMock) Clean(cfg *config.Config) error {
	args := m.Called(cfg)
	return args.Error(0)
}

func (m *clientMock) AttachToContainer(container *Container) error {
	args := m.Called(container)
	return args.Error(0)
}

func (m *clientMock) AttachToContainers(container []*Container) error {
	args := m.Called(container)
	return args.Error(0)
}

func (m *clientMock) FetchImages(container []*Container, vars template.Vars) error {
	args := m.Called(container, vars)
	return args.Error(0)
}

func (m *clientMock) WaitForContainer(container *Container) error {
	args := m.Called(container)
	return args.Error(0)
}

func (m *clientMock) GetPulledImages() []*imagename.ImageName {
	m.Called()
	return []*imagename.ImageName{}
}

func (m *clientMock) GetRemovedImages() []*imagename.ImageName {
	m.Called()
	return []*imagename.ImageName{}
}

func (m *clientMock) Pin(local, hub bool, vars template.Vars, container []*Container) error {
	args := m.Called(local, hub, vars, container)
	return args.Error(0)
}

type clientMock struct {
	mock.Mock
}
