package main

import (
	"compose"
	"fmt"
	"os"
	"path"
	"strings"
	"time"

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
		cli.BoolFlag{
			Name: "verbose, vv",
		},
		cli.StringFlag{
			Name: "log, l",
		},
		cli.BoolFlag{
			Name: "json",
		},
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
			Name:  "auth, a",
			Value: "",
			Usage: "Docker auth, username and password in user:password format",
		},
	}

	app.Commands = []cli.Command{
		{
			Name:   "run",
			Usage:  "execute manifest",
			Action: run,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "file, f",
					Usage: "Path to configuration file which should be run",
				},
				cli.BoolFlag{
					Name:  "global, g",
					Usage: "Search for existing containers globally, not only ones started with compose",
				},
				cli.BoolFlag{
					Name:  "force",
					Usage: "Force recreation of current configuration",
				},
				cli.BoolFlag{
					Name:  "dry, d",
					Usage: "Don't execute any run/stop operations on target docker",
				},
				cli.BoolFlag{
					Name:  "attach",
					Usage: "Stream stdout of all containers to log",
				},
				cli.DurationFlag{
					Name:  "wait",
					Value: 1 * time.Second,
					Usage: "Wait and check exit codes of launched containers",
				},
				cli.StringSliceFlag{
					Name:  "var",
					Value: &cli.StringSlice{},
					Usage: "set variables to pass to build tasks, value is like \"key=value\"",
				},
			},
		},
		{
			Name:   "pull",
			Usage:  "pull images specified in the manifest",
			Action: pull,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "file, f",
					Usage: "Path to configuration file which should be run",
				},
				cli.BoolFlag{
					Name:  "dry, d",
					Usage: "Don't execute any run/stop operations on target docker",
				},
				cli.StringSliceFlag{
					Name:  "var",
					Value: &cli.StringSlice{},
					Usage: "set variables to pass to build tasks, value is like \"key=value\"",
				},
			},
		},
	}
	app.Run(os.Args)
}

func run(ctx *cli.Context) {
	initLogs(ctx)

	config := initComposeConfig(ctx)
	dockerCfg := initDockerConfig(ctx)
	auth := initAuthConfig(ctx)

	compose, err := compose.New(&compose.ComposeConfig{
		Manifest:  config,
		DockerCfg: dockerCfg,
		Global:    ctx.Bool("global"),
		Force:     ctx.Bool("force"),
		DryRun:    ctx.Bool("dry"),
		Attach:    ctx.Bool("attach"),
		Wait:      ctx.Duration("wait"),
		Auth:      auth,
	})

	if err != nil {
		log.Fatal(err)
	}

	if err := compose.Run(); err != nil {
		log.Fatal(err)
	}
}

func pull(ctx *cli.Context) {
	initLogs(ctx)

	config := initComposeConfig(ctx)
	dockerCfg := initDockerConfig(ctx)
	auth := initAuthConfig(ctx)

	compose, err := compose.New(&compose.ComposeConfig{
		Manifest:  config,
		DockerCfg: dockerCfg,
		DryRun:    ctx.Bool("dry"),
		Auth:      auth,
	})
	if err != nil {
		log.Fatal(err)
	}

	if err := compose.Pull(); err != nil {
		log.Fatal(err)
	}
}

func initLogs(ctx *cli.Context) {
	if ctx.GlobalBool("verbose") {
		log.SetLevel(log.DebugLevel)
	}

	if ctx.GlobalBool("json") {
		log.SetFormatter(&log.JSONFormatter{})
	}

	logFilename, err := toAbsolutePath(ctx.GlobalString("log"), false)
	if err != nil {
		log.Debugf("Initializing log: Skipped, because Log %s", err)
		return
	}

	logFile, err := os.OpenFile(logFilename, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0755)
	if err != nil {
		log.Warnf("Initializing log: Cannot initialize log file %s due to error %s", logFilename, err)
		return
	}

	log.SetOutput(logFile)

	if path.Ext(logFilename) == ".json" {
		log.SetFormatter(&log.JSONFormatter{})
	}

	log.Debugf("Initializing log: Successfuly started loggin to '%s'", logFilename)
}


func initComposeConfig(ctx *cli.Context) *compose.Config {
	log.Debugf("Reading manifest: '%s'", ctx.String("file"))

	configFilename, err := toAbsolutePath(ctx.String("file"), true)

	if err != nil {
		log.Fatalf("Cannot read manifest: %s", err)
		os.Exit(1) // no config - no pichenka
	}

	vars := varsFromStrings(ctx.StringSlice("var"))
	log.Infof("Reading manifest: %s", configFilename)
	config, err := compose.ReadConfigFile(configFilename, vars)
	if err != nil {
		log.Fatal(err)
	}

	return config
}

func initDockerConfig(ctx *cli.Context) *compose.DockerClientConfig {
	dockerCfg := compose.NewDockerClientConfig()
	dockerCfg.Host = globalString(ctx, "host")

	if ctx.GlobalIsSet("tlsverify") {
		dockerCfg.Tlsverify = ctx.Bool("tlsverify")
		dockerCfg.Tlscacert = globalString(ctx, "tlscacert")
		dockerCfg.Tlscert = globalString(ctx, "tlscert")
		dockerCfg.Tlskey = globalString(ctx, "tlskey")
	}

	return dockerCfg
}

func initAuthConfig(ctx *cli.Context) *compose.AuthConfig {
	auth := &compose.AuthConfig{}
	authParam := globalString(ctx, "auth")

	if strings.Contains(authParam, ":") {
		userPass := strings.Split(authParam, ":")
		auth.Username = userPass[0]
		auth.Password = userPass[1]
	}
	return auth
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

func varsFromStrings(pairs []string) map[string]interface{} {
	vars := map[string]interface{}{}
	for _, varPair := range pairs {
		tmp := strings.SplitN(varPair, "=", 2)
		if len(tmp) == 2 {
			vars[tmp[0]] = tmp[1]
		} else {
			vars[tmp[0]] = ""
		}
	}
	return vars
}
