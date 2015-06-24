# rocker-compose

Composition tool for running multiple containers on a host.

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

