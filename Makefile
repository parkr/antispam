REV:=$(shell git rev-parse HEAD)

all: build test

golint:
	go get github.com/golang/lint/golint

.PHONY: statik
statik:
	go get github.com/rakyll/statik

bundle: statik
	statik -src=$(shell pwd)/blacklists

build: bundle
	go install github.com/parkr/antispam

test: bundle vet lint
	go test github.com/parkr/antispam

vet:
	go vet github.com/parkr/antispam

lint: golint
	golint github.com/parkr/antispam

docker-build:
	docker build -t parkr/antispam:$(REV) .

docker-release: docker-build
	docker push parkr/antispam:$(REV)
