all: build test

bundle:
	godep save github.com/parkr/antispam
	statik -src=$(shell pwd)/blacklists

build: bundle
	go install github.com/parkr/antispam

test: bundle
	go test github.com/parkr/antispam
