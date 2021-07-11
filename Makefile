REV:=$(shell git rev-parse HEAD)

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

test: bundle vet lint
	go test ./...

vet:
	go vet ./...

lint: golint
	golint ./...

clean:
	rm -f statik/statik.go

dive: docker-build
	dive parkr/antispam:$(REV)

docker-build: Dockerfile *.go
	docker build -t parkr/antispam:$(REV) .

docker-test: docker-build
	docker run parkr/antispam:$(REV) -h

docker-release: docker-build
	docker push parkr/antispam:$(REV)
