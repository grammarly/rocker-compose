package main

import (
	"compose"
	"flag"
	"fmt"
	"log"
	"os"
	"path"
)

func main() {
	var configFilename string
	flag.StringVar(&configFilename, "config", "compose.yml", "config file path")
	flag.Parse()

	if !path.IsAbs(configFilename) {
		wd, err := os.Getwd()
		if err != nil {
			log.Fatal(err)
		}
		configFilename = path.Join(wd, configFilename)
	}

	config, err := compose.ReadConfigFile(configFilename)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Config path: %s\n", configFilename)

	fmt.Printf("Config: %+q\n", config)
}
