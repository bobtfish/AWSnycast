.PHONY: coverage get test clean

all: get coverage AWSnycast

AWSnycast: *.go */*.go
	godep go build -a -tags netgo -ldflags '-w'  .

test:
	godep go test -short ./...

get:
	CGO_ENABLED=0 go get -a -x -installsuffix cgo -ldflags '-d -s -w' && godep go install -a -x -installsuffix cgo -ldflags '-d -s -w'

coverage:
	godep go test -cover -short ./...

integration:
	godep go test ./...

clean:
	rm -rf dist */coverage.out */coverprofile.out coverage.out coverprofile.out AWSnycast

realclean: clean
	make -C tests/integration realclean

coverage.out:
	cd aws ; go test -coverprofile=coverage.out ./... ; cd ..
	cd daemon ; go test -coverprofile=coverage.out ./... ; cd ..
	cd config ; go test -coverprofile=coverage.out ./... ; cd ..
	cd healthcheck ; go test -coverprofile=coverage.out ./... ; cd ..
	cd instancemetadata ; go test -coverprofile=coverage.out ./... ; cd ..
	echo "mode: set" > coverage.out && cat */coverage.out | grep -v mode: | sort -r | awk '{if($$1 != last) {print $$0;last=$$1}}' >> coverage.out

itest_%:
	make -C package itest_$*

Gemfile.lock:
	bundle install

dist: AWSnycast Gemfile.lock
	rm -rf bin/ dist/ *.deb
	mkdir bin ; cp AWSnycast bin/
	bundle exec fpm -s dir -t deb --name awsnycast --iteration 1 --version 0.0.5 --prefix /usr/ ./bin/
	rm -rf bin
	mkdir dist
	mv *.deb dist/
