CGO_ENABLED=0
TRAVIS_BUILD_NUMBER?=debug0

.PHONY: coverage get test clean

all: get coverage AWSnycast

_vendor: Gomfile
	gom install

AWSnycast: *.go */*.go _vendor
	gom build -a -tags netgo -ldflags '-w' .
	strip AWSnycast

test: _vendor
	gom test -short ./...

fmt:
	go fmt ./...

coverage:
	gom test -cover -short ./...

integration:
	gom test ./...

clean:
	rm -rf dist */coverage.out */coverprofile.out coverage.out coverprofile.out AWSnycast _vendor

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
	rm -rf dist/ *.deb
	bundle exec fpm -s dir -t deb --name awsnycast --url "https://github.com/bobtfish/AWSnycast" --maintainer "Tomas Doran <bobtfish@bobtfish.net>" --description "Anycast in AWS" --license Apache2 --iteration $(TRAVIS_BUILD_NUMBER) --version $$(./AWSnycast -version) --prefix /usr/bin AWSnycast
	bundle exec fpm -s dir -t rpm --name awsnycast --url "https://github.com/bobtfish/AWSnycast" --maintainer "Tomas Doran <bobtfish@bobtfish.net>" --description "Anycast in AWS" --license Apache2 --iteration $(TRAVIS_BUILD_NUMBER) --version $$(./AWSnycast -version) --prefix /usr/bin AWSnycast
	mkdir dist
	mv *.deb *.rpm dist/
