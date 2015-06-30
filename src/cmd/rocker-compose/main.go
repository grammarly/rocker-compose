package main

import (
	"compose"
	"flag"
	"fmt"
	"os"
	"path"
	log "github.com/Sirupsen/logrus"
)

func main() {
	var configFilename string
	var logFilename string
	var verbose bool
	var err error

	flag.Bool("verbose", verbose, "Set logging level to debug")
	flag.StringVar(&logFilename, "log", "rocker-compose.log", "path to log file")
	flag.StringVar(&configFilename, "config", "compose.yml", "config file path")
	flag.Parse()

	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)

	if logFilename, err = toAbsolutePath(logFilename); err == nil {
		logFile, err := os.OpenFile(logFilename, os.O_WRONLY|os.O_CREATE, 0755)
		if err != nil {
			log.Warnf("Cannot initialize log file %s due to error %s", logFilename, err)
		}

		if path.Ext(logFilename) == "json" {
			log.SetFormatter(&log.JSONFormatter{})
		}

		log.SetOutput(logFile)
	}

	if configFilename, err = toAbsolutePath(configFilename); err != nil {
		log.Fatal(err)
//		os.Exit(1) // no config - no pichenka
	}

	if verbose {
		log.SetLevel(log.DebugLevel)
	}

	config, err := compose.ReadConfigFile(configFilename, map[string]interface{}{})
	if err != nil {
		log.Fatal(err)
	}

	// if c.GlobalIsSet("tlsverify") {
	//   config.tlsverify = c.GlobalBool("tlsverify")
	//   config.tlscacert = globalString(c, "tlscacert")
	//   config.tlscert = globalString(c, "tlscert")
	//   config.tlskey = globalString(c, "tlskey")
	// }

	log.Infof("Config path: %s\n", configFilename)
	log.Infof("Config: %+q\n", config)
}

func toAbsolutePath(filePath string) (string, error)  {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return filePath, fmt.Errorf("No such file or directory: %s", filePath)
	}

	if !path.IsAbs(filePath) {
		wd, err := os.Getwd()
		if err != nil {
			log.Errorf("Cannot get absolute path to %s due to error %s", filePath, err)
			return filePath, err
		}
		return path.Join(wd, filePath), nil
	}
	return filePath, nil
}

// Fix string arguments enclosed with boudle quotes
// 'docker-machine config' gives such arguments
// func globalString(c *cli.Context, name string) string {
// 	str := c.GlobalString(name)
// 	if len(str) >= 2 && str[0] == '\u0022' && str[len(str)-1] == '\u0022' {
// 		str = str[1 : len(str)-1]
// 	}
// 	return str
// }
