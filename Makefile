all: build test

godep:
	go get github.com/tools/godep

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

lint:
	go get github.com/golang/lint/golint
	golint github.com/parkr/antispam
