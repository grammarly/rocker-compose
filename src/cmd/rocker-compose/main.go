package main

import (
	"compose"
	"fmt"
	"os"
	"path"
	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
)

func init() {
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)
}

func main() {
	app := cli.NewApp()
	app.Name = "rocker-compose"
	app.Version = "0.0.1"
	app.Usage = "Tool for docker orchestration"
	app.Commands = []cli.Command{
		{
			Name:    "run",
			Usage:    "execute manifest",
			Action: run,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name: "log, l",
				},
				cli.StringFlag{
					Name: "manifest, m",
				},
				cli.BoolFlag{
					Name: "verbose, v",
				},
			},
		},
	}
	app.Run(os.Args)
}

func run(ctx *cli.Context) {
	if ctx.Bool("verbose") {
		log.SetLevel(log.DebugLevel)
	}

	if logFilename, err := toAbsolutePath(ctx.String("log"), false); err != nil {
		log.Debugf("Initializing log: Skipped, because Log %s", err)
	}else {
		logFile, err := os.OpenFile(logFilename, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0755)
		if err != nil {
			log.Warnf("Initializing log: Cannot initialize log file %s due to error %s", logFilename, err)
		}

		if path.Ext(logFilename) == "json" {
			log.Debugf("Initializing log: Using JSON as a result format")
			log.SetFormatter(&log.JSONFormatter{})
		}
		log.SetOutput(logFile)

		log.Debugf("Initializing log: Successfuly started loggin to '%s'", logFilename)
	}

	log.Debugf("Reading manifest: '%s'", ctx.String("manifest"))
	if configFilename, err := toAbsolutePath(ctx.String("manifest"), true); err != nil {
		log.Fatalf("Cannot read manifest: %s", err)
		os.Exit(1) // no config - no pichenka
	} else {
		config, err := compose.ReadConfigFile(configFilename, map[string]interface{}{})
		if err != nil {
			log.Fatal(err)
		}
		log.Infof("Successfully read manifest: '%s'", configFilename)

		compose.Run(
			&compose.ComposeConfig{
				manifest:	config,
			})
	}

	// if c.GlobalIsSet("tlsverify") {
	//   config.tlsverify = c.GlobalBool("tlsverify")
	//   config.tlscacert = globalString(c, "tlscacert")
	//   config.tlscert = globalString(c, "tlscert")
	//   config.tlskey = globalString(c, "tlskey")
	// }
}

func toAbsolutePath(filePath string, shouldExist bool) (string, error) {
	if filePath == "" {
		return filePath, fmt.Errorf("File path is not provided")
	}

	if !path.IsAbs(filePath) {
		wd, err := os.Getwd()
		if err != nil {
			log.Errorf("Cannot get absolute path to %s due to error %s", filePath, err)
			return filePath, err
		}
		filePath = path.Join(wd, filePath)
	}

	if _, err := os.Stat(filePath); os.IsNotExist(err) && shouldExist {
		return filePath, fmt.Errorf("No such file or directory: %s", filePath)
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
