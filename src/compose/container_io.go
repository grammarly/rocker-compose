/*-
 * Copyright 2014 Grammarly, Inc.
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
	"io"
	"time"

	log "github.com/Sirupsen/logrus"
)

type ContainerIo struct {
	Stdout io.Writer
	Stderr io.Writer

	done  chan error
	alive bool
}

func NewContainerIo(container *Container) *ContainerIo {
	def := log.StandardLogger()
	outLogger := &log.Logger{
		Out:       def.Out,
		Formatter: NewContainerFormatter(container, log.InfoLevel),
		Level:     def.Level,
	}
	errLogger := &log.Logger{
		Out:       def.Out,
		Formatter: NewContainerFormatter(container, log.ErrorLevel),
		Level:     def.Level,
	}

	cio := &ContainerIo{}
	cio.Stdout = outLogger.Writer()
	cio.Stderr = errLogger.Writer()
	cio.done = make(chan error, 1)
	cio.alive = true

	return cio
}

func (cio *ContainerIo) Resurrect() {
	cio.alive = true
}

func (cio *ContainerIo) Done(err error) {
	cio.alive = false
	time.Sleep(1 * time.Second)

	// if io was resurrected
	if cio.alive {
		return
	}

	cio.done <- err
	return
}

func (cio *ContainerIo) Wait() error {
	return <-cio.done
}
