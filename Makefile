# Static binaries are where it's at!
CGO_ENABLED=0

# I'm waiting till the discussion pans out on vendoring. (https://github.com/golang/go/issues/14417 and others)
#
# I personally support the idea that vendor directories should *not* be tested (i.e. that the vendor directory
# should be named _vendor), as I like go test ./... as an idiom, but don't want to, and more importantly can't
# reliably run the tests for all my dependencies as we can't reliably pull in the transitive closure of all
# the dependencies of our dependencies at working versions - so running tests on dependencies opens us up to random
# flakes (for stuff we don't care about).
#
# The following 4 lines basically put back Go <= 1.5 gom behavior, using _vendor (which is then ignored in tests).
GO15VENDOREXPERIMENT=0
GOM_VENDOR_NAME=_vendor
export GO15VENDOREXPERIMENT
export GOM_VENDOR_NAME

TRAVIS_BUILD_NUMBER?=debug0

.PHONY: coverage get test clean

all: _vendor coverage AWSnycast

_vendor: Gomfile
	gom install || true

_vendor/src/github.com/stretchr/testify/assert: Gomfile
	gom -test install

AWSnycast: *.go */*.go _vendor
	gom build -a -tags netgo -ldflags '-w' .

test: _vendor/src/github.com/stretchr/testify/assert
	gom test -short ./...

fmt:
	go fmt ./...

coverage: _vendor/src/github.com/stretchr/testify/assert
	gom test -cover -short ./...

integration: _vendor/src/github.com/stretchr/testify/assert
	gom test ./...

clean:
	rm -rf dist */coverage.out */coverprofile.out coverage.out coverprofile.out AWSnycast _vendor
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

itest_%:
	mkdir -p dist
	make -C package itest_$*

Gemfile.lock:
	bundle install

dist: AWSnycast Gemfile.lock
	rm -rf dist/ *.deb
	strip AWSnycast
	bundle exec fpm -s dir -t deb --name awsnycast --url "https://github.com/bobtfish/AWSnycast" --maintainer "Tomas Doran <bobtfish@bobtfish.net>" --description "Anycast in AWS" --license Apache2 --iteration $(TRAVIS_BUILD_NUMBER) --version $$(./AWSnycast -version) --prefix /usr/bin AWSnycast
	bundle exec fpm -s dir -t rpm --name awsnycast --url "https://github.com/bobtfish/AWSnycast" --maintainer "Tomas Doran <bobtfish@bobtfish.net>" --description "Anycast in AWS" --license Apache2 --iteration $(TRAVIS_BUILD_NUMBER) --version $$(./AWSnycast -version) --prefix /usr/bin AWSnycast
	mkdir dist
	mv *.deb *.rpm dist/
