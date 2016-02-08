#!/bin/bash

set -e

VERSION=`cat VERSION`
LAST_TAG=`git describe --abbrev=0 --tags 2>/dev/null`

GITHUB_USER=grammarly
GITHUB_REPO=rocker-compose

docker run --rm -ti \
  -e GITHUB_TOKEN=$GITHUB_TOKEN \
  -v /etc/ssl/certs/ca-certificates.crt:/etc/ssl/certs/ca-certificates.crt \
  -v `pwd`/dist:/dist \
  dockerhub.grammarly.io/tools/github-release:master release \
      --user $GITHUB_USER \
      --repo $GITHUB_REPO \
      --tag $VERSION \
      --name $VERSION \
      --description "https://github.com/$GITHUB_USER/$GITHUB_REPO/compare/$LAST_TAG...$VERSION"

docker run --rm -ti \
  -e GITHUB_TOKEN=$GITHUB_TOKEN \
  -v /etc/ssl/certs/ca-certificates.crt:/etc/ssl/certs/ca-certificates.crt \
  -v `pwd`/dist:/dist \
  dockerhub.grammarly.io/tools/github-release:master upload \
      --user $GITHUB_USER \
      --repo $GITHUB_REPO \
      --tag $VERSION \
      --name rocker-compose-$VERSION-linux_amd64.tar.gz \
      --file ./dist/rocker-compose_linux_amd64.tar.gz

docker run --rm -ti \
  -e GITHUB_TOKEN=$GITHUB_TOKEN \
  -v /etc/ssl/certs/ca-certificates.crt:/etc/ssl/certs/ca-certificates.crt \
  -v `pwd`/dist:/dist \
  dockerhub.grammarly.io/tools/github-release:master upload \
      --user $GITHUB_USER \
      --repo $GITHUB_REPO \
      --tag $VERSION \
      --name rocker-compose-$VERSION-darwin_amd64.tar.gz \
      --file ./dist/rocker-compose_darwin_amd64.tar.gz
