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

	fmt.Printf("Config path: %s\n", configFilename)

	fmt.Printf("Config: %+q\n", config)
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
