.PHONY: build clean test package serve run-compose-test
PKGS := $(shell go list ./... | grep -v /vendor/)
#VERSION := $(shell git describe --always)
VERSION := $(shell date "+%Y-%m-%d:%H:%M:%S")
#GOOS ?= linux
GOOS ?= darwin
GOARCH ?= amd64

build:
	@echo "Compiling source for $(GOOS) $(GOARCH)"
	@mkdir -p build
	@GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=0 go build -a -installsuffix cgo -ldflags "-X main.version=$(VERSION)" -o build/rump$(BINEXT) rump.go

clean:
	@echo "Cleaning up workspace"
	@rm -rf build

build-other:
	@mkdir -p build
	@echo "Compiling source for darwin amd64"
	@GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -a -installsuffix cgo -ldflags "-w -X main.version=$(VERSION)" -o build/rump_darwin$(BINEXT) rump.go
	@echo "Compiling source for linux amd64"
	@GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -a -installsuffix cgo -ldflags "-s -X main.version=$(VERSION)" -o build/rump_linux$(BINEXT) rump.go
	@echo "Compiling source for Windows amd64"
	@GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -a -installsuffix cgo -ldflags "-s -X main.version=$(VERSION)" -o build/rump_win64.exe$(BINEXT) rump.go
