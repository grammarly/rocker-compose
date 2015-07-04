# rocker-compose

Composition tool for running multiple containers on a host.

TODO: usage manual

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
* [ ] Dry mode, todo: ensure dry works for all actions
* [ ] Clean command, keep_versions config attribute for containers
* [ ] ansible-module mode for rocker-compose executable
