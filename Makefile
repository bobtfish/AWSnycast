# Static binaries are where it's at!
CGO_ENABLED=0

TRAVIS_BUILD_NUMBER?=debug0

.PHONY: coverage get test clean

all: AWSnycast

AWSnycast: *.go */*.go
	go get ./...
	go build -a -tags netgo -ldflags '-w' .

test:
	go test -short ./...

fmt:
	go fmt ./...

coverage:
	go test -cover -short ./...

integration:
	go test ./...

clean:
	rm -rf dist */coverage.out */coverprofile.out coverage.out coverprofile.out AWSnycast
	make -C package clean

realclean: clean
	make -C tests/integration realclean

coverage.out:
	cd aws ; go test -coverprofile=coverage.out ./... ; cd ..
	cd daemon ; go test -coverprofile=coverage.out ./... ; cd ..
	cd config ; go test -coverprofile=coverage.out ./... ; cd ..
	cd healthcheck ; go test -coverprofile=coverage.out ./... ; cd ..
	cd instancemetadata ; go test -coverprofile=coverage.out ./... ; cd ..
	echo "mode: set" > coverage.out && cat */coverage.out | grep -v mode: | sort -r | awk '{if($$1 != last) {print $$0;last=$$1}}' >> coverage.out

itest_%: AWSnycast
	mkdir -p dist
	make -C package itest_$*

dist: AWSnycast
	rm -rf dist/ *.deb
	strip AWSnycast
	docker run --rm -t --interactive -v $PWD:/awsnycast -w /awsnycast --entrypoint="" goreleaser/nfpm  /usr/local/bin/nfpm package --config nfpm.yaml --target AWSnycast.deb
	mkdir dist
	mv *.deb *.rpm dist/
