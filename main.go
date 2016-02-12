/*-
 * Copyright 2015 Grammarly, Inc.
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

// Main rocker-compose executable
// type rocker-compose --help for more info
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/grammarly/rocker-compose/src/compose"
	"github.com/grammarly/rocker-compose/src/compose/ansible"
	"github.com/grammarly/rocker-compose/src/compose/config"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/fsouza/go-dockerclient"
	"github.com/go-yaml/yaml"
	"github.com/grammarly/rocker/src/dockerclient"
	"github.com/grammarly/rocker/src/rocker/debugtrap"
	"github.com/grammarly/rocker/src/rocker/textformatter"
	"github.com/grammarly/rocker/src/template"
)

var (
	// Version that is passed on compile time through -ldflags
	Version = "built locally"

	// GitCommit that is passed on compile time through -ldflags
	GitCommit = "none"

	// GitBranch that is passed on compile time through -ldflags
	GitBranch = "none"

	// BuildTime that is passed on compile time through -ldflags
	BuildTime = "none"
)

func init() {
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)
	debugtrap.SetupDumpStackTrap()
}

func main() {
	app := cli.NewApp()

	app.Name = "rocker-compose"
	app.Version = fmt.Sprintf("%s - %.7s (%s) %s", Version, GitCommit, GitBranch, BuildTime)
	app.Usage = "Tool for docker orchestration"
	app.Authors = []cli.Author{
		{"Yura Bogdanov", "yuriy.bogdanov@grammarly.com"},
		{"Stas Levental", "stas.levental@grammarly.com"},
	}

	composeFlags := []cli.Flag{
		cli.StringFlag{
			Name:  "file, f",
			Value: "compose.yml",
			Usage: "Path to configuration file which should be run, if `-` is given as a value, then STDIN will be used",
		},
		cli.StringSliceFlag{
			Name:  "var",
			Value: &cli.StringSlice{},
			Usage: "Set variables to pass to build tasks, value is like \"key=value\"",
		},
		cli.StringSliceFlag{
			Name:  "vars",
			Value: &cli.StringSlice{},
			Usage: "Load variables form a file, either JSON or YAML. Can pass multiple of this.",
		},
		cli.BoolFlag{
			Name:  "dry, d",
			Usage: "Don't execute any run/stop operations on target docker",
		},
		cli.BoolFlag{
			Name:  "print",
			Usage: "just print the rendered compose config and exit",
		},
		cli.BoolFlag{
			Name:  "demand-artifacts",
			Usage: "fail if artifacts not found for {{ image }} helpers",
		},
	}

	app.Flags = append([]cli.Flag{
		cli.BoolFlag{
			Name: "verbose, vv, D",
		},
		cli.StringFlag{
			Name: "log, l",
		},
		cli.BoolFlag{
			Name: "json",
		},
		cli.StringFlag{
			Name:  "auth, a",
			Value: "",
			Usage: "Docker auth, username and password in user:password format",
		},
		cli.BoolTFlag{
			Name: "colors",
		},
	}, dockerclient.GlobalCliParams()...)

	app.Commands = []cli.Command{
		{
			Name:   "run",
			Usage:  "execute manifest",
			Action: runCommand,
			Flags: append([]cli.Flag{
				cli.BoolFlag{
					Name:  "force",
					Usage: "Force recreation of current configuration",
				},
				cli.BoolFlag{
					Name:  "attach",
					Usage: "Stream stdout of all containers to log",
				},
				cli.BoolFlag{
					Name:  "pull",
					Usage: "Do pull images before running",
				},
				cli.DurationFlag{
					Name:  "wait",
					Value: 1 * time.Second,
					Usage: "Wait and check exit codes of launched containers",
				},
				cli.BoolFlag{
					Name:  "ansible",
					Usage: "output json in ansible format for easy parsing",
				},
			}, composeFlags...),
		},
		{
			Name:   "pull",
			Usage:  "pull images specified in the manifest",
			Action: pullCommand,
			Flags: append([]cli.Flag{
				cli.BoolFlag{
					Name:  "ansible",
					Usage: "output json in ansible format for easy parsing",
				},
			}, composeFlags...),
		},
		{
			Name:   "rm",
			Usage:  "stop and remove any containers specified in the manifest",
			Action: rmCommand,
			Flags:  composeFlags,
		},
		{
			Name:   "clean",
			Usage:  "cleanup old tags for images specified in the manifest",
			Action: cleanCommand,
			Flags: append([]cli.Flag{
				cli.IntFlag{
					Name:  "keep, k",
					Value: 5,
					Usage: "number of last images to keep",
				},
				cli.BoolFlag{
					Name:  "ansible",
					Usage: "output json in ansible format for easy parsing",
				},
			}, composeFlags...),
		},
		{
			Name:   "pin",
			Usage:  "pin versions",
			Action: pinCommand,
			Flags: append([]cli.Flag{
				cli.BoolTFlag{
					Name:  "local, l",
					Usage: "search across images available locally",
				},
				cli.BoolTFlag{
					Name:  "hub",
					Usage: "search across images in the registry",
				},
				cli.StringFlag{
					Name:  "type, t",
					Value: "yaml",
					Usage: "output in specified format: json|yaml",
				},
				cli.StringFlag{
					Name:  "output, O",
					Value: "-",
					Usage: "write result in a file or stdout if the value is `-`",
				},
			}, composeFlags...),
		},
		{
			Name:   "recover",
			Usage:  "recover containers from machine reboot or docker daemon restart",
			Action: recoverCommand,
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name:  "dry, d",
					Usage: "Don't execute any run/stop operations on target docker",
				},
				cli.DurationFlag{
					Name:  "wait",
					Value: 1 * time.Second,
					Usage: "Wait and check exit codes of launched containers",
				},
			},
		},
		dockerclient.InfoCommandSpec(),
	}

	app.CommandNotFound = func(ctx *cli.Context, command string) {
		fmt.Printf("Command not found: %v\n", command)
		os.Exit(1)
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Printf(err.Error())
		os.Exit(1)
	}
}

func runCommand(ctx *cli.Context) {
	ansibleResp := initAnsubleResp(ctx)

	// TODO: here we duplicate fatalf in both run(), pull() and clean()
	// maybe refactor to make it cleaner
	fatalf := func(err error) {
		if ansibleResp != nil {
			ansibleResp.Error(err).WriteTo(os.Stdout)
		}
		log.Fatal(err)
	}

	initLogs(ctx)

	dockerCli := initDockerClient(ctx)
	config := initComposeConfig(ctx, dockerCli)
	auth := initAuthConfig(ctx)

	compose, err := compose.New(&compose.Config{
		Manifest: config,
		Docker:   dockerCli,
		Force:    ctx.Bool("force"),
		DryRun:   ctx.Bool("dry"),
		Attach:   ctx.Bool("attach"),
		Wait:     ctx.Duration("wait"),
		Pull:     ctx.Bool("pull"),
		Auth:     auth,
	})

	if err != nil {
		fatalf(err)
	}

	// in case of --force given, first remove all existing containers
	if ctx.Bool("force") {
		if err := doRemove(ctx, config, dockerCli, auth); err != nil {
			fatalf(err)
		}
	}

	if err := compose.RunAction(); err != nil {
		fatalf(err)
	}

	if ansibleResp != nil {
		// ansibleResp.Success("done hehe").WriteTo(os.Stdout)
		compose.WritePlan(ansibleResp).WriteTo(os.Stdout)
	}
}

func pullCommand(ctx *cli.Context) {
	ansibleResp := initAnsubleResp(ctx)

	fatalf := func(err error) {
		if ansibleResp != nil {
			ansibleResp.Error(err).WriteTo(os.Stdout)
		}
		log.Fatal(err)
	}

	initLogs(ctx)

	dockerCli := initDockerClient(ctx)
	config := initComposeConfig(ctx, dockerCli)
	auth := initAuthConfig(ctx)

	compose, err := compose.New(&compose.Config{
		Manifest: config,
		Docker:   dockerCli,
		DryRun:   ctx.Bool("dry"),
		Auth:     auth,
	})
	if err != nil {
		fatalf(err)
	}

	if err := compose.PullAction(); err != nil {
		fatalf(err)
	}

	if ansibleResp != nil {
		// ansibleResp.Success("done hehe").WriteTo(os.Stdout)
		compose.WritePlan(ansibleResp).WriteTo(os.Stdout)
	}
}

func rmCommand(ctx *cli.Context) {
	initLogs(ctx)

	dockerCli := initDockerClient(ctx)
	config := initComposeConfig(ctx, dockerCli)
	auth := initAuthConfig(ctx)

	if err := doRemove(ctx, config, dockerCli, auth); err != nil {
		log.Fatal(err)
	}
}

func cleanCommand(ctx *cli.Context) {
	ansibleResp := initAnsubleResp(ctx)

	fatalf := func(err error) {
		if ansibleResp != nil {
			ansibleResp.Error(err).WriteTo(os.Stdout)
		}
		log.Fatal(err)
	}

	initLogs(ctx)

	dockerCli := initDockerClient(ctx)
	config := initComposeConfig(ctx, dockerCli)
	auth := initAuthConfig(ctx)

	compose, err := compose.New(&compose.Config{
		Manifest:   config,
		Docker:     dockerCli,
		DryRun:     ctx.Bool("dry"),
		Remove:     true,
		Auth:       auth,
		KeepImages: ctx.Int("keep"),
	})
	if err != nil {
		fatalf(err)
	}

	if err := compose.CleanAction(); err != nil {
		fatalf(err)
	}

	if ansibleResp != nil {
		// ansibleResp.Success("done hehe").WriteTo(os.Stdout)
		compose.WritePlan(ansibleResp).WriteTo(os.Stdout)
	}
}

func pinCommand(ctx *cli.Context) {
	initLogs(ctx)

	var (
		vars   template.Vars
		data   []byte
		output = ctx.String("output")
		format = ctx.String("type")
		local  = ctx.BoolT("local")
		hub    = ctx.BoolT("hub")
		fd     = os.Stdout
	)

	if output == "-" && !ctx.GlobalIsSet("verbose") {
		log.SetLevel(log.WarnLevel)
	}

	dockerCli := initDockerClient(ctx)
	config := initComposeConfig(ctx, dockerCli)
	auth := initAuthConfig(ctx)

	compose, err := compose.New(&compose.Config{
		Manifest: config,
		Docker:   dockerCli,
		Auth:     auth,
	})
	if err != nil {
		log.Fatal(err)
	}

	if vars, err = compose.PinAction(local, hub); err != nil {
		log.Fatal(err)
	}

	if output != "-" {
		if fd, err = os.Create(output); err != nil {
			log.Fatal(err)
		}
		defer fd.Close()

		if ext := filepath.Ext(output); !ctx.IsSet("type") && ext == ".json" {
			format = "json"
		}
	}

	switch format {
	case "yaml":
		if data, err = yaml.Marshal(vars); err != nil {
			log.Fatal(err)
		}
	case "json":
		if data, err = json.Marshal(vars); err != nil {
			log.Fatal(err)
		}
	default:
		log.Fatalf("Possible tyoes are `yaml` and `json`, unknown type `%s`", format)
	}

	if _, err := io.Copy(fd, bytes.NewReader(data)); err != nil {
		log.Fatal(err)
	}
}

func recoverCommand(ctx *cli.Context) {
	initLogs(ctx)

	dockerCli := initDockerClient(ctx)
	auth := initAuthConfig(ctx)

	compose, err := compose.New(&compose.Config{
		Docker:  dockerCli,
		DryRun:  ctx.Bool("dry"),
		Wait:    ctx.Duration("wait"),
		Recover: true,
		Auth:    auth,
	})

	if err != nil {
		log.Fatal(err)
	}

	if err := compose.RecoverAction(); err != nil {
		log.Fatal(err)
	}
}

func initLogs(ctx *cli.Context) {
	logger := log.StandardLogger()

	if ctx.GlobalBool("verbose") {
		logger.Level = log.DebugLevel
	} else if ctx.Bool("print") && ctx.GlobalString("log") == "" {
		logger.Level = log.ErrorLevel
	}

	var (
		err       error
		isTerm    = log.IsTerminal()
		logFile   = ctx.GlobalString("log")
		logExt    = path.Ext(logFile)
		json      = ctx.GlobalBool("json") || logExt == ".json"
		useColors = isTerm && !json && logFile == ""
	)

	if ctx.GlobalIsSet("colors") {
		useColors = ctx.GlobalBool("colors")
	}

	if logFile != "" {
		if logFile, err = toAbsolutePath(logFile, false); err != nil {
			log.Fatal(err)
		}
		if logger.Out, err = os.OpenFile(logFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644); err != nil {
			log.Fatalf("Initializing log: Cannot initialize log file %s due to error %s", logFile, err)
		}
		log.Debugf("Initializing log: Successfuly started loggin to '%s'", logFile)
	}

	if json {
		logger.Formatter = &log.JSONFormatter{}
	} else {
		formatter := &textformatter.TextFormatter{}
		formatter.DisableColors = !useColors
		logger.Formatter = formatter
	}
}

func initComposeConfig(ctx *cli.Context, dockerCli *docker.Client) *config.Config {
	file := ctx.String("file")

	if file == "" {
		log.Fatalf("Manifest file is empty")
		os.Exit(1)
	}

	var (
		manifest *config.Config
		err      error
		bridgeIP *string
		print    = ctx.Bool("print")
	)

	vars, err := template.VarsFromFileMulti(ctx.StringSlice("vars"))
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	cliVars, err := template.VarsFromStrings(ctx.StringSlice("var"))
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	vars = vars.Merge(cliVars)

	if ctx.Bool("demand-artifacts") {
		vars["DemandArtifacts"] = true
	}

	// TODO: find better place for providing this helper
	funcs := map[string]interface{}{
		// lazy get bridge ip
		"bridgeIp": func() (ip string, err error) {
			if bridgeIP == nil {
				ip, err = compose.GetBridgeIP(dockerCli)
				if err != nil {
					return "", err
				}
				bridgeIP = &ip
			}
			return *bridgeIP, nil
		},
	}

	if file == "-" {
		if !print {
			log.Infof("Reading manifest from STDIN")
		}
		manifest, err = config.ReadConfig(file, os.Stdin, vars, funcs, print)
	} else {
		if !print {
			log.Infof("Reading manifest: %s", file)
		}
		manifest, err = config.NewFromFile(file, vars, funcs, print)
	}

	if err != nil {
		log.Fatal(err)
	}

	// Check the docker connection before we actually run
	if err := dockerclient.Ping(dockerCli, 5000); err != nil {
		log.Fatal(err)
	}

	return manifest
}

func initDockerClient(ctx *cli.Context) *docker.Client {
	dockerClient, err := dockerclient.NewFromCli(ctx)
	if err != nil {
		log.Fatal(err)
	}
	return dockerClient
}

func initAuthConfig(c *cli.Context) (auth *docker.AuthConfigurations) {
	var err error
	if c.GlobalIsSet("auth") {
		// Obtain auth configuration from cli params
		authParam := c.GlobalString("auth")
		if strings.Contains(authParam, ":") {
			userPass := strings.Split(authParam, ":")
			auth = &docker.AuthConfigurations{
				Configs: map[string]docker.AuthConfiguration{
					"*": docker.AuthConfiguration{
						Username: userPass[0],
						Password: userPass[1],
					},
				},
			}
		}
		return
	}
	// Obtain auth configuration from .docker/config.json
	if auth, err = docker.NewAuthConfigurationsFromDockerCfg(); err != nil && !os.IsNotExist(err) {
		log.Fatal(err)
	}
	return
}

func initAnsubleResp(ctx *cli.Context) (ansibleResp *ansible.Response) {
	if ctx.Bool("ansible") {
		ansibleResp = &ansible.Response{}

		if !ctx.GlobalIsSet("log") {
			ansibleResp.Error(fmt.Errorf("--log param should be provided for ansible mode")).WriteTo(os.Stdout)
			os.Exit(1)
		}
	}
	return
}

func doRemove(ctx *cli.Context, config *config.Config, dockerCli *docker.Client, auth *docker.AuthConfigurations) error {
	compose, err := compose.New(&compose.Config{
		Manifest: config,
		Docker:   dockerCli,
		DryRun:   ctx.Bool("dry"),
		Remove:   true,
		Auth:     auth,
	})
	if err != nil {
		return err
	}
	return compose.RunAction()
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

// globalString fixes string arguments enclosed with double quotes
// 'docker-machine config' gives such arguments
func globalString(c *cli.Context, name string) string {
	str := c.GlobalString(name)
	if len(str) >= 2 && str[0] == '\u0022' && str[len(str)-1] == '\u0022' {
		str = str[1 : len(str)-1]
	}
	return str
}
