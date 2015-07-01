package compose
import (
	log "github.com/Sirupsen/logrus"
)

type formatter struct {
	container *Container
	level     log.Level
	delegate  log.Formatter
}

func NewContainerFormatter(container *Container, level log.Level) log.Formatter {
	return &formatter{
		container:   container,
		level:        level,
		delegate:    log.StandardLogger().Formatter,
	}
}

func (f *formatter) Format(entry *log.Entry) ([]byte, error) {
	e := entry.WithFields(log.Fields{
		"container": f.container.Name.String(),
	})
	e.Message = entry.Message
	e.Level = f.level
	return f.delegate.Format(e)
}
