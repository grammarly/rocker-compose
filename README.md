# rocker-compose

Docker composition tool with idempotency features for deploying apps composed of multiple containers. It's intended to be used in the following cases:

1. Deploying containerized apps to servers
2. Running containerized apps locally for development or testing

# Table of contents
* [Rationale](#rationale)
* [How it works](#how-it-works)
* [Production use](#production-use)
* [Installation](#installation)
* [Migrating from docker-compose](#migrating-from-docker-compose)
* [Tutorial](#tutorial)
* [Command line reference](#command-line-reference)
* [compose.yml spec](#composeyml-spec)
  * [Types](#types)
  * [Root level properties](#root-level-properties)
  * [Container properties](#container-properties)
* [State](#state)
* [Volumes](#volumes)
  * [Data volume](#data-volume)
  * [Mounted host directory](#mounted-host-directory)
* [Extends](#extends)
* [Templating](#templating)
* [Dynamic scaling](#dynamic-scaling)
* [Patterns](#patterns)
  * [Data volume containers](#data-volume-containers)
  * [Bootstrapping](#bootstrapping)
  * [Loose coupling: network](#loose-coupling-network)
  * [Network share](#network-share)
  * [Loose coupling: files](#loose-coupling-files)
* [Command-line completions](#command-line-completions)
* [Contributing](#contributing)
* [Todo](#todo)
* [Authors](#authors)
* [License](#license)

# Rationale
There is an official [docker-compose](https://github.com/docker/compose) tool that may do the trick. But we found that it is missing a few key features that make us unable to use it for production deployment. `rocker-compose` is designed to be a deployment tool in the first place and be useful for development as a bonus (docker-compose is vice versa). For us, a docker deployment tool should:

1. Be able to read the manifest (configuration file) and run an isolated chain of containers, respecting a dependency graph
2. Be idempotent: only affected containers should be restarted *(docker-compose simply restarts everything on every run)*
3. Support configurable namespaces and avoid name clashes between apps *(docker-compose does not even support underscores in container names - that's a bummer)*
4. Remove containers that are not in the manifest anymore *(docker-compose does not)*
5. Respect any changes that can be made to containers' configuration. Images can be updated, their names might stay the same (in case of using mutable tags)
6. From the dependency graph, we can determine, which actions may run in parallel, and utilize that
7. Support templating in the manifest file: not only ENV variables, but also conditionals, etc. *(docker-compose does not have it, but they recently came up with a [pretty good solution](https://github.com/docker/compose/issues/1377) that we have also adopted)*

Contributing these features to docker-compose was also an option, but we decided to come up with a new solution due the following reasons:

1. docker-compose is written in Python, and we don't have tools in Python. Also it would be nice if the tool was written in Go to benefit from the existing ecosystem and to ease installations on development machines and any instance or CI server
2. We wanted to have full control over the tool and be able to add any feature to it at any time
3. Also, there is [libcompose](https://github.com/docker/libcompose) and it’s a great initiative. However, it’s in an experimental stage.
4. The time factor was also critical; we were able to come up with a working solution in four days

# How it works
The most notable feature of `rocker-compose` is **idempotency**. We have to be able to compare any bit of a container runtime, which includes configuration and state.
For every run, rocker-compose is building two data sets: **desired** and **actual**. "Desired" is the list of containers given in the manifest. "Actual" is the list of currently running containers we get through the Docker API. By [comparing the two sets](/src/compose/diff.go) and knowing the dependencies among the containers we are about to run, we build an [ordered action list](src/compose/action.go). You can also consider the action list as a *delta* between the two sets.

If a desired container does not exist, `rocker-compose` simply creates it (and optionally starts). For an existing container with the same name (namespace does help here), it does a more sophisticated comparison:

1. **Compare configuration.** When starting a container, `rocker-compose` puts the serialized source YAML configuration under a label called `rocker-compose-config`. By [comparing](/src/compose/config/compare.go) the source config from the manifest and the one stored in a running container label, `rocker-compose` can detect changes.
2. **Compare image id**. `rocker-compose` also checks if the image id has changed. It may happen when you are using `:latest` tags, and an image can be updated without changing the tag.
3. [Compare state](#state).

It allows `rocker-compose` to perform **as few changes as possible** to make the actual state match the desired one. If something was changed, `rocker-compose` recreates the container from scratch. Note that any container change can trigger recreations of other containers depending on that one.

**In cases of loose coupling**, you can benefit from a micro-services approach and do clever updates, affecting only a single container, without touching others. See [patterns](#patterns) to learn more about the best practices.

# Production use
rocker-compose isn't yet battle-tested for production. However, it's intended to be used for deployments due to its idempotent properties. Also, anything you do with rocker-compose on your local machine you can do on a remote machine by simply adding remote host parameters or having [appropriate ENV](https://docs.docker.com/reference/commandline/cli/#environment-variables). rocker-compose implements docker's native client interface for connection parameterization.

```bash
$ rocker-compose run                              # gathers info about docker server from ENV
$ rocker-compose $(docker-machine config qa1) run # connects to qa1 server and runs there  
```

*NOTE: You should have qa1 machine registered in your docker-machine*

See [command line reference](#command-line-reference) for more details.

# Installation

### For OSX users

```
brew tap grammarly/tap
brew install grammarly/tap/rocker-compose
```

Ensure that it is built with `go 1.5.x` . If not, make `brew update` before installing `rocker-compose`.

### Manual installation

Go to the [releases](https://github.com/grammarly/rocker-compose/releases) section and download the latest binary for your platform. Then unpack the tar archive and copy the binary somewhere to your path, such as `/usr/local/bin`, and give it executable permissions.

Something like this:
```bash
curl -SL https://github.com/grammarly/rocker-compose/releases/download/0.1.3/rocker-compose-0.1.3_darwin_amd64.tar.gz | tar -xzC /usr/local/bin && chmod +x /usr/local/bin/rocker-compose
```

# Migrating from docker-compose

```
diff docker-compose rocker-compose
```

`rocker-compose` does its best to be compatible with docker-compose manifests, however there are a few differences you should consider in order to migrate:

1. `rocker-compose` does not support image names without tags specified. In case you have images without tags, just add `:latest` explicitly.
2. `rocker-compose` does not support `build` and `dockerfile` properties for the container spec. If you rely on it heavily, please file an issue and describe your use case.
3. Instead of `external_links` property, you can specify a different or empty namespace, e.g. `links: other.app` or `links: .redis`. However, it is suggested to use [loose coupling strategies](#loose-coupling-network) instead.
4. No [Swarm](https://docs.docker.com/swarm/) integration, since we don't use it. It seems to be not a big deal to implement, so PR or issue, please.
5. `rocker-compose` has `restart:always` by default. Despite Docker's default value being "no", we found that more often we want to have "always" and people constantly forget to put it.
6. By default, `rocker-compose` sets `max-file:5 max-size:100m` options for `json-file` log driver. We found that it is much more expected behavior to have log rotation by default.
7. There is no `rocker-compose scale`. Instead, we took a more [declarative approach](#dynamic-scaling) to replicate containers.
8. `extends` works differently: you cannot extend from a different file. [More info](#extends)
9. Other properties that are not supported but may be added easily - file an issue or open a pull request if you miss them: `env_file`, `cap_add`, `devices`, `security_opt`, `stdin_open`, `tty`, `read_only`, `volume_driver`, `mac_address`.

# Tutorial

Here is an [example of running a wordpress application](/example/wordpress.yml) with rocker-compose:
```yaml
namespace: wordpress # specify a manifest-level namespace under which all containers will be named
containers:
  main: # container name will be "wordpress.main"
    image: wordpress:4.1.2 # run from "wordpress" image of version 4.1.2
    links:
      # link container named "db" as alias "mysql", inside the "main" container
      # you can reach "db" container by using "mysql" host or using MYSQL_PORT_3306_TCP_ADDR env var
      - db:mysql
    ports:
      - "8080:80" # throw 8080 port to a host network, map it to 80 internal port

  db:
    image: mysql:5.6
    env:
      MYSQL_ROOT_PASSWORD: example # provide MYSQL_ROOT_PASSWORD env var
    volumes_from:
      # specify to mount all volumes from "db_data" container, this way we can
      # update "db" container without loosing data
      - db_data 

  db_data:
    image: grammarly/scratch:latest # use empty image, just for data
    state: created # this tells compose to not try to run this container, data containers need to be only created
    volumes:
      # define the empty directory that will be used by the "db" container
      - /var/lib/mysql
```

You can run this manifest with the following command:
```bash
rocker-compose run -f example/wordpress.yml
```

Or simply this, in case your manifest is in the same directory and is named `compose.yml`:
```bash
rocker-compose run
```

The output will be something like the following:
```
INFO[0000] Reading manifest: .../rocker-compose/example/wordpress.yml
INFO[0000] Gathering info about 17 containers
INFO[0000] Create container wordpress.db_data
INFO[0000] Create container wordpress.db
INFO[0000] Starting container wordpress.db id:810cb0e65e2d from image mysql:5.6
INFO[0001] Waiting for 1s to ensure wordpress.db not exited abnormally...
INFO[0002] Create container wordpress.main
INFO[0002] Starting container wordpress.main id:20aa94bd256d from image wordpress:4.1.2
INFO[0002] Waiting for 1s to ensure wordpress.main not exited abnormally...
INFO[0003] Running containers: wordpress.main, wordpress.db, wordpress.db_data
```

*NOTE: I have all images downloaded already. Rocker-compose will download missing images during the first run. If you want to pull all images from the manifest separately, there is a `rocker-compose pull` command for that*

*NOTE 2: The line "Gathering info about 17 containers" just means that there are 17 containers on my machine that were created by rocker-compose. You will have 0*

Rocker-compose creates containers in a deliberate order respecting inter-container dependencies. Let's see what we've created:

```
$ docker ps -a | grep wordpress
13f34666431e        wordpress:4.1.2           "/entrypoint.sh apac   2 minutes ago      Up 2 minutes                  0.0.0.0:8080->80/tcp     wordpress.main
810cb0e65e2d        mysql:5.6                 "/entrypoint.sh mysq   2 minutes ago      Up 2 minutes                  3306/tcp                 wordpress.db
26511eaeccd2        grammarly/scratch:latest         "true"                 2 minutes ago                                               wordpress.db_data
$
```

`rocker-compose` prefixed container names with the namespace "wordpress". Namespaces help `rocker-compose` to isolate container names and also detect obsolete containers that should be removed.

You can now go to your browser and check `:8080` under your `docker-machine ip` address. Wordpress application should be there.

Assuming you have a virtual machine named `dev`, you can do:
```bash
$ open http://$(docker-machine ip dev):8080/
```

Let's inspect some stuff and connect to the Wordpress application container to see how it interacts with mysql:
```bash
# as you can see, wordpress is running a bunch of apache2 processes
$ docker exec -ti wordpress.main ps aux
USER       PID %CPU %MEM    VSZ   RSS TTY      STAT START   TIME COMMAND
root         1  0.0  0.9 177140 20184 ?        Ss   14:59   0:00 apache2 -DFOREGROUND
www-data   118  0.0  0.3 177172  7416 ?        S    14:59   0:00 apache2 -DFOREGROUND
www-data   119  0.0  0.3 177172  7416 ?        S    14:59   0:00 apache2 -DFOREGROUND
www-data   120  0.0  0.3 177172  7416 ?        S    14:59   0:00 apache2 -DFOREGROUND
www-data   121  0.0  0.3 177172  7416 ?        S    14:59   0:00 apache2 -DFOREGROUND
www-data   122  0.0  0.3 177172  7416 ?        S    14:59   0:00 apache2 -DFOREGROUND
root       131  0.0  0.1  17492  2100 ?        Rs+  15:12   0:00 ps aux

# let's look at ENV variables that are in our wordpress container
# there is a MYSQL_PORT_3306_TCP_ADDR which can be used to connect to a db container
$ docker exec -ti wordpress.main env
PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin
HOSTNAME=20aa94bd256d
MYSQL_PORT=tcp://172.17.3.21:3306
MYSQL_PORT_3306_TCP=tcp://172.17.3.21:3306
MYSQL_PORT_3306_TCP_ADDR=172.17.3.21
MYSQL_PORT_3306_TCP_PORT=3306
MYSQL_PORT_3306_TCP_PROTO=tcp
MYSQL_NAME=/wordpress.main/mysql
MYSQL_ENV_MYSQL_ROOT_PASSWORD=example
MYSQL_ENV_MYSQL_MAJOR=5.6
MYSQL_ENV_MYSQL_VERSION=5.6.25
PHP_INI_DIR=/usr/local/etc/php
PHP_EXTRA_BUILD_DEPS=apache2-dev
PHP_EXTRA_CONFIGURE_ARGS=--with-apxs2
PHP_VERSION=5.6.8
WORDPRESS_VERSION=4.1.2
WORDPRESS_UPSTREAM_VERSION=4.1.2
WORDPRESS_SHA1=9e9745bb8a1166622de866076eac73a49cb3eba0
HOME=/root

# /etc/hosts shows that there is a host entry for db container as well
$ docker exec -ti wordpress.main cat /etc/hosts
172.17.3.23 20aa94bd256d
127.0.0.1 localhost
::1 localhost ip6-localhost ip6-loopback
fe00::0 ip6-localnet
ff00::0 ip6-mcastprefix
ff02::1 ip6-allnodes
ff02::2 ip6-allrouters
172.17.3.21 mysql 810cb0e65e2d wordpress.db

# you can also open a shell inside of the wordpress container and inspect some stuff
$ docker exec -ti wordpress.main bash
root@20aa94bd256d:/var/www/html# df -h
Filesystem      Size  Used Avail Use% Mounted on
none             19G   17G  598M  97% /
tmpfs          1002M     0 1002M   0% /dev
shm              64M     0   64M   0% /dev/shm
/dev/sda1        19G   17G  598M  97% /etc/hosts
root@20aa94bd256d:/var/www/html# exit
exit
$
```

As you can see, I am almost out of space on my boot2docker virtual machine.

In case you run `rocker-compose` again without changing anything, it will ensure that nothing was changed and quit:

```
$ rocker-compose run
INFO[0000] Reading manifest: .../rocker-compose/example/wordpress.yml
INFO[0000] Gathering info about 20 containers
INFO[0000] Running containers: wordpress.main, wordpress.db, wordpress.db_data
$
```

Let's update our Wordpress application and set the newer version:

```yaml
# ...
main:
  image: wordpress:4.2.2
# ...
```

And to apply the effects of our changes, you have to repeat the run:
```
$ rocker-compose run
INFO[0000] Reading manifest:.../rocker-compose/example/wordpress.yml
INFO[0000] Gathering info about 20 containers
INFO[0000] Pulling image: wordpress:4.2.2 for wordpress.main
...
INFO[0045] Removing container wordpress.main id:20aa94bd256d
INFO[0045] Create container wordpress.main
INFO[0045] Starting container wordpress.main id:13f34666431e from image wordpress:4.2.2
INFO[0046] Waiting for 1s to ensure wordpress.main not exited abnormally...
INFO[0047] Running containers: wordpress.main, wordpress.db, wordpress.db_data
$
```

`rocker-compose` has automatically pulled the newer version 4.2.2 of Wordpress and restarted the container. Note that our `db` and `db_data` containers were untouched since they haven't been changed.

```
$ docker ps -a | grep wordpress
13f34666431e        wordpress:4.2.2          "/entrypoint.sh apac   2 minutes ago      Up 2 minutes                  0.0.0.0:8080->80/tcp     wordpress.main
810cb0e65e2d        mysql:5.6                "/entrypoint.sh mysq   15 minutes ago     Up 15 minutes                  3306/tcp                 wordpress.db
26511eaeccd2        grammarly/scratch:latest        "true"                 15 minutes ago                                               wordpress.db_data
$
```

You can see that `wordpress.main` container was restarted later than others. Also, it is running a newer version now.

Any attribute can be changed and after running `compose` again, it will change as little as it can to make the actual state match the desired one.

After experimenting you can remove containers from the manifest:
```
$ rocker-compose rm
INFO[0000] Reading manifest: .../rocker-compose/example/wordpress.yml
INFO[0000] Gathering info about 20 containers
INFO[0000] Removing container wordpress.main id:13f34666431e
INFO[0001] Removing container wordpress.db id:810cb0e65e2d
INFO[0002] Removing container wordpress.db_data id:26511eaeccd2
INFO[0002] Nothing is running
$
```

# Command line reference

##### `rocker-compose`

These options are global and can be used with any subcommand:

| option | alias | default value | description | example |
|--------|-------|---------------|-------------|---------|
| `-verbose` | `-vv` | `false` | makes debug output | `rocker-compose -vv run` |
| `-log` | `-l` | `nil` | redirects output to a log file | `rocker-compose -l out.log run` |
| `-json` | *none* | makes json output | `rocker-compose -json run` |
| `-host` | `-H` | `unix:///var/run/docker.sock` | Daemon socket(s) to connect to [$DOCKER_HOST] | `rocker-compose -H tcp://10.10.41.2:2376 run` |
| `-tlsverify` | `-tls` | `false` | Use TLS and verify the remote | |
| `-tlscacert` | *none* | `~/.docker/ca.pem` | Trust certs signed only by this CA | |
| `-tlscert` | *none* | `~/.docker/cert.pem` | Path to TLS certificate file | |
| `-tlskey` | *none* | `~/.docker/key.pem` | Path to TLS key file | |
| `-auth` | `-a` | `nil` | Docker auth, username and password in user:password format | `rocker-compose -a user:pass run` |
| `-help` | `-h` | `nil` | shows help | `rocker-compose --help` |
| `-version` | `-v` | `nil` | prints rocker-compose version | `rocker-compose -v` |

##### Common options for `run`, `pull`, `rm` and `clean` commands

| option | alias | default value | description | example |
|--------|-------|---------------|-------------|---------|
| `-file` | `-d` | `compose.yml` | Path to configuration file, if `-` is given as a value, then STDIN will be used | `rocker-compose run -f c.yml`, `cat c.yml | rocker-compose run -f -` |
| `-var` | *none* | `[]` | Set variables to pass to build tasks | `rocker-compose run -var v=1 -var dev=true` |
| `-dry` | `-d` | `false` | Don't execute any operations on target docker | `rocker-compose clean -d` |

##### `rocker-compose run` — executes manifest (compose.yml)

| option | alias | default value | description | example |
|--------|-------|---------------|-------------|---------|
| `-force` | *none* | `false` | Force recreation of all containers | `rocker-compose run -force` |
| `-attach` | *none* | `false` | Stream stdout and stderr of all containers from the spec | `rocker-compose run -attach` |
| `-pull` | *none* | `false` | Pull images before running | `rocker-compose run -pull` |
| `-wait` | *none* | `1s` | Wait and check exit codes of launched containers | `rocker-compose run -wait 5s` |
| `-ansible` | *none* | `false` | output json in ansible format for easy parsing | `rocker-compose clean -ansible` |

\+ Common options.

##### `rocker-compose pull` — pull images specified in the manifest

| option | alias | default value | description | example |
|--------|-------|---------------|-------------|---------|
| `-ansible` | *none* | `false` | output json in ansible format for easy parsing | `rocker-compose clean -ansible` |

\+ Common options.

##### `rocker-compose rm` — stop and remove any containers specified in the manifest

\+ Common options.

##### `rocker-compose clean` — cleanup old tags for images specified in the manifest

| option | alias | default value | description | example |
|--------|-------|---------------|-------------|---------|
| `-keep` | `-k` | `5` | number of last images to keep | `rocker-compose clean -k 10` |
| `-ansible` | *none* | `false` | output json in ansible format for easy parsing | `rocker-compose clean -ansible` |

\+ Common options.
 
##### `rocker-compose info` — show docker info (check connectivity, versions, etc.)

| option | alias | default value | description | example |
|--------|-------|---------------|-------------|---------|
| `-all` | `-a` | `false` | show advanced info | `rocker-compose info -a` |

# compose.yml spec

The spec is a [YAML](http://yaml.org/) file format. Note that the indentation is 2 spaces. Empty lines should be unindented.

### Example

```yaml
namespace: wordpress # specify a manifest-level namespace under which all containers will be named
containers:
  main: # container name will be "wordpress.main"
    image: wordpress:4.1.2 # run from "wordpress" image of version 4.1.2
    links:
      # link container named "db" as alias "mysql", inside the "main" container
      # you can reach "db" container by using "mysql" host or using MYSQL_PORT_3306_TCP_ADDR env var
      - db:mysql
    ports:
      - "8080:80" # throw 8080 port to a host network, map it to 80 internal port

  db:
    image: mysql:5.6
    env:
      MYSQL_ROOT_PASSWORD: example # provide MYSQL_ROOT_PASSWORD env var
    volumes_from:
      # specify to mount all volumes from "db_data" container, this way we can
      # update "db" container without loosing data
      - db_data 

  db_data:
    image: grammarly/scratch:latest # use empty image, just for data
    state: created # this tells compose to not try to run this container, data containers need to be only created
    volumes:
      # define the empty directory that will be used by the "db" container
      - /var/lib/mysql
```

rocker-compose is also compatible with docker-compose format, where containers are specified in the root level:

```yaml
main:
  image: wordpress:4.1.2
  links: db:mysql
  ports: "8080:80"

db:
  image: mysql:5.6
  env: MYSQL_ROOT_PASSWORD=example
  volumes_from: db_data 

db_data:
  image: grammarly/scratch:latest
  state: created
  volumes: /var/lib/mysql
```

In this case, namespace will be the name of parent directory of your `compose.yml` file.

### Types

String:
```yaml
image: wordpress

cmd: while true; do sleep 1; done

cmd: |-
  set -e
  touch /var/log/out.log
  while true; do echo "hello" >> /var/log/out.log; sleep 1; done

str1: and i am also a string
str2: "and i"
```

Array:
```yaml
cmd: ["/bin/sh", "-c", "echo hello"]

volumes_from:
  - data
  - config

ports:
  - "8080:80"
```

Hash:
```yaml
env:
  DB_PASSWORD: lopata
  DB_HOST: localhost
```

Bool:
```yaml
evil: false
good: true
```

Number:
```yaml
kill_timeout: 123
```

Ulimit:
```yaml
ulimits:
  - name: nofile
    soft: 1024
    hard: 2048
```

### Root level properties

| Property | Default value | Type | Description |
|----------|---------------|------|-------------|
| **namespace** | *REQUIRED* | String | root namespace to prefix all container names in the current manifest |
| **containers** | *REQUIRED* | Hash | list of containers to run within the current namespace where every key:value pair is a container name as a key and container spec as a value |

### Container properties

Example:
```yaml
namespace: wordpress
containers:
  main:
    image: wordpress
```

Where `main` is a container name and `image: wordpress` is its spec. If container name begins with an underscore (`_`) then `rocker-compose` will not consider it — useful for creating base specs for [extends](#extends). Note that by convention, properties should be maintained in the given order when writing compose manifests.

| Property | Default | Type | Run param | Description |
|----------|---------|------|-----------|-------------|
| **extends** | *nil* | String | *none* | `container_name` - extend spec from another container of the current manifest |
| **image** | *REQUIRED* | String | `docker run <image>` | image name for the container, the syntax is `[registry/][repo/]name[:tag]` |
| **state** | `running` | String | *none* | `running`, `ran`, `created` - desired state of a container ([read more about state](#state)) |
| **entrypoint** | *nil* | Array\|String | [`--entrypoint`](https://docs.docker.com/reference/run/#entrypoint-default-command-to-execute-at-runtime) | overwrite the default entrypoint set by the image |
| **cmd** | *nil* | Array\|String | `docker run <image> <cmd>` | the list of command arguments to pass |
| **workdir** | *nil* | String | [`-w`](https://docs.docker.com/reference/run/#workdir) | set working directory inside the container |
| **restart** | `always` | String | [`--restart`](https://docs.docker.com/reference/run/#restart-policies-restart) | `never`, `always`, `on-failure,N` - container restart policy |
| **labels** | *nil* | Hash\|String | `--label FOO=BAR` | key/value labels to add to the container |
| **env** | *nil* | Hash\|String | [`-e`](https://docs.docker.com/reference/run/#env-environment-variables) | key/value ENV variables |
| **wait_for** | *nil* | Array\|String | *none* | array of container names - wait for other containers to start before starting the container |
| **links** | *nil* | Array\|String | [`--link`](https://docs.docker.com/userguide/dockerlinks/) | other containers to link with; can be `container` or `container:alias` |
| **volumes_from** | *nil* | Array\|String | [`--volumes-from`](https://docs.docker.com/userguide/dockervolumes/) | mount volumes from other containers |
| **volumes** | *nil* | Array\|String | [`-v`](https://docs.docker.com/userguide/dockervolumes/) | specify volumes of a container, can be `path` or `src:dest` [read more](#volumes) |
| **expose** | *nil* | Array\|String | [`--expose`](https://docs.docker.com/articles/networking/) | expose a port or a range of ports from the container without publishing it/them to your host; e.g. `8080` or `8125/udp` |
| **ports** | *nil* | Array\|String | [`-p`](https://docs.docker.com/articles/networking/) | publish a container᾿s port or a range of ports to the host, e.g. `8080:80` or `0.0.0.0:8080:80` or `8125:8125/udp` |
| **publish_all_ports** | `false` | Bool | [`-P`](https://docs.docker.com/articles/networking/) | every port in `expose` will be published to the host |
| **log_driver** | `json-file` | string | [`--log-driver`](https://docs.docker.com/reference/logging/overview/) | logging driver |
| **log_opt** | `max-file:5 max-size:100m` | Hash | [`--log-opt`](https://docs.docker.com/reference/logging/overview/) | logging driver configuration |
| **dns** | *nil* | Array\|String | [`--dns`](https://docs.docker.com/reference/run/#network-settings) | add DNS servers to the container |
| **add_host** | *nil* | Array\|String | [`--add-host`](https://docs.docker.com/reference/run/#network-settings) | add records to `/etc/hosts` file, e.g. `mysql:172.17.3.21` |
| **net** | `bridge` | String | [`--net`](https://docs.docker.com/reference/run/#network-settings) | network mode, options are: `bridge`, `host`, `container:<name|id>`; `none` is used to disable networking |
| **hostname** | *nil* | String | [`--hostname`](https://docs.docker.com/reference/run/#network-settings) | set a custom hostname for the container |
| **domainname** | *nil* | String | [`--dns-search`](https://docs.docker.com/articles/networking/#configuring-dns) | set the search domain to `/etc/resolv.conf` |
| **user** | *nil* | String | [`-u`](https://docs.docker.com/reference/run/#user) | run container process with specified user or UID |
| **uts** | *nil* | String | [`--uts`](https://docs.docker.com/reference/run/#uts-settings-uts) | if set to `host` container will inherit host machine's hostname and domain; warning, **insecure**, use only with trusted containers |
| **pid** | *nil* | String | [`--pid`](https://docs.docker.com/reference/run/#pid-settings-pid) | set the PID (Process) Namespace mode for the container, when set to `host` will be in host machine's namespace |
| **privileged** | `false` | Bool | [`--privileged`](https://docs.docker.com/reference/run/#runtime-privilege-linux-capabilities-and-lxc-configuration) | give extended privileges to this container |
| **memory** | *nil* | String|Number | [`--memory`](https://docs.docker.com/reference/run/#runtime-constraints-on-resources) | `<number><unit>` limit memory for container where units are `b`, `k`, `m` or `g` |
| **memory_swap** | *nil* | String|Number | [`--memory-swap`](https://docs.docker.com/reference/run/#runtime-constraints-on-resources) | limit total memory (memory + swap), format same as for **memory** |
| **cpu_shares** | *nil* | Number | [`--cpu-shares`](https://docs.docker.com/reference/run/#runtime-constraints-on-resources) | CPU shares (relative weight) |
| **cpu_period** | *nil* | Number | [`--cpu-period`](https://docs.docker.com/reference/run/#runtime-constraints-on-resources) | limit the CPU CFS (Completely Fair Scheduler) period |
| **cpuset_cpus** | *nil* | String | [`--cpuset-cpus`](https://docs.docker.com/reference/run/#runtime-constraints-on-resources) | CPUs in which to allow execution, e.g. `0-3` or `0,1` |
| **ulimits** | *nil* | Array of Ulimit | [`--ulimit`](https://github.com/docker/docker/pull/9437) | ulimit spec for the container |
| **kill_timeout** | `0` | Number | *none* | timeout in seconds to wait for container to [stop before killing it](https://docs.docker.com/reference/commandline/stop/) with `-9` |
| **keep_volumes** | `false` | Bool | *none* | tell `rocker-compose` to keep volumes when removing the container |

Some aliases are supported for compatibility with `docker-compose` and `docker run` specs:

| docker_compose | rocker_compose  |
|----------------|-----------------|
| `command`      | `cmd`           |
| `link`         | `links`         |
| `label`        | `labels`        |
| `hosts`        | `add_host`      |
| `extra_hosts`  | `add_host`      |
| `working_dir`  | `workdir`       |
| `environment`  | `env`           |

# State
For every pair of containers with the same name, `rocker-compose` does a comparison of all properties to figure out changes, as well as a check of the running state. To determine if the container should be restarted, in case all other properties are equal, `rocker-compose` uses the following decision scheme:

| Desired State | Actual State | Exit Code | Action                 |
|---------------|--------------|-----------|------------------------|
| running       | not exists   | *none*    | start                  |
| running       | exists       | *any*     | remove and start       |
| running       | restarting   | *any*     | remove and start       |
| running       | running      | *none*    | NOOP                   |
| created       | not exists   | *none*    | create                 |
| created       | exists       | *any*     | NOOP                   |
| created       | restarting   | *any*     | remove and create      |
| created       | running      | *none*    | remove and create      |
| ran           | not exists   | *none*    | start and wait         |
| ran           | exists       | `0`       | NOOP                   |
| ran           | exists       | non-zero  | remove, start and wait |
| ran           | restarting   | *any*     | wait                   |
| ran           | running      | *none*    | NOOP?                  |

*NOTE: by "start" here we mean "create" and then "start"*

**state: ran** is used for single-shot commands to perform some initialization. `rocker-compose` does not re-run such containers unless they have changed or previous executions exited with non-zero code.

**state: created** is mostly used for data volume and network-share containers. They are described in the [patterns](#patterns) section.

# Volumes
It is possible to mount volumes to a running container the same way as it is when using plain `docker run`. In Docker, there are two types of volumes: **Data volume** and **Mounted host directory**. 

### Data volume
"Data volume" is a reusable directory managed by Docker daemon that can be shared between containers. Most often, this type of file sharing across containers should be used because of its [12factor](http://12factor.net/) compliance — you can think of it as "data volume as a service".

Example:
```yaml
namespace: wordpress
containers:
  db:
    image: mysql:5.6
    volumes_from:
      # specify to mount all volumes from "db_data" container, this way we can
      # update "db" container without loosing data
      - db_data 

  db_data:
    image: grammarly/scratch:latest # use empty image, just for data
    state: created # this tells compose to not try to run this container, data containers needs to be just created
    volumes:
      # define the empty directory that will be used by "db" container
      - /var/lib/mysql

  # Cron job container that will periodically backup data from /var/lib/mysql volume in db_data container
  db_backup:
    image: some_cron_backuper_image
    volumes_from: db_data
```

### Mounted host directory
While it is useful for development and testing, it's unsafe and error-prone for production use. It requires some external folder to exist on a host machine in order to run your container. Also, it may cause some unpleasant failure modes hard to reproduce. And finally, you cannot guarantee reproducibility of your manifests.

The rule of thumb with "Mounted host directories" is the following:

1. Use it only for development
2. Use it for logging or mounting external devices, such as EBS volumes *(this one may be covered by tools like [flocker](https://github.com/ClusterHQ/flocker) or future docker volume drivers)*

Example:
```yaml
namespace: wordpress
containers:
  db:
    image: mysql:5.6
    volumes:
      # mount /mnt/data directory from host machine to /var/lib/mysql in the container
      # container can be safely removed without data loss
      - /mnt/data:/var/lib/mysql

  # Cron job container that will periodically backup data from /mnt/data host machine directory
  db_backup:
    image: some_cron_backuper_image
    volumes: /mnt/data:/var/lib/mysql
```

Development example:
```yaml
namespace: wordpress
containers:
  main:
    image: wordpress:4.1.2
    links: db:mysql
    volumes:
      # mount ./wordpress-src directory to /var/www/html in the container, this way we can hack wordpress sources while the container is running
      - ./wordpress-src:/var/www/html
    ports:
      - "8080:80"
```

*NOTE: you cannot use the last example for production, obviously, because there should be no such directory as `./wordpress-src`*

# Extends
You can extend some container specifications within a single manifest file. In this example, we will run two identical wordpress containers and assign them to different ports:
```yaml
namespace: wordpress
containers:
  # define base _main container spec; it will be ignored by rocker-compose because it starts from _
  _main:
    image: wordpress:4.1.2
    links: db:mysql

  # extend main1 from _main and override ports to listen on :8080
  main1:
    extends: _main
    ports: "8080:80"

  # extend main2 from _main and override ports to listen on :8081
  main2:
    extends: _main
    ports: "8081:80"
```

**NOTE:** nested extends are not allowed by `rocker-compose`.

# Templating
`rocker-compose` uses Go [text/template](http://golang.org/pkg/text/template/) engine to render manifests. This way you can put some logic into your manifests or even inject some variables from the outside:
```yaml
namespace: wordpress
containers:
  main:
    image: wordpress:4.1.2
    links: db:mysql
    {{ if eq .env "dev" }}
    volumes:
      # mount ./wordpress-src directory to /var/www/html in the container, such way we can hack wordpress sources while the container is running
      - ./wordpress-src:/var/www/html
    {{ end }}
    ports:
      - {{ or .port "8080" }}:80
```

You can run this manifest as follows:
```bash
$ rocker-compose run                             # will not mount src volume and run on 8080
$ rocker-compose run -var env=dev                # will mount src volume and run on :8080
$ rocker-compose run -var env=dev -var port=8081 # will mount src volume and run on :8081
```

In addition to the [builtin helper functions](http://golang.org/pkg/text/template/#hdr-Functions) there are some provided by `rocker-compose`:

###### {{ bridgeIp }} [Example](#loose-coupling-network)
Returns Docker's [bridge gateway ip](https://docs.docker.com/articles/networking/), which can be used to access any exposed ports of an external container. Useful for loose coupling. [Source](https://github.com/grammarly/rocker-compose/blob/88007dcf571da7617f775c9abe1824eedc9598fb/src/compose/docker.go#L59)

###### {{ seq *To* }} or {{ seq *From* *To* }} or {{ seq *From* *To* *Step* }}
Sequence generator. Returns an array of integers of a given sequence. Useful when you need to duplicate some configuration, for example scale containers of the same type. Mostly used in combination with `range`:
```
{{ range $i := seq 1 5 2 }}
container-$i
{{ end }}
```

This template will yield:
```
container-1
container-3
container-5
```

See [this example](#dynamic-scaling) of using `seq` for dynamically scaling containers.

###### {{ getenv *key* }} 
Get environment variable. Returns string.
```
{{ $i := getenv "TEST_STUFF" }}
```

```
TEST_STUFF=1 rocker-compose
```

# Dynamic scaling
Sometimes you need to dynamically set the number of containers to be started. `docker-compose` has [scale](https://docs.docker.com/compose/cli/#scale) command that does exactly what we want. With `rocker-compose` we can template the configuration with the help of the `seq` generator:

```yaml
namespace: scaling
containers:
  {{ range $n := seq .n }}
  worker_{{$n}}:
    image: busybox:buildroot-2013.08.1
    command: for i in `seq 1 10000`; do echo "hello $i!!!!"; sleep 1; done
  {{ end }}
```

By running `rocker-compose` with some value for a variable `n`, it will spawn a desired number of "worker" containers:
```bash
rocker-compose run -var n=1 # will spawn worker_1
rocker-compose run -var n=2 # will add worker_2 while worker_1 is still running
rocker-compose run -var n=4 # will add worker_3 and worker_4 while worker_1 and worker_2 are running
rocker-compose run -var n=1 # will kill worker_2, worker_3, and worker_4, while keeping worker_1 running
```

### A more advanced example
We can specify complete groups of containers running independently. Here we use `_base` container configuration to extend our workers from. Each worker writes its name and message sequence number to a log, which is stored in a dedicated volume container. From the other side, there is a `tail_container` for each worker, that tails the worker's log.
```yaml
namespace: scaling
containers:
  _base:
    image: busybox:buildroot-2013.08.1
    cmd: for i in `seq 1 10000`; do echo "hello $NAME $i!!!!" >> /tmp/log; sleep 1; done
    env:
      NAME: {{or .name "NONE"}}

  {{ range $n := seq .n }}
  worker_{{$n}}:
    extends: _base
    env: NAME=worker-{{$n}}
    volumes_from: volume_container_{{$n}}

  volume_container_{{$n}}:
    image: grammarly/scratch:latest
    state: created
    volumes: /tmp

  tail_container_{{$n}}:
    image: ubuntu:12.04
    cmd: tail -f /tmp/log
    volumes_from: volume_container_{{$n}}
    links: worker_{{$n}}
  {{ end }}
```

# Patterns
Here is a list of the most common problems with multi-container applications and ways you can solve them with `rocker-compose`.

### Data volume containers
By design, containers are transient. Most of the tools for containerized applications are built expecting your apps to respect this rule. Your container can be dropped and created from scratch at any time. For example, to update the image some container is running, you have to remove the container and create a new one. This is a property of **immutable infrastructure**. In Docker, every container has its own dedicated file system, and by default it is removed along with the container. There is a `VOLUME` directive, which creates a separate data volume associated with the container that is able to persist after container removal. But there is no way to re-associate the old detached volume with a new container.

A known workaround for containers' transient properties while not losing persistent data is to make a "data volume container" and mount its volumes to your application container.
```yaml
namespace: wordpress
containers:
  db:
    image: mysql:5.6
    env: MYSQL_ROOT_PASSWORD=example
    volumes_from:
      # db container can be easily re-created without losing data
      # all data will remain in /var/lib/mysql associated with db_data container
      - db_data 

  db_data:
    image: grammarly/scratch:latest # use empty image, just for data
    state: created # this tells compose to not try to run this container, data containers need to be just created
    volumes:
      # define the empty directory that will be used by "db" container
      - /var/lib/mysql
```

Another reason for using this pattern is to split the lifecycle of your application from its configuration:
```yaml
namespace: wordpress
containers:
  main:
    image: wordpress:4.1.2
    volumes_from: main_config

  main_config:
    image: my_wordpress_config
    state: created
```

This way, you can release `my_wordpress_config` independently from `wordpress` and deliver it separately. Keep in mind, though, that the `main` container will be restarted any time `main_config` changes.

Dockerfile of the `my_wordpress_config` image might look like the following:
```bash
FROM scratch
ADD ./config.json /etc/wordpress/config.json  # add config file from the context directory to the image
VOLUME /etc/wordpress                         # declare /etc/wordpress to be shareable
```

The directory `/etc/wordpress` will appear in the `main` container when it is started.

### Bootstrapping
Sometimes you want to do some initialization prior to starting your application container. For example, you may want to create an initial user for a database. [Most of the time](https://github.com/docker-library/mysql/blob/master/5.6/docker-entrypoint.sh), people do some sort of `./docker-entrypoint.sh` wrapping entrypoint script, which is executed as the main process in a container. It does some checks and initialization and then runs an actual process.

There are situations when you are using some vendor image and do not want to extend from it or modify it in any way. In this case, you can use single-run container semantics:
```yaml
namespace: sensu
containers:
  sensu_client:
    image: sensu-client
    links: rabbitmq
    wait_for:
      - bootstrap # wait for `bootstrap` container to finish

  rabbitmq:
    image: rabbitmq:3-management

  bootstrap:
    image: tutumcloud/curl
    state: ran # rocker-compose will run this container once, unless it's changed
    cmd: |-
      curl --silent -u guest:guest -XPUT -H "Content-Type: application/json" \
      http://$RABBITMQ_PORT_15672_TCP_ADDR:15672/api/users/sensu \
      -d '{"password":"sensu","tags":""}'
    links:
      - rabbitmq
```
This may look like an ugly example, but [`state:ran`](#state) and [`wait_for`](#container-properties) are useful primitives that can be used in other cases as well.

Here, the start order will be the following:

1. `rabbitmq` will go first because it does not have any dependencies
2. `bootstrap` will run its curl command
3. `sensu_client` will be started when `bootstrap` finished successfully, "sensu" user will be already created

### Loose coupling: network
The most correct way of linking Docker containers between each other is using [`--link`](https://docs.docker.com/userguide/dockerlinks/) primitive. However, in case of container A linked to container B, A needs to be recreated every time we update B. If you rely on ENV variables provided by the links functionality, [you cannot update dependencies](https://docs.docker.com/userguide/dockerlinks/#important-notes-on-docker-environment-variables) without recreating your container.

Docker also automatically populates entries to `/etc/hosts` inside your container. It also updates hosts entries when dependencies **restart**, but not when you **recreate** them (that commonly happens when you update underlying images). Accessing links through `/etc/hosts` also does not help loose coupling because you need B to exist anyway in order to even start A.

Both approaches are considered tight coupling and it's ok to use them as long as you acknowledge that fact.

Sometimes you want to detach your application container from some dependency. Let's assume you have an app which is writing metrics to [StatsD](https://github.com/etsy/statsd) daemon by UDP, and you don't care if the daemon is present at the time you start your application. 

```yaml
namespace: myapp
containers:
  # container A
  main:
    image: alpine:3.2
    # writes counter to statsd every second, note `statsd` host to `nc`
    cmd: while true; do echo "foo:1|c" | nc -u -w0 statsd 8125; sleep 1; done
    add_host:
      # this will populate "statsd" host mapped to a
      # Docker's bridge ip address
      - statsd:{{ bridgeIp }}

# in another manifest managed by another team
namespace: platform
containers:
  # container B
  statsd:
    image: statsd
    ports:
      # this will bind 8125/udp port and it will be available
      # on the Docker's bridge ip
      - "8125:8125/udp"
```

This way, `myapp.main` can run independently from `platform.statsd` and at the same time be able to write metrics to `statsd:8125`. In case statsd is not present, metrics will be simply dropped on the floor.

This is a tradeoff because you have to expose a known port to a host network. Ports may clash, you don't have full isolation here and benefit from random port mapping. In every particular situation you have to balance between loose coupling and isolation.

### Network share
In case you have containers A and B from the example above in the same manifest, you can do loose coupling without exposing any global ports to a host network. The trick is to use `net: container:<id|name>` feature:

```yaml
namespace: myapp
containers:
  # container A
  main:
    image: alpine:3.2
    # wire the network to a dummy container
    net: container:dummy
    # writes counter to statsd every second, note `127.0.0.1` host to `nc`
    cmd: while true; do echo "foo:1|c" | nc -u -w0 127.0.0.1 8125; sleep 1; done

  # container B
  statsd:
    image: statsd
    # wire the network to a dummy container
    net: container:dummy

  # simple container that hangs in 'while true' forever
  dummy:
    image: grammarly/net
```

In this example `dummy` container plays the role of a network host. Containers that connected to it by `net: container:dummy` share all ports between each other.

**Note** that ports now may clash between container A and B since they now are in the same network.

**Keep in mind** that you cannot mix links with `net:container` mode.

### Loose coupling: files

TODO

# Contributing

### Dependencies

Use [gb](http://getgb.io/) to test and build. We vendor all dependencies, you can find them under `/vendor` directory.

Please, use [gofmt](https://golang.org/cmd/gofmt/) in order to automatically re-format Go code into vendor standardised convention. Ideally, you have to set it on post-save action in your IDE. For SublimeText3, [GoSublime](https://github.com/DisposaBoy/GoSublime) package does the right thing. Also, [solution for Intellij IDEA](http://marcesher.com/2014/03/30/intellij-idea-run-goimports-on-file-save/).

### Build

(will produce a binary into `bin/` directory)
```bash
gb build
```

or build for all platforms:
```bash
make
```

If you have a github access token, you can also do a github release:
```bash
make release
```

Also a useful thing to have:
```bash
echo "make test" > .git/hooks/pre-push && chmod +x .git/hooks/pre-push
```

### Test 

```bash
make test
```

or
```bash
gb test compose/...
```

### Test something particular

```bash
gb test compose/... -run TestMyFunction
```

### Command-line completions
You can find [completions](https://en.wikipedia.org/wiki/Command-line_completion) for Zsh in `completion/zsh` source directory.
Install procedure described at [docker site](https://docs.docker.com/compose/completion/).

### TODO

* [x] Introduce templating for compose.yml to substitute variables from the outside
* [x] Refactor config.go - move some functions to config_convert.go
* [X] Remove obsolete containers (e.g. removed from compose.yml)
* [X] EnsureContainer for containers out of namespace (cannot be created)
* [X] client.go execution functions
* [X] Add labels for containers launched by compose?
* [X] rocker-compose executable with docker connection and cli flags
* [X] Choose and adopt a logging framework
* [X] Protect from loops in dependencies
* [X] Cross-compilation for linux and darwin (run in container? how will gb work?)
* [x] Attach stdout of launched (or existing) containers
* [x] Force-restart option
* [x] Never remove volumes of some containers
* [x] Parallel pull operation
* [x] Force-pull option (if image exists, only for "latest" or non-semver tags?)
* [x] Clean command, keep_versions config attribute for containers
* [x] ansible-module mode for rocker-compose executable
* [x] Write detailed readme, manual and tutorial
* [ ] Dry mode, todo: ensure dry works for all actions

```bash
grep -R TODO **/*.go | grep -v '^vendor/'
```

# Authors

- Yura Bogdanov <yuriy.bogdanov@grammarly.com>
- Stas Levental <stanislav.levental@grammarly.com>

# License

(c) Copyright 2015 Grammarly, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
