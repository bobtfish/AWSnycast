.PHONY: itest_% clean shell
PROJECT = awsnycast


BUILD_NUMBER?=0
ifdef upstream_build_number
	REAL_BUILD_NUMBER=$(upstream_build_number)
else
	REAL_BUILD_NUMBER?=$(BUILD_NUMBER)
endif
VERSION = 0.2.1
ITERATION = yelp$(REAL_BUILD_NUMBER)
ARCH := $(shell facter architecture)

PACKAGE_NAME := $(PROJECT)_$(VERSION)-$(ITERATION)_$(ARCH).deb
PACKAGE_FILE := dist/$(PACKAGE_NAME)

DOCKER_TAG = $(PROJECT)_$(shell date +%s)

itest_%: $(PACKAGE_FILE)
	docker run --rm -v $(CURDIR)/../dist:/dist:ro -v $(CURDIR)/itest.sh:/itest.sh:ro docker-dev.yelpcorp.com/$*_yelp:latest bash /itest.sh /$(PACKAGE_FILE)

$(PACKAGE_FILE): 
	docker run --rm -t -e VERSION=$(VERSION)-$(ITERATION) --interactive -v $$PWD/..:/awsnycast -w /awsnycast --entrypoint="" goreleaser/nfpm  /usr/local/bin/nfpm package --config nfpm.yaml --target $(PACKAGE_FILE)

clean:
	$(CLEAN_CONTAINER)
	rm -f .docker_image_id
	rm -rf ../dist

shell: .docker_image_id
	docker run --rm -t -i $$(cat .docker_image_id) /bin/bash
