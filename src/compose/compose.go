package compose

import (
	log "github.com/Sirupsen/logrus"
	"github.com/kr/pretty"
)

type ComposeConfig struct {
	manifest	*Config
}

func Run(config *ComposeConfig) {
	log.Debugf("Running configuration: \n%s", pretty.Sprintf("%# v", config))
	//todo: implement running configuration
}