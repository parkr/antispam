REV:=$(shell git rev-parse HEAD)
CONTAINER_TAG=parkr/antispam:$(REV)

all: build test

golint:
	go get golang.org/x/lint/golint

.PHONY: statik
statik:
	go get github.com/rakyll/statik

bundle: statik
	statik -f -src=$(shell pwd)/blocklists

build: bundle *.go
	go install ./...
	go build ./...
	go build .

test: bundle vet lint
	go test ./...

vet:
	go vet ./...

lint: golint
	golint ./...

clean:
	rm -f statik/statik.go
	rm -f antispam

dive: docker-build
	dive $(CONTAINER_TAG)

docker-build: Dockerfile *.go
	docker build -t $(CONTAINER_TAG) .

docker-test: docker-build
	docker run $(CONTAINER_TAG) -h

docker-release: docker-build
	docker push $(CONTAINER_TAG)
