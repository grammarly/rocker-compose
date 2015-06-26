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

[ ] Should remove obsolete containers (e.g. removed from compose.yml)
[ ] EnsureContainer for containers out of namespace (cannot be created)
[ ] Introduce templating for compose.yml to substitute variables from the outside
[ ] client.go execution functions
[ ] Refactor config.go - move some functions to config_convert.go
[ ] Add labels for containers launched by compose?
