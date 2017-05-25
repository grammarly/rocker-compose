VERSION ?= $(shell cat VERSION)

GITCOMMIT := $(shell git rev-parse HEAD 2>/dev/null)
GITBRANCH := $(shell git rev-parse --abbrev-ref HEAD 2>/dev/null)
BUILDTIME := $(shell TZ=GMT date "+%Y-%m-%d_%H:%M_GMT")

SRCS = $(shell find . -name '*.go' | grep -v '^./vendor/')
PKGS := $(foreach pkg, $(sort $(dir $(SRCS))), $(pkg))

TESTARGS ?=

default:
	GO15VENDOREXPERIMENT=1 go build -v

install:
	cp rocker-compose /usr/local/bin/rocker-compose
	chmod +x /usr/local/bin/rocker-compose

cross: dist_dir
	docker run --rm -ti -v $(shell pwd):/go/src/github.com/snkozlov/rocker-compose \
		-e GOOS=linux -e GOARCH=amd64 -e GO15VENDOREXPERIMENT=1 -e GOPATH=/go \
		-w /go/src/github.com/snkozlov/rocker-compose \
		golang go build \
		-ldflags "-X main.Version=$(VERSION) -X main.GitCommit=$(GITCOMMIT) -X main.GitBranch=$(GITBRANCH) -X main.BuildTime=$(BUILDTIME)" \
		-v -o ./dist/linux_amd64/rocker-compose

	docker run --rm -ti -v $(shell pwd):/go/src/github.com/snkozlov/rocker-compose \
		-e GOOS=darwin -e GOARCH=amd64 -e GO15VENDOREXPERIMENT=1 -e GOPATH=/go \
		-w /go/src/github.com/snkozlov/rocker-compose \
		golang go build \
		-ldflags "-X main.Version=$(VERSION) -X main.GitCommit=$(GITCOMMIT) -X main.GitBranch=$(GITBRANCH) -X main.BuildTime=$(BUILDTIME)" \
		-v -o ./dist/darwin_amd64/rocker-compose

cross_tars: cross
	COPYFILE_DISABLE=1 tar -zcvf ./dist/rocker-compose_linux_amd64.tar.gz -C dist/linux_amd64 rocker-compose
	COPYFILE_DISABLE=1 tar -zcvf ./dist/rocker-compose_darwin_amd64.tar.gz -C dist/darwin_amd64 rocker-compose

dist_dir:
	mkdir -p ./dist/linux_amd64
	mkdir -p ./dist/darwin_amd64

clean:
	rm -Rf dist

testdeps:
	@ go get github.com/GeertJohan/fgt

fmtcheck:
	$(foreach file,$(SRCS),gofmt $(file) | diff -u $(file) - || exit;)

lint:
	@ go get github.com/golang/lint/golint
	$(foreach file,$(SRCS),fgt golint $(file) || exit;)

vet:
	$(foreach pkg,$(PKGS),fgt go vet $(pkg) || exit;)

gocyclo:
	@ go get github.com/fzipp/gocyclo
	gocyclo -over 25 ./src

test: testdeps fmtcheck lint vet
	GO15VENDOREXPERIMENT=1 go test ./src/... $(TESTARGS)

version:
	@echo $(VERSION)

.PHONY: clean test fmtcheck lint vet gocyclo version testdeps cross cross_tars dist_dir default install

