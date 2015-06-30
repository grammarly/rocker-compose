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
* [ ] client.go execution functions
* [ ] Add labels for containers launched by compose?
* [ ] rocker-compose executable with docker connection and cli flags
* [ ] ansible-module mode for rocker-compose executable
* [X] Choose and adopt logging framework
* [ ] Attach stdout of launched (or existing) containers
* [ ] Force-restart option
* [ ] Never remove volumes of some containers
* [ ] Parallel pull operation
* [ ] Force-pull option (if image is existing, only for "latest" or non-semver tags?)
* [ ] Clean command, keep_versions config attribute for containers
* [ ] Dry mode
* [ ] Cross-compilation for linux and darwin (run in container? how gb will work?)
* [X] Protect from looped dependencies
