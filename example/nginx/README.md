# rocker-compose nginx example

As noted in the [blog post](http://tech.grammarly.com/blog/posts/How-We-Deploy-Containers-at-Grammarly.html):

You might want to decouple configuration to be able to iterate **without restarting nginx**. While deploying the nginx application itself [is not a big deal](https://www.nginx.com/blog/deploying-nginx-nginx-plus-docker/), delivering configuration gets trickier if you want to do it with containers only. You might also use tools like [docker-gen](https://github.com/jwilder/docker-gen) (container) to notify an nginx container to reload, but you will likely describe both containers in a single app manifest, which requires the deployment tool to support granularity.

This example demostrates how to use `rocker-compose` to make a decoupled nginx deployment.

To start, see [compose.yml](/example/nginx/compose.yml).

Also, see [grammarly/rsync-docker](https://hub.docker.com/r/grammarly/rsync-docker/ for more documentation) image, which is used as a base image for files delivery.
