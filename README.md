# rocker-compose

Composition tool for running multiple Docker containers on any machine. It's intended to be used in a following cases:

1. Deploying containerized apps to servers
2. Running containerized apps locally for development or testing

There is an official [docker-compose](https://github.com/docker/compose) tool which is made exactly for the same purpose. But we found that it is missing a few key features that makes us unable using it for deployments. For us, composition tool should:

1. Be able to read the manifest (configuration file) and run an isolated chain of containers
2. Support all Docker's configuration options, such as all you can do with plain `docker run`, you can do with `compose` *for some options, docker-compose do not have the most intuitive names, rocker-compose uses convention to fit the names of original run spec*
3. Be idempotent. Only affected containers should be restarted. *docker-compose simply restarts everything on every run* 
4. Support configurable namespaces and avoid name clashes between apps *docker-compose does not even support underscores in container names, that's a bummer*
5. Remove containers that are not in the manifest anymore *docker-compose does not*
6. Respect any changes that can be made to containers configuration. Any change should effect in a container restart. Images can be updated, their names might stay same, in cases of using `:latest` tags
7. Dependency graph cat also define which actions can run in parallel, utilize it
8. Support templating in the manifest file. Not just putting ENV variables, but also be able to do conditionals, etc. *docker-compose does not have it, but they recently came up with a [pretty good solution](https://github.com/docker/compose/issues/1377), which we may adopt soon as well*

Contributing these features to docker-compose was also an option, but we decided to come up with own solution due the following reasons:

1. docker-machine is written in Python, we don't have tools in Python
2. We have a full control over the tool and can add any feature to it any time
3. The tool should be written in Go to benefit from the existing ecosystem, also it is easier to install it on a development machine or on any instance or CI server
4. Time factor was also critical, we were able to come up with a working solution in a four days

# Examples

IN PROGRESS

# compose.yml spec

IN PROGRESS

# Contributing

### Dependencies

Use [gb](http://getgb.io/) to test and build.

### Fetch dependencies

```bash
gb vendor update -all
```

### Build

```bash
gb build
```

### Test 

```bash
gb test
```

### Test something particular

```bash
gb test -run TestMyFunction
```

### TODO

* [x] Introduce templating for compose.yml to substitute variables from the outside
* [x] Refactor config.go - move some functions to config_convert.go
* [X] Should remove obsolete containers (e.g. removed from compose.yml)
* [X] EnsureContainer for containers out of namespace (cannot be created)
* [X] client.go execution functions
* [X] Add labels for containers launched by compose?
* [X] rocker-compose executable with docker connection and cli flags
* [X] Choose and adopt logging framework
* [X] Protect from looped dependencies
* [X] Cross-compilation for linux and darwin (run in container? how gb will work?)
* [x] Attach stdout of launched (or existing) containers
* [x] Force-restart option
* [x] Never remove volumes of some containers
* [x] Parallel pull operation
* [x] Force-pull option (if image is existing, only for "latest" or non-semver tags?)
* [x] Clean command, keep_versions config attribute for containers
* [x] ansible-module mode for rocker-compose executable
* [ ] Write detailed readme, manual and tutorial
* [ ] Dry mode, todo: ensure dry works for all actions
