# rocker-compose

Composition tool for running multiple Docker containers on any machine. It's intended to be used in a following cases:

1. Deploying containerized apps to servers
2. Running containerized apps locally for development or testing

# Table of contents
* [Rationale](#rationale)
* [How it works](#how-it-works)
* [Production use](#production-use)
* [Tutorial](#tutorial)
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
* [Patterns](#patterns)
  * [Data volume containers](#data-volume-containers)
  * [Bootstrapping](#bootstrapping)
  * [Loose coupling: network](#loose-coupling-network)
  * [Network share](#network-share)
  * [Loose coupling: files](#loose-coupling-files)
* [Contributing](#contributing)
* [Todo](#todo)
* [Authors](#authors)

# Rationale
There is an official [docker-compose](https://github.com/docker/compose) tool that was made exactly for the same purpose. But we found that it is missing a few key features that makes us unable using it for deployments. For us, composition tool should:

1. Be able to read the manifest (configuration file) and run an isolated chain of containers, respecting a dependency graph
2. Be idempotent. Only affected containers should be restarted. *(docker-compose simply restarts everything on every run)*
3. Support all Docker's configuration options, such as all you can do with plain `docker run`
4. Support configurable namespaces and avoid name clashes between apps *(docker-compose does not even support underscores in container names, that's a bummer)*
5. Remove containers that are not in the manifest anymore *docker-compose does not*
6. Respect any changes that can be made to containers configuration. Images can be updated, their names might stay same, in cases of using `:latest` tags
7. Dependency graph can also define which actions may run in parallel, utilize it
8. Support templating in the manifest file. Not just putting ENV variables, but also be able to do conditionals, etc. *(docker-compose does not have it, but they recently came up with a [pretty good solution](https://github.com/docker/compose/issues/1377), which we may adopt soon as well)*

Contributing these features to docker-compose was also an option, but we decided to come up with own solution due the following reasons:

1. docker-machine is written in Python, we don't have tools in Python. Also it would be nice if the tool was written in Go to benefit from the existing ecosystem and to ease installations on development machines and any instance or CI server
2. We want to have a full control over the tool and can add any feature to it any time
3. Time factor was also critical, we were able to come up with a working solution in a four days

# How it works
The most notable feature of rocker-compose is **idempotency**. We have to be able to compare any bit of a container runtime, which includes configuration and state.
For every run, rocker-compose is building two data sets: **desired** and **actual**. "Desired" is the list of containers given in the manifest. "Actual" is the list of currently running containers we get through the Docker API. By [comparing the two sets](/src/compose/diff.go) and knowing the dependencies between the containers we are about to run, we build an [ordered action list](src/compose/action.go). You can also consider the action list as a *delta* between the two sets.

If a desired container does not exist, rocker-compose simply creates it (and optionally starts). For existing container with the same name (namespace does help here), it does a more sophisticated comparison:

1. **Compare configuration.** When starting a container, rocker-compose puts the serialized source YAML configuration to a label called "rocker-compose-config". By [comparing](/src/compose/config/compare.go) the source config from the manifest and the one stored in a running container label, rocker-compose can detect changes.
2. **Compare image id**. Rocker-compose also checks if the image id was changed. It may happen when you are using `:latest` tags, when image can be updated without changing the tag.
3. [Compare state](#state).

It allows rocker-compose to do **as fewer changes as possible** to make the actual state match the desired one. If something was changed, rocker-compose re-creates the container from scratch. Note that any container change can trigger re-creations of other containers depending on the first one.

**In case of loose coupling**, you can benefit from a micro-services approach and do clever updates, affecting only a single container, without touching others. See [patterns](#patterns) to know more about best practices.

# Production use
Rocker-compose isn't yet battle tested for production. However, it's intended to use for deployments, due its idempotent properties. The idea is that anything you do with rocker-compose on your local machine, you can do with remote machine by simply adding remote host parameters or having [appropriate ENV](https://docs.docker.com/reference/commandline/cli/#environment-variables). Rocker-compose implements docker's native client interface for connection parameterization.

```bash
$ rocker-compose run                              # gathers info about docker server from ENV
$ rocker-compose $(docker-machine config qa1) run # connects to qa1 server and runs there  
```

*NOTE: You should have qa1 machine registered in your docker-machine*

# Tutorial

Here is an [example of running a wordpress application](/example/wordpress.yml) with rocker-compose:
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

Assuming you have a virtual machine named `dev`, you can do:
```bash
$ open http://$(docker-machine ip dev):8080/
```

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

Where `main` is a container name and `image: wordpress` is its spec. If container name beginning with underscore `_` then rocker-compose will not consider it — useful for doing base specs for [extends](#extends). Note that by convension, properties should be maintained in the given order when writing compose manifests.

| Property | Default | Type | Run param | Description |
|----------|---------|------|-----------|-------------|
| **extends** | *nil* | String | *none* | `container_name` - extend spec from another container of current manifest |
| **image** | *REQUIRED* | String | `docker run <image>` | image name for the container, the syntax is `[registry/][repo/]name[:tag]` |
| **state** | `running` | String | *none* | `running`, `ran`, `created` - desired state of a container, [read more about state](#state) |
| **entrypoint** | *nil* | Array\|String | [`--entrypoint`](https://docs.docker.com/reference/run/#entrypoint-default-command-to-execute-at-runtime) | overwrite the default entrypoint set by the image |
| **cmd** | *nil* | Array\|String | `docker run <image> <cmd>` | the list of command arguments to pass |
| **workdir** | *nil* | String | [`-w`](https://docs.docker.com/reference/run/#workdir) | set working directory inside the container |
| **restart** | `always` | String | [`--restart`](https://docs.docker.com/reference/run/#restart-policies-restart) | `never`, `always`, `on-failure,N` - container restart policy |
| **labels** | *nil* | Hash\|String | `--label FOO=BAR` | key/value labels to add to a container |
| **env** | *nil* | Hash\|String | [`-e`](https://docs.docker.com/reference/run/#env-environment-variables) | key/value ENV variables |
| **wait_for** | *nil* | Array\|String | *none* | array of container names - wait for other containers to start before starting the container |
| **links** | *nil* | Array\|String | [`--link`](https://docs.docker.com/userguide/dockerlinks/) | other containers to link with; can be `container` or `container:alias` |
| **volumes_from** | *nil* | Array\|String | [`--volumes-from`](https://docs.docker.com/userguide/dockervolumes/) | mount volumes from another containers |
| **volumes** | *nil* | Array\|String | [`-v`](https://docs.docker.com/userguide/dockervolumes/) | specify volumes of a container, can be `path` or `src:dest` [read more](#volumes) |
| **expose** | *nil* | Array\|String | [`--expose`](https://docs.docker.com/articles/networking/) | expose a port or a range of ports from the container without publishing it to your host; e.g. `8080` or `8125/udp` |
| **ports** | *nil* | Array\|String | [`-p`](https://docs.docker.com/articles/networking/) | publish a container᾿s port or a range of ports to the host, e.g. `8080:80` or `0.0.0.0:8080:80` or `8125:8125/udp` |
| **publish_all_ports** | `false` | Bool | [`-P`](https://docs.docker.com/articles/networking/) | every port in `expose` will publish to a host |
| **dns** | *nil* | Array\|String | [`--dns`](https://docs.docker.com/reference/run/#network-settings) | add DNS servers to a container |
| **add_host** | *nil* | Array\|String | [`--add-host`](https://docs.docker.com/reference/run/#network-settings) | add records to `/etc/hosts` file, e.g. `mysql:172.17.3.21` |
| **net** | `bridge` | String | [`--net`](https://docs.docker.com/reference/run/#network-settings) | network mode, options are: `bridge`, `host`, `container:<name|id>`; `none` is to disable networking |
| **hostname** | *nil* | String | [`--hostname`](https://docs.docker.com/reference/run/#network-settings) | set custom hostname to a container |
| **domainname** | *nil* | String | [`--dns-search`](https://docs.docker.com/articles/networking/#configuring-dns) | set the search domain to `/etc/resolv.conf` |
| **user** | *nil* | String | [`-u`](https://docs.docker.com/reference/run/#user) | run container process with specified user or UID |
| **uts** | *nil* | String | [`--uts`](https://docs.docker.com/reference/run/#uts-settings-uts) | if set to `host` container will inherit host machine's hostname and domain; warning, **insecure**, use only with trusted containers |
| **pid** | *nil* | String | [`--pid`](https://docs.docker.com/reference/run/#pid-settings-pid) | set the PID (Process) Namespace mode for the container, when set to `host` will be in host machine's namespace |
| **privileged** | `false` | Bool | [`--privileged`](https://docs.docker.com/reference/run/#runtime-privilege-linux-capabilities-and-lxc-configuration) | give extended privileges to this container |
| **memory** | *nil* | String|Number | [`--memory`](https://docs.docker.com/reference/run/#runtime-constraints-on-resources) | `<number><unit>` limit memory for container where units are `b`, `k`, `m` or `g` |
| **memory_swap** | *nil* | String|Number | [`--memory-swap`](https://docs.docker.com/reference/run/#runtime-constraints-on-resources) | limit total memory (memory + swap), formar same as for **memory** |
| **cpu_shares** | *nil* | Number | [`--cpu-shares`](https://docs.docker.com/reference/run/#runtime-constraints-on-resources) | CPU shares (relative weight) |
| **cpu_period** | *nil* | Number | [`--cpu-period`](https://docs.docker.com/reference/run/#runtime-constraints-on-resources) | limit the CPU CFS (Completely Fair Scheduler) period |
| **cpuset_cpus** | *nil* | String | [`--cpuset-cpus`](https://docs.docker.com/reference/run/#runtime-constraints-on-resources) | CPUs in which to allow execution, e.g. `0-3` or `0,1` |
| **ulimits** | *nil* | Array of Ulimit | [`--ulimit`](https://github.com/docker/docker/pull/9437) | ulimit spec for container |
| **kill_timeout** | `0` | Number | *none* | timeout in seconds to wait container to [stop before killing it](https://docs.docker.com/reference/commandline/stop/) with `-9` |
| **keep_volumes** | `false` | Bool | *none* | tell rocker-compose to keep volumes when removing the container |

# State
For every pair of containers with a same name, rocker-compose does a comparison of all properties to figure out changes, as well as checking running state. To define, should the container be restarted, in case all other properties are equal, rocker-compose uses the following decision scheme:

| Desired State | Actual State | Exit Code | Action                 |
|---------------|--------------|-----------|------------------------|
| running       | not exist    | *none*    | start                  |
| running       | exist        | *any*     | remove and start       |
| running       | restarting   | *any*     | wait                   |
| running       | running      | *none*    | NOOP                   |
| created       | not exist    | *none*    | create                 |
| created       | exist        | *any*     | NOOP                   |
| created       | restarting   | *any*     | remove and create      |
| created       | running      | *none*    | remove and create      |
| ran           | not exist    | *none*    | start and wait         |
| ran           | exist        | `0`       | NOOP                   |
| ran           | exist        | non-zero  | remove, start and wait |
| ran           | restarting   | *any*     | wait                   |
| ran           | running      | *none*    | NOOP?                  |

*NOTE: by "start" here we mean "create" and then "start"*

**state: ran** is used for a single shot commands, for doing some initialization stuff. Rocker-compose do not re-run such containers unless they have changed or previous executions exited with non-zero code.

**state: created** is mostly used for data volume and network-share containers. They are described in [patterns](#patterns) section.

# Volumes
It is possible to mount volumes to a running containers same way as it is when using plain `docker run`. In Docker, there are two types of volumes: **Data volume** and **Mounted host directory**. 

### Data volume
"Data volume" is a reusable directory managed by Docker daemon, that can be shared between containers. Most often, this type of file sharing across containers should be used, because of its [12factor](http://12factor.net/) compliance — you can think of it as "data volume as a service".

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
    image: grammarly/scratch # use empty image, just for data
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
While it is useful for development and testing, is unsafe and error-prone for production use. It requires some external folder to exist on a host machine on order to run your container. Also, it may cause some unpleasant failure modes hard to reproduce. And finally, you cannot guarantee reproducibility of your manifests.

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
      # mount ./wordpress-src directory to /var/www/html in the container, such way we can hack wordpress sources while the container is running
      - ./wordpress-src:/var/www/html
    ports:
      - "8080:80"
```

*NOTE: you cannot use the last example for production, obviously because there should be no such directory as `./wordpress-src`*

# Extends
You can extend some container specifications within a single manifest file. In this example, we will run two identical wordpress containers and assign them to a different ports:
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

**NOTE:** nested extends are not allowed by rocker-compose.

# Templating
Rocker-compose uses Go's [text/template](http://golang.org/pkg/text/template/) engine to render manifests. This way you can put some logic to your manifests or even throw some variables from the outside:
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
      - {{ .port | default "8080" }}:80
```

You can run this manifest as follows:
```bash
$ rocker-compose run                             # will not mount src volume and run on 8080
$ rocker-compose run -var env=dev                # will mount src volume and run on :8080
$ rocker-compose run -var env=dev -var port=8081 # will mount src volume and run on :8081
```

In addition to the [builtin helper functions](http://golang.org/pkg/text/template/#hdr-Functions) there are few provided by rocker-compose:

###### {{ default *arg1* *arg2* }} or {{ *arg2* | default *arg1* }}
Returns the passed default value *arg1* if given value *arg2* is empty. By emptiness we mean any of `nil`, `[]`, `""` and `0`.

###### {{ bridgeIp }} [Example](#loose-coupling-network)
Returns Docker's [bridge gateway ip](https://docs.docker.com/articles/networking/), which can be used to access any exposed ports of an external container. Useful for loose coupling. [Source](https://github.com/grammarly/rocker-compose/blob/88007dcf571da7617f775c9abe1824eedc9598fb/src/compose/docker.go#L59)

# Patterns
Here is the list of the most common problems with multi-container applications and ways how you can solve it with rocker-compose.

### Data volume containers
By design, containers are transient. Most of the tools for containerized applications are built expecting your apps to respect this rule. Your container can be dropped and created from scratch any time. For example, to update the image some container is running, you have to remove container and create an new one. This is a property of **immutable infrastructure**. In Docker, every container have its own dedicated file system by default it is removed along with the container. There is `VOLUME` directive, which creates a separate data volume associated with the container and is able to stay alive after container removal. But there is no way to re-associate the old detached volume with a new container.

A known pattern to workaround containers transient properties while not losing persistent data is to make a "data volume container" and mount it's volumes to your application container.
```yaml
namespace: wordpress
containers:
  db:
    image: mysql:5.6
    env: MYSQL_ROOT_PASSWORD=example
    volumes_from:
      # db container can be easily re-created without losing data
      # all data will be remained in /var/lib/mysql associated with db_data container
      - db_data 

  db_data:
    image: grammarly/scratch # use empty image, just for data
    state: created # this tells compose to not try to run this container, data containers needs to be just created
    volumes:
      # define the empty directory that will be used by "db" container
      - /var/lib/mysql
```

Another reason why you can use this pattern is when you want to split the lifecycle between your application and its configuration:
```yaml
namespace: wordpress
containers:
  main:
    image: wordpress:4.1.2
    volumes_from:
      - main_config

  main_config:
    image: my_wordpress_config
    state: created
```

This way, you can release `my_wordpress_config` independently from `wordpress` and deliver it separately. Keep in ming though that `main` container will be restarted any time `main_config` changes.

Dockerfile if the `my_wordpress_config` image might look like the following:
```bash
FROM scratch
ADD ./config.json /etc/wordpress/config.json  # add config file from the context directory to the image
VOLUME /etc/wordpress                         # declare /etc/wordpress to be shareable
```

The directory `/etc/wordpress` will show in `main` container when it is started.

### Bootstrapping
Sometimes you want to do some initialization proir to start your application container. For example, you may want to create an initial user for a database. [Most of times](https://github.com/docker-library/mysql/blob/master/5.6/docker-entrypoint.sh), people do some sort of `./docker-entrypoint.sh` wrapping entrypoint script, which is executed as a main process in a container. It does some checks and initialization and then runs an actual process.

There are situations when you are using some vendor image and do not want to extend from it or modify it in any way. In this case, you can use a single-run container semantics:
```yaml
namespace: sensu
containers:
  sensu_client:
    image: sensu-client
    links: rabbitmq
    wait_for:
      - bootstrap # wait for `bootstrap` container finish

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
This may look an ugly example, but [`state:ran`](#state) and [`wait_for`](#container-properties) are useful primitives that can be used in other cases as well.

Here the start order will be the following:

1. `rabbitmq` will go first, because it does not have any dependencies
2. `bootstrap` will run it's curl command
3. `sensu_client` will be started when `bootstrap` finished successfully, "sensu" user will be already created

### Loose coupling: network
The most correct way of linking Docker containers between each other is using [`--link`](https://docs.docker.com/userguide/dockerlinks/) primitive. However, in case if container A is linked to a container B, then A needs to be re-created every time we update B. In case you rely on ENV variables provided by links functionality, [you cannot update dependency](https://docs.docker.com/userguide/dockerlinks/#important-notes-on-docker-environment-variables) without re-creating your container.

Docker also automatically populates entries to `/etc/hosts` inside your container. It also updates hosts entires when dependencies **restart**, but not when you **re-create** them — what commonly happens when you update underlying images. Accessing links though `/etc/hosts` does not also help loose coupling because you need B to exist anyway in order to even start A.

Both approaches are considered tight coupling and it's ok using it as long as you acknowledge that fact.

Sometimes you want to detach your application container from some dependency. Let's assume you have an app which is writing metrics to [StatsD](https://github.com/etsy/statsd) daemon by UDP, and you don't care if the daemon is present the time you start your application. 

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

# in another manifest, managed by other team
namespace: platform
containers:
  # container B
  statsd:
    image: statsd
    ports:
      # this will throw 8125/udp port and it will be available
      # on the Docker's bridge ip
      - "8125:8125/udp"
```

This way, `myapp.main` can run independently from `platform.statsd` and at the same time be able to write metrics to `statsd:8125`. In case statsd is not present, metrics will be simply dropped on the floor.

This is a tradeoff because you have to expose a known port to a host network. Ports may clash, you don't have a full isolation here and benefit from random port mapping. In every particular situation you have to balance between loose coupling and isolation.

### Network share
In case you have containers A and B from example above in the same manifest, you can do loose coupling without exposing any global ports to a host network. The trick is using `net: container:<id|name>` feature:

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

  # simple container that hang in 'while true' forever
  dummy:
    image: grammarly/net
```

In this example `dummy` container plays role of a network host. Containers that connected to it by `net: container:dummy` share all ports between each other.

**Note** that ports now may clash between container A and B since they now are in the same network.

**Keep in mind** that you cannot mix links with net:conainer mode.

### Loose coupling: files

TODO

# Contributing

### Dependencies

Use [gb](http://getgb.io/) to test and build. We vendor all dependencies, you can find them under `/vendor` directory.

Please, use [gofmt](https://golang.org/cmd/gofmt/) in order to automatically re-format Go code into vendor standartised convension. Ideally, you have to set it on post-save action in your IDE. For SublimeText3, [GoSublime](https://github.com/DisposaBoy/GoSublime) package makes it right.

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

```bash
grep -R TODO **/*.go | grep -v '^vendor/'
```

# Authors

- Yura Bogdanov <yuriy.bogdanov@grammarly.com>
- Stas Levental <stas.levental@grammarly.com>
