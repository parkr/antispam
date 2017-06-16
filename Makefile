all: build test

build:
	go install github.com/parkr/antispam

test:
	go test github.com/parkr/antispam
