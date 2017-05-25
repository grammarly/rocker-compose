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
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-yaml/yaml"
	"github.com/snkozlov/rocker-compose/src/compose"
	"github.com/snkozlov/rocker-compose/src/compose/ansible"
	"github.com/snkozlov/rocker-compose/src/compose/config"
	"github.com/snkozlov/rocker-compose/src/compose/tarmaker"

	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/fsouza/go-dockerclient"
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

	fileArg := cli.StringFlag{
		Name:  "file, f",
		Value: "compose.yml",
		Usage: "Path to configuration file which should be run, if `-` is given as a value, then STDIN will be used",
	}

	varsFlags := []cli.Flag{
		cli.StringSliceFlag{
			Name:  "var",
			Value: &cli.StringSlice{},
			Usage: "Set variables to pass to build tasks, value is like \"key=value\"",
		},
		cli.StringSliceFlag{
			Name:  "var-file",
			Value: &cli.StringSlice{},
			Usage: "Load variables form a file, either JSON or YAML. Can pass multiple of this.",
		},
	}

	composeFlags := appendFlags(fileArg, varsFlags, []cli.Flag{
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
		cli.BoolFlag{
			Name:  "tar",
			Usage: "the input compose file is a release tar archive (see 'tar' command)",
		},
	})

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
		cli.DurationFlag{
			Name:  "docker-ping-timeout",
			Value: 2 * time.Second,
			Usage: "Timeout for docker to send a response to ping during initialization",
		},
		cli.IntFlag{
			Name:  "docker-ping-retries",
			Value: 5,
			Usage: "Number of retries when checking docker during initialization of docker client. Sleep between retries: 1s",
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
			Name:   "tar",
			Usage:  "make a tar release including artifacts that can then be executed instead of compose.yml",
			Action: tarCommand,
			Flags: appendFlags(fileArg, varsFlags, []cli.Flag{
				cli.StringFlag{
					Name:  "output, O",
					Value: "-",
					Usage: "write result in a file or stdout if the value is `-`",
				},
				cli.StringFlag{
					Name:  "prefix, P",
					Value: "release/",
					Usage: "specify a prefix directory inside tar archive, can be only one level prefix",
				},
			}),
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
		fmt.Println(err.Error())
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

func tarCommand(ctx *cli.Context) {
	initLogs(ctx)

	var (
		file   = ctx.String("file")
		output = ctx.String("output")
		prefix = ctx.String("prefix")
	)

	// TODO: test logs
	if output == "-" && !ctx.GlobalIsSet("verbose") {
		log.SetLevel(log.WarnLevel)
	}

	vars := initVars(ctx)

	options := tarmaker.MakeTarOptions{
		File:   file,
		Output: output,
		Prefix: prefix,
		Vars:   vars,
	}

	if err := tarmaker.MakeTar(options); err != nil {
		log.Fatalln(err)
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
		fd       io.Reader = os.Stdin
		isTar              = ctx.Bool("tar")
		print              = ctx.Bool("print")
	)

	vars := initVars(ctx)

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
                "getenv": func(key string) (ip string, err error) {
                       return os.Getenv(key), nil
                },
	}

	if file != "-" {
		if !print {
			log.Infof("Reading manifest: %s", file)
		}

		if !path.IsAbs(file) {
			wd, err := os.Getwd()
			if err != nil {
				log.Fatalf("Cannot get absolute path to %s due to error %s", file, err)
			}
			file = path.Join(wd, file)
		}

		// Also detect tar input by extension
		if filepath.Ext(file) == ".tar" {
			isTar = true
		}

		if fd, err = os.Open(file); err != nil {
			log.Fatal(err)
		}
		defer fd.(io.ReadCloser).Close()
	} else {
		if !print {
			log.Infof("Reading manifest from STDIN")
		}
	}

	if isTar {
		tr := tar.NewReader(fd)
		var composePrefix *string

		varsByPrefix := map[string]template.Vars{}

		for {
			hdr, err := tr.Next()
			if err == io.EOF {
				// end of tar archive
				break
			}
			if err != nil {
				log.Fatal(err)
			}

			if composePrefix == nil && filepath.Base(hdr.Name) == "compose.yml" {
				fd = new(bytes.Buffer)
				if _, err := io.Copy(fd.(io.Writer), tr); err != nil {
					log.Fatal(err)
				}
				pref := filepath.Dir(hdr.Name)
				composePrefix = &pref
				continue
			}

			// read variables from variables.yml
			if filepath.Base(hdr.Name) == "variables.yml" {
				var (
					pref  = filepath.Dir(hdr.Name)
					data  []byte
					fvars template.Vars
				)

				if data, err = ioutil.ReadAll(tr); err != nil {
					log.Fatal(err)
				}

				if err := yaml.Unmarshal(data, &fvars); err != nil {
					log.Fatal(err)
				}

				varsByPrefix[pref] = fvars
			}

		}

		if composePrefix == nil {
			log.Fatal("Cannot find compose.yml file inside tar archive. It may be corrupt. Test it with `tar -t`.")
		}

		if prefixVars, ok := varsByPrefix[*composePrefix]; ok {
			vars = template.Vars{}.Merge(prefixVars, vars)
		}
	}

	manifest, err = config.ReadConfig(file, fd, vars, funcs, print)
	if err != nil {
		log.Fatal(err)
	}

	// Timeout for docker daemon to respond after accepting connection
	dockerCli.SetTimeout(ctx.GlobalDuration("docker-ping-timeout"))
	defer dockerCli.SetTimeout(0 * time.Second)

	max := ctx.GlobalInt("docker-ping-retries")
	for i := 1; i <= max; i++ {
		var err error

		if err = dockerCli.Ping(); err == nil {
			return manifest
		}

		log.Infof("Error connecting to docker endpoint %s, attempt %d/%d, error: %s", dockerCli.Endpoint(), i, max, err)
		time.Sleep(1 * time.Second)
	}

	log.Fatalf("Unable to connect to docker endpoint %s", dockerCli.Endpoint())
	os.Exit(1)

	return manifest
}

func initVars(c *cli.Context) template.Vars {
	vars, err := template.VarsFromFileMulti(c.StringSlice("var-file"))
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	cliVars, err := template.VarsFromStrings(c.StringSlice("var"))
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	vars = vars.Merge(cliVars)
	return vars
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

func appendFlags(flags ...interface{}) []cli.Flag {
	result := []cli.Flag{}
	for _, f := range flags {
		switch flag := f.(type) {
		case cli.Flag:
			result = append(result, flag)
		case []cli.Flag:
			result = append(result, flag...)
		}
	}
	return result
}
