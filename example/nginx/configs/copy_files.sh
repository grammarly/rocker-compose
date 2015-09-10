#!/usr/bin/env sh
set -ex

# Copy the files from /src to the corresponding directories that are
# in the "shared" data volume container where the "nginx" container can read them
rsync -av --delete /src/conf.d/ /etc/nginx/conf.d
rsync -av --delete /src/html/ /usr/share/nginx/html

# Because the "nginx" container waits for "configs" (ourselves) to finish,
# on the first run there will be nothing to reload.
if [ $(docker ps -qf name="nginx.nginx" | wc -l) -gt 0 ]; then
  # Use `docker exec` to reach nginx container, test the configuration
  # and then do the reload
  docker exec nginx.nginx nginx -t
  docker exec nginx.nginx nginx -s reload
fi
