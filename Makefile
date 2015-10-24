.PHONY: coverage get test

all: get coverage AWSnycast

AWSnycast: *.go */*.go
	go build .

test:
	go test ./...

get:
	go get ./...

coverage:
	go test -cover ./...

coverage.out:
	cd aws ; go test -coverprofile=coverage.out ./... ; cd ..
	cd daemon ; go test -coverprofile=coverage.out ./... ; cd ..
	cd config ; go test -coverprofile=coverage.out ./... ; cd ..
	echo "mode: set" > coverage.out && cat */coverage.out | grep -v mode: | sort -r | awk '{if($$1 != last) {print $$0;last=$$1}}' >> coverage.out
