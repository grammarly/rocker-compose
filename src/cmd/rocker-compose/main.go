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
	"compose"
	"compose/ansible"
	"compose/config"
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/fsouza/go-dockerclient"
	"github.com/kr/pretty"
)

var (
	Version   = "built locally"
	GitCommit = "none"
	GitBranch = "none"
	BuildTime = "none"
)

func init() {
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)
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
		cli.BoolFlag{
			Name:  "dry, d",
			Usage: "Don't execute any run/stop operations on target docker",
		},
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
			Action: runCommand,
			Flags: append([]cli.Flag{
				cli.BoolFlag{
					Name:  "global, g",
					Usage: "Search for existing containers globally, not only ones started with compose",
				},
				cli.BoolFlag{
					Name:  "force",
					Usage: "Force recreation of current configuration",
				},
				cli.BoolFlag{
					Name:  "upgrade, u",
					Usage: "Force recreation of containers which have greater version",
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
		{
			Name:   "info",
			Usage:  "show docker info (check connectivity, versions, etc.)",
			Action: infoCommand,
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name:  "all, a",
					Usage: "show advanced info",
				},
			},
		},
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

	compose, err := compose.New(&compose.ComposeConfig{
		Manifest: config,
		Docker:   dockerCli,
		Global:   ctx.Bool("global"),
		Force:    ctx.Bool("force"),
		DryRun:   ctx.Bool("dry"),
		Attach:   ctx.Bool("attach"),
		Wait:     ctx.Duration("wait"),
		Pull:     ctx.Bool("pull"),
		Upgrade:  ctx.Bool("upgrade"),
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

	compose, err := compose.New(&compose.ComposeConfig{
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

	compose, err := compose.New(&compose.ComposeConfig{
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

func recoverCommand(ctx *cli.Context) {
	initLogs(ctx)

	dockerCli := initDockerClient(ctx)
	auth := initAuthConfig(ctx)

	compose, err := compose.New(&compose.ComposeConfig{
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

func infoCommand(ctx *cli.Context) {
	dockerCfg := initDockerConfig(ctx)

	log.Printf("Rocker-compose %s", Version)

	log.Printf("Docker host: %s", dockerCfg.Host)
	log.Printf("Docker use TLS: %s", strconv.FormatBool(dockerCfg.Tlsverify))
	if dockerCfg.Tlsverify {
		log.Printf("  TLS CA cert: %s", dockerCfg.Tlscacert)
		log.Printf("  TLS cert: %s", dockerCfg.Tlscert)
		log.Printf("  TLS key: %s", dockerCfg.Tlskey)
	}

	dockerClient := initDockerClient(ctx)

	// TODO: golang randomizes maps every time, so the output is not consistent
	//       find out a way to sort it correctly

	version, err := dockerClient.Version()
	if err != nil {
		log.Fatal(err)
	}

	for _, kv := range *version {
		parts := strings.SplitN(kv, "=", 2)
		log.Printf("Docker %s: %s", parts[0], parts[1])
	}

	if ctx.Bool("all") {
		info, err := dockerClient.Info()
		if err != nil {
			log.Fatal(err)
		}

		log.Printf("Docker advanced info:")
		for key, val := range info.Map() {
			log.Printf("  %s: %s", key, val)
		}
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

func initComposeConfig(ctx *cli.Context, dockerCli *docker.Client) *config.Config {
	file := ctx.String("file")

	if file == "" {
		log.Fatalf("Manifest file is empty")
		os.Exit(1)
	}

	vars := pairsFromStrings(ctx.StringSlice("var"), "=")

	var bridgeIp *string

	// TODO: find better place for providing this helper
	funcs := map[string]interface{}{
		// lazy get bridge ip
		"bridgeIp": func() (ip string, err error) {
			if bridgeIp == nil {
				ip, err = compose.GetBridgeIp(dockerCli)
				if err != nil {
					return "", err
				}
				bridgeIp = &ip
			}
			return *bridgeIp, nil
		},
	}

	var (
		manifest *config.Config
		err      error
	)
	if file == "-" {
		log.Infof("Reading manifest from STDIN")
		manifest, err = config.ReadConfig(file, os.Stdin, vars, funcs)
	} else {
		log.Infof("Reading manifest: %s", file)
		manifest, err = config.NewFromFile(file, vars, funcs)
	}

	if err != nil {
		log.Fatal(err)
	}

	return manifest
}

func initDockerConfig(ctx *cli.Context) *compose.DockerClientConfig {
	dockerCfg := compose.NewDockerClientConfig()
	dockerCfg.Host = globalString(ctx, "host")

	if ctx.GlobalIsSet("tlsverify") {
		dockerCfg.Tlsverify = ctx.GlobalBool("tlsverify")
		dockerCfg.Tlscacert = globalString(ctx, "tlscacert")
		dockerCfg.Tlscert = globalString(ctx, "tlscert")
		dockerCfg.Tlskey = globalString(ctx, "tlskey")
	}

	return dockerCfg
}

func initDockerClient(ctx *cli.Context) *docker.Client {
	dockerCfg := initDockerConfig(ctx)

	cli, err := compose.NewDockerClientFromConfig(dockerCfg)
	if err != nil {
		log.Fatalf("Docker client initialization failed with error '%s' and config:\n%# v", err, pretty.Formatter(dockerCfg))
	}

	return cli
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

func doRemove(ctx *cli.Context, config *config.Config, dockerCli *docker.Client, auth *compose.AuthConfig) error {
	compose, err := compose.New(&compose.ComposeConfig{
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

func pairsFromStrings(pairs []string, sep string) map[string]interface{} {
	vars := map[string]interface{}{}
	for _, varPair := range pairs {
		tmp := strings.SplitN(varPair, sep, 2)
		if len(tmp) == 2 {
			vars[tmp[0]] = tmp[1]
		} else {
			vars[tmp[0]] = ""
		}
	}
	return vars
}
