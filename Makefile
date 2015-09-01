VERSION := 0.1.0

OSES := linux darwin windows
ARCHS := amd64
BINARIES := rocker-compose

LAST_TAG = $(shell git describe --abbrev=0 --tags 2>/dev/null)
GITCOMMIT := $(shell git rev-parse HEAD 2>/dev/null)
GITBRANCH := $(shell git rev-parse --abbrev-ref HEAD 2>/dev/null)
BUILDTIME := $(shell date "+%Y-%m-%d %H:%M GMT")

GITHUB_USER := grammarly
GITHUB_REPO := rocker-compose
GITHUB_RELEASE := docker run --rm -ti \
											-e GITHUB_TOKEN=$(GITHUB_TOKEN) \
											-v /etc/ssl/certs/ca-certificates.crt:/etc/ssl/certs/ca-certificates.crt \
											-v $(shell pwd)/dist:/dist \
											dockerhub.grammarly.io/tools/github-release:master

ALL_ARCHS := $(foreach os, $(OSES), $(foreach arch, $(ARCHS), $(os)/$(arch) ))
ALL_BINARIES := $(foreach arch, $(ALL_ARCHS), $(foreach bin, $(BINARIES), dist/$(VERSION)/$(arch)/$(bin) ))
OUT_BINARIES := $(foreach arch, $(ALL_ARCHS), $(foreach bin, $(BINARIES), dist/$(bin)_$(subst /,_,$(arch)) ))
ALL_TARS := $(ALL_BINARIES:%=%.tar.gz)

os = $(shell echo "$(1)" | awk -F/ '{print $$3}' )
arch = $(shell echo "$(1)" | awk -F/ '{print $$4}' )
bin = $(shell echo "$(1)" | awk -F/ '{print $$5}' )

UPLOAD_CMD = $(GITHUB_RELEASE) upload \
			--user $(GITHUB_USER) \
			--repo $(GITHUB_REPO) \
			--tag $(VERSION) \
			--name $(call bin,$(FILE))-$(VERSION)_$(call os,$(FILE))_$(call arch,$(FILE)).tar.gz \
			--file $(FILE).tar.gz

all: $(ALL_BINARIES)
	$(foreach BIN, $(BINARIES), $(shell cp dist/$(VERSION)/$(shell go env GOOS)/amd64/$(BIN) dist/$(BIN)))

$(OUT_BINARIES): $(ALL_BINARIES)
	cp $< $@

release: $(ALL_TARS)
	git pull
	git push && git push --tags
	$(GITHUB_RELEASE) release \
			--user $(GITHUB_USER) \
			--repo $(GITHUB_REPO) \
			--tag $(VERSION) \
			--name $(VERSION) \
			--description "https://github.com/$(GITHUB_USER)/$(GITHUB_REPO)/compare/$(LAST_TAG)...$(VERSION)"
	$(foreach FILE,$(ALL_BINARIES),$(UPLOAD_CMD);)

tar: $(ALL_TARS)

%.tar.gz: %
	COPYFILE_DISABLE=1 tar -jcvf $@ -C dist/$(VERSION)/$(call os,$<)/$(call arch,$<) $(call bin,$<)

$(ALL_BINARIES): build_image
	docker run --rm -ti -v $(shell pwd)/dist:/src/dist \
		-e GOOS=$(call os,$@) -e GOARCH=$(call arch,$@) -e GOPATH=/src:/src/vendor \
		rocker-compose-build:latest go build \
		-ldflags "-X main.Version '$(VERSION)' -X main.GitCommit '$(GITCOMMIT)' -X main.GitBranch '$(GITBRANCH)' -X main.BuildTime '$(BUILDTIME)'" \
		-v -o $@ src/cmd/$(call bin,$@)/main.go

build_image:
	rocker build

clean:
	rm -Rf dist

.PHONY: clean build_image
