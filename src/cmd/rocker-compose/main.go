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
	app.Authors = []cli.Author{
		{"Yura Bogdanov", "yuriy.bogdanov@grammarly.com"},
		{"Stas Levental", "stas.levental@grammarly.com"},
	}
	app.Flags = []cli.Flag{
		cli.IntFlag{
			Name: "timeout, t",
			Value:    10000,
		},
		cli.BoolFlag{
			Name: "verbose, vv",
		},
		cli.StringFlag{
			Name: "log, l",
		},
	}

	app.Commands = []cli.Command{
		{
			Name:    "run",
			Usage:    "execute manifest",
			Action: run,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:   "host, H",
					Value:  "unix:///var/run/docker.sock",
					Usage:  "Daemon socket(s) to connect to",
					EnvVar: "DOCKER_HOST",
				},
				cli.BoolFlag{
					Name:  "tlsverify, tls",
					Usage: "Use TLS and verify the remote",
				},
				cli.StringFlag{
					Name:  "tlscacert",
					Value: "~/.docker/ca.pem",
					Usage: "Trust certs signed only by this CA",
				},
				cli.StringFlag{
					Name:  "tlscert",
					Value: "~/.docker/cert.pem",
					Usage: "Path to TLS certificate file",
				},
				cli.StringFlag{
					Name:  "tlskey",
					Value: "~/.docker/key.pem",
					Usage: "Path to TLS key file",
				},
				cli.StringFlag{
					Name: "manifest, m",
					Usage: "Path to configuration file which should be run",
				},
				cli.BoolFlag{
					Name: "global, g",
					Usage: "Search for existing containers globally, not only ones started with compose",
				},
				cli.BoolFlag{
					Name: "force, f",
					Usage: "Force recreation of current configuration",
				},
			},
		},
	}
	app.Run(os.Args)
}

func initLogs(ctx *cli.Context){
	if ctx.GlobalBool("verbose") {
		log.SetLevel(log.DebugLevel)
	}

	if logFilename, err := toAbsolutePath(ctx.GlobalString("log"), false); err != nil {
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
}

func run(ctx *cli.Context) {
	initLogs(ctx)

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

		dockerCfg := compose.DockerClientConfig{
			Host: globalString(ctx, "host"),
		}

		if ctx.GlobalIsSet("tlsverify") {
			dockerCfg.Tlsverify = ctx.Bool("tlsverify")
			dockerCfg.Tlscacert = globalString(ctx, "tlscacert")
			dockerCfg.Tlscert = globalString(ctx, "tlscert")
			dockerCfg.Tlskey = globalString(ctx, "tlskey")
		}

		compose.Run(
			&compose.ComposeConfig{
				Manifest: config,
				DockerCfg: dockerCfg,
				Timeout: ctx.Int("timeout"),
				Global: ctx.Bool("global"),
				Force: ctx.Bool("force"),
			})
	}
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
func globalString(c *cli.Context, name string) string {
	str := c.GlobalString(name)
	if len(str) >= 2 && str[0] == '\u0022' && str[len(str)-1] == '\u0022' {
		str = str[1 : len(str)-1]
	}
	return str
}
