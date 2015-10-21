.PHONY: coverage get test

all: get test AWSnycast

AWSnycast: *.go */*.go
	go build .

test:
	go test ./...

get:
	go get ./...

coverage:
	go test -cover ./...
