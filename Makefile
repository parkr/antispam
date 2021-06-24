REV:=$(shell git rev-parse HEAD)

all: build test

golint:
	go get golang.org/x/lint/golint

.PHONY: statik
statik:
	go get github.com/rakyll/statik

bundle: statik
	statik -src=$(shell pwd)/blacklists

build: bundle
	go install ./...

test: bundle vet lint
	go test ./...

vet:
	go vet ./...

lint: golint
	golint ./...

docker-build:
	docker build -t parkr/antispam:$(REV) .

docker-release: docker-build
	docker push parkr/antispam:$(REV)
