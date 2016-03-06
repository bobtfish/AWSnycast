
TRAVIS_BUILD_NUMBER?=debug0

AWSNYCAST      := AWSnycast
SRCDIR         := src

.PHONY: coverage get test clean

all: get coverage AWSnycast

PKGS           := \
	$(AWSNYCAST)/\
	$(AWSNYCAST)/aws \
	$(AWSNYCAST)/config \
	$(AWSNYCAST)/daemon \
	$(AWSNYCAST)/healthcheck \
	$(AWSNYCAST)/instancemetadata \
	$(AWSNYCAST)/utils

SOURCES        := $(foreach pkg, $(PKGS), $(wildcard $(SRCDIR)/$(pkg)/*.go))

GOPATH  := $(shell pwd -L)
export GOPATH

PATH := bin:$(PATH)
export PATH

CGO_ENABLED := 0
export CGO_ENABLED

AWSnycast: $(SOURCES)
	@echo Building $(AWSNYCAST)...
	bin/gom build -a -tags netgo -ldflags '-w' -o bin/$(AWSNYCAST) $@
	strip bin/$(AWSNYCAST)

test:
	bin/gom test -short ./...

get:
	@echo Getting dependencies...
	@go get github.com/mattn/gom
	@bin/gom install

fmt:
	bin/gom fmt ./...

coverage:
	bin/gom test -cover -short ./...

integration:
	bin/gom test ./...

clean:
	rm -rf dist */coverage.out */coverprofile.out coverage.out coverprofile.out AWSnycast

realclean: clean
	make -C tests/integration realclean

coverage.out:
	@$(foreach pkg, $(PKGS), bin/gom test -coverprofile=coverage.out $(pkg);)
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
