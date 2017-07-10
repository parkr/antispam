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

test: bundle
	go test github.com/parkr/antispam
