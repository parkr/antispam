REV=$(shell git rev-parse HEAD)

all: build test

godep:
	go get github.com/tools/godep

golint:
	go get github.com/golang/lint/golint

.PHONY: statik
statik:
	go get github.com/rakyll/statik

bundle: godep statik
	godep save github.com/parkr/antispam
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
