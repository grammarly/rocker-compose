# rocker-compose

Composition tool for running multiple Docker containers on any machine. It's intended to be used in a following cases:

1. Deploying containerized apps to servers
2. Running containerized apps locally for development or testing

There is an official [docker-compose](https://github.com/docker/compose) tool which is made exactly for the same purpose. But we found that it is missing a few key features that makes us unable using it for deployments. For us, composition tool should:

1. Be able to read the manifest (configuration file) and run an isolated chain of containers
2. Support all Docker's configuration options, such as all you can do with plain `docker run`, you can do with `compose` *(for some options, docker-compose do not have the most intuitive names, rocker-compose uses convention to fit the names of original run spec)*
3. Be idempotent. Only affected containers should be restarted. *(docker-compose simply restarts everything on every run)* 
4. Support configurable namespaces and avoid name clashes between apps *(docker-compose does not even support underscores in container names, that's a bummer)*
5. Remove containers that are not in the manifest anymore *docker-compose does not*
6. Respect any changes that can be made to containers configuration. Images can be updated, their names might stay same, in cases of using `:latest` tags
7. Dependency graph can also define which actions may run in parallel, utilize it
8. Support templating in the manifest file. Not just putting ENV variables, but also be able to do conditionals, etc. *(docker-compose does not have it, but they recently came up with a [pretty good solution](https://github.com/docker/compose/issues/1377), which we may adopt soon as well)*

Contributing these features to docker-compose was also an option, but we decided to come up with own solution due the following reasons:

1. docker-machine is written in Python, we don't have tools in Python
2. We have a full control over the tool and can add any feature to it any time
3. The tool should be written in Go to benefit from the existing ecosystem, also it is easier to install it on a development machine or on any instance or CI server
4. Time factor was also critical, we were able to come up with a working solution in a four days

# Tutorial

Here is an [example of running a wordpress application](/example/wordpress.yml) with `rocker-compose`:
```yaml
namespace: wordpress # specify a manifest-level namespace under which all containers will be named
containers:
  main: # container name will be "wordpress.main"
    image: wordpress:4.1.2 # run from "wordpress" image of version 4.1.2
    links:
      # link container named "db" as alias "mysql", inside "main" container
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
    image: grammarly/scratch # use empty image, just for data
    state: created # this tells compose to not try to run this container, data containers needs to be just created
    volumes:
      # define the empty directory that will be used by "db" container
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

*NOTE 2: the line "Gathering info about 17 containers" just means that there are 17 containers on my machine that were created by rocker-compose. You will have 0*

Rocker-compose creates containers in a deliberate order, respecting inter-container dependencies. Let's see what we've created:

```
$ docker ps -a | grep wordpress
13f34666431e        wordpress:4.1.2           "/entrypoint.sh apac   2 minutes ago      Up 2 minutes                  0.0.0.0:8080->80/tcp     wordpress.main
810cb0e65e2d        mysql:5.6                 "/entrypoint.sh mysq   2 minutes ago      Up 2 minutes                  3306/tcp                 wordpress.db
26511eaeccd2        grammarly/scratch         "true"                 2 minutes ago                                               wordpress.db_data
$
```

Rocker-compose prefixed container names with the namespace "wordpress". Namespace helps rocker-compose to isolate containers names and also detecting obsolete containers that should be removed.

You can now go to your browser and check `:8080` under your `docker-machine ip` address. Wordpress application should be there.

Let's inspect some stuff and connect to the wordpress application container to see how it interacts with mysql:
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

# you can also open a shell inside the wordpress container and inspect some stuff
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

In case you run rocker-compose again without changing anything, it will ensure that nothing was changed and quit:

```
$ rocker-compose run
INFO[0000] Reading manifest: .../rocker-compose/example/wordpress.yml
INFO[0000] Gathering info about 20 containers
INFO[0000] Running containers: wordpress.main, wordpress.db, wordpress.db_data
$
```

Let's update our wordpress application and set the newer version:

```yaml
# ...
main:
  image: wordpress:4.2.2
# ...
```

And to make effect of our changes, you have to repeat the run:
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

Rocker-compose automatically pulled the newer version 4.2.2 of wordpress and restarted the container. Note that our "db" and "db_data" containers were untouched since they haven't been changed.

```
$ docker ps -a | grep wordpress
13f34666431e        wordpress:4.2.2          "/entrypoint.sh apac   2 minutes ago      Up 2 minutes                  0.0.0.0:8080->80/tcp     wordpress.main
810cb0e65e2d        mysql:5.6                "/entrypoint.sh mysq   15 minutes ago     Up 15 minutes                  3306/tcp                 wordpress.db
26511eaeccd2        grammarly/scratch        "true"                 15 minutes ago                                               wordpress.db_data
$
```

You can see that "wordpress.main" container was restarted later than others. Also, it is running a newer version now.

Any attribute can be changed and after running compose again, it will change as little as it can to make the actual state match the desired one.

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

# compose.yml spec

The spec is a [YAML](http://yaml.org/) file format. Note that the indentation is 2 spaces. Empty lines should be unindented.

### Types

String:
```yaml
image: wordpress

cmd: while true; do sleep 1; done

cmd: >
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
| **namespace** | *REQUIRED* | String | root namespace to prefix all container names in current manifest |
| **containers** | *REQUIRED* | Hash | list of containers to run within current namespace where every key:value pair is a container name as a key and container spec as a value |

### Container properties

example:
```yaml
namespace: wordpress
containers:
  main:
    image: wordpress
```

Where `main` is a container name and `image: wordpress` is its spec. If container name beginning with underscore `_` then rocker-compose will consider it. Useful for doing base specs for [extends](#extends). Note that by convension, properties should be maintained in a given order when writing compose manifests.

| Property | Default | Type | Run param | Description |
|----------|---------|------|-----------|-------------|
| **extends** | *nil* | String | *none* | `container_name` - extend spec from another container of current manifest |
| **image** | *REQUIRED* | String | `docker run <image>` | image name for the container, the syntax is `[registry/][repo/]name[:tag]` |
| **state** | `running` | String | *none* | `running`, `ran`, `created` - desired state of a container, [read more about state](#state) |
| **entrypoint** | *nil* | Array/String | [`--entrypoint`](https://docs.docker.com/reference/run/#entrypoint-default-command-to-execute-at-runtime) | overwrite the default entrypoint set by the image |
| **cmd** | *nil* | Array/String | `docker run <image> <cmd>` | the list of command arguments to pass |
| **workdir** | *nil* | String | [`-w`](https://docs.docker.com/reference/run/#workdir) | set working directory inside the container |
| **restart** | `always` | String | [`--restart`](https://docs.docker.com/reference/run/#restart-policies-restart) | `never`, `always`, `on-failure,N` - container restart policy |
| **labels** | *nil* | Hash | `--label FOO=BAR` | key/value labels to add to a container |
| **env** | *nil* | Hash | [`-e`](https://docs.docker.com/reference/run/#env-environment-variables) | key/value ENV variables |
| **wait_for** | *nil* | Array | *none* | array of container names - wait for other containers to start before starting the container |
| **links** | *nil* | Array | [`--link`](https://docs.docker.com/userguide/dockerlinks/) | other containers to link with; can be `container` or `container:alias` |
| **volumes_from** | *nil* | Array | [`--volumes-from`](https://docs.docker.com/userguide/dockervolumes/) | mount volumes from another containers |
| **volumes** | *nil* | Array | [`-v`](https://docs.docker.com/userguide/dockervolumes/) | specify volumes of a container, can be `path` or `src:dest` [read more](#volumes) |
| **expose** | *nil* | Array | [`--expose`](https://docs.docker.com/articles/networking/) | expose a port or a range of ports from the container without publishing it to your host; e.g. `8080` or `8125/udp` |
| **ports** | *nil* | Array | [`-p`](https://docs.docker.com/articles/networking/) | publish a container᾿s port or a range of ports to the host, e.g. `8080:80` or `0.0.0.0:8080:80` or `8125:8125/udp` |
| **publish_all_ports** | `false` | Bool | [`-P`](https://docs.docker.com/articles/networking/) | every port in `expose` will publish to a host |
| **dns** | *nil* | Array | [`--dns`](https://docs.docker.com/reference/run/#network-settings) | add DNS servers to a container |
| **add_host** | *nil* | Array | [`--add-host`](https://docs.docker.com/reference/run/#network-settings) | add records to `/etc/hosts` file, e.g. `mysql:172.17.3.21` |
| **net** | `bridge` | String | [`--net`](https://docs.docker.com/reference/run/#network-settings) | network mode, options are: `bridge`, `host`, `container:<name|id>`; `none` is to disable networking |
| **hostname** | *nil* | String | [`--hostname`](https://docs.docker.com/reference/run/#network-settings) | set custom hostname to a container |
| **domainname** | *nil* | String | [`--dns-search`](https://docs.docker.com/articles/networking/#configuring-dns) | set the search domain to `/etc/resolv.conf` |
| **user** | *nil* | String | [`-u`](https://docs.docker.com/reference/run/#user) | run container process with specified user or UID |
| **uts** | *nil* | String | [`--uts`](https://docs.docker.com/reference/run/#uts-settings-uts) | if set to `host` container will inherit host machine's hostname and domain; warning, **insecure**, use only with trusted containers |
| **pid** | *nil* | String | [`--pid`](https://docs.docker.com/reference/run/#pid-settings-pid) | set the PID (Process) Namespace mode for the container, when set to `host` will be in host machine's namespace |
| **privileged** | `false` | Bool | [`--privileged`](https://docs.docker.com/reference/run/#runtime-privilege-linux-capabilities-and-lxc-configuration) | give extended privileges to this container |
| **memory** | *nil* | String/Number | [`--memory`](https://docs.docker.com/reference/run/#runtime-constraints-on-resources) | `<number><unit>` limit memory for container where units are `b`, `k`, `m` or `g` |
| **memory_swap** | *nil* | String/Number | [`--memory-swap`](https://docs.docker.com/reference/run/#runtime-constraints-on-resources) | limit total memory (memory + swap), formar same as for **memory** |
| **cpu_shares** | *nil* | Number | [`--cpu-shares`](https://docs.docker.com/reference/run/#runtime-constraints-on-resources) | CPU shares (relative weight) |
| **cpu_period** | *nil* | Number | [`--cpu-period`](https://docs.docker.com/reference/run/#runtime-constraints-on-resources) | limit the CPU CFS (Completely Fair Scheduler) period |
| **cpuset_cpus** | *nil* | String | [`--cpuset-cpus`](https://docs.docker.com/reference/run/#runtime-constraints-on-resources) | CPUs in which to allow execution, e.g. `0-3` or `0,1` |
| **ulimits** | *nil* | Array of Ulimit | [`--ulimit`](https://github.com/docker/docker/pull/9437) | ulimit spec for container |
| **kill_timeout** | 0 | Number | *none* | timeout in seconds to wait container to [stop before killing it](https://docs.docker.com/reference/commandline/stop/) with `-9` |
| **keep_volumes** | `false` | Bool | *none* | tell rocker-compose to keep volumes when removing the container |

# State

TODO

# Volumes

TODO

# Extends

TODO

# Templating

TODO

# Contributing

### Dependencies

Use [gb](http://getgb.io/) to test and build. We vendor all dependencies, you can find them under `/vendor` directory.

### Build

```bash
gb build
```

or build for all platforms:
```bash
make
```

if you have a github access token, you can also do a github release
```bash
make release
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
