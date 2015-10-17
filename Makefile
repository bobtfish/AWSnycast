.PHONY: coverage get test

all: AWSnycast

AWSnycast: *.go */*.go
	go build .

test:
	go test ./...

get:
	go get ./...

coverage:
	go test -cover ./...
