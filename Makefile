GOBUILD := GOPATH=$(shell pwd):$(shell pwd)/vendor go build -ldflags "-X main.Version $(VERSION) -X main.GitCommit $(GITCOMMIT) -X main.GitBranch $(GITBRANCH)" -v

tar: compile
	tar -czvf dist/rocker-compose-$(VERSION)_linux_amd64.tar.gz -C dist/linux_amd64 rocker-compose
	tar -czvf dist/rocker-compose-$(VERSION)_darwin_amd64.tar.gz -C dist/darwin_amd64 rocker-compose

compile:
	GOOS=linux GOARCH=amd64 $(GOBUILD) -o dist/linux_amd64/rocker-compose src/cmd/rocker-compose/main.go
	GOOS=darwin GOARCH=amd64 $(GOBUILD) -o dist/darwin_amd64/rocker-compose src/cmd/rocker-compose/main.go

install:
	cp dist/$(shell go env GOOS)_amd64/rocker-compose /usr/local/bin/rocker-compose
	chmod +x /usr/local/bin/rocker-compose

clean:
	rm -Rf dist

