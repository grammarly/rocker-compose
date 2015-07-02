package compose

import (
	"io"

	log "github.com/Sirupsen/logrus"
)

type ContainerIo struct {
	Stdout io.Writer
	Stderr io.Writer

	done chan error
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

	return cio
}

func (cio *ContainerIo) Done(err error) {
	cio.done <- err
	return
}

func (cio *ContainerIo) Wait() error {
	return <-cio.done
}
