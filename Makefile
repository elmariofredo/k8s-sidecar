APP_NAME_TOOL  := amtool
APP_NAME  := sidecar
NAMESPACE := sysincz
IMAGE := $(NAMESPACE)/$(APP_NAME)
REGISTRY := docker.io
ARCH := linux 
VERSION := $(shell git describe --tags 2>/dev/null)
ifeq "$(VERSION)" ""
VERSION := $(shell git rev-parse --short HEAD)
endif
COMMIT=$(shell git rev-parse --short HEAD)
BRANCH=$(shell git rev-parse --abbrev-ref HEAD)
BUILD_DATE=$(shell date +%FT%T%z)
LDFLAGS = -ldflags "-X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.Branch=$(BRANCH) -X main.BuildDate=$(BUILD_DATE)"

.PHONY: clean

clean:
	rm -rf bin/$(APP_NAME)
	rm -rf bin/$(APP_NAME_TOOL)

dep:
	go get -v ./...

build: clean dep 
	GOOS=$(ARCH) GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -a -installsuffix cgo -o bin/$(APP_NAME) ./cmd/$(APP_NAME)
	GOOS=$(ARCH) GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -a -installsuffix cgo -o bin/$(APP_NAME_TOOL) ./cmd/$(APP_NAME_TOOL)

docker: build
	docker build . -t $(IMAGE):$(VERSION)


docker-push: docker
	docker push $(IMAGE):$(VERSION)


debug: build-image
	docker run -p 2112:2112/tcp --network host -v "/tmp/config:/config" --rm --name $(APP_NAME) $(IMAGE):$(VERSION) -debug

debug_bash: build-image
	docker run -it -p 2112:2112/tcp -e SNMP_SERVER=localhost --network host --rm --entrypoint "/bin/bash" --name $(APP_NAME) $(IMAGE):$(VERSION) 
