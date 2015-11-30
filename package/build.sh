#!/bin/bash

project=$1
version=$2
iteration=$3

export PATH=$PATH:/go/bin:/usr/local/go/bin

cd /go/src/github.com/bobtfish/AWSnycast
CGO_ENABLED=0 go get -a -x -installsuffix cgo -ldflags '-d -s -w' && godep go install -a -x -installsuffix cgo -ldflags '-d -s -w'
CGO_ENABLED=0 godep go build -a -x -installsuffix cgo -ldflags '-d -s -w' .
strip AWSnycast
mkdir /dist && cd /dist
fpm -s dir -t deb --name ${project} \
    --iteration ${iteration} --version ${version} \
    --prefix /usr/bin \
    /go/bin/AWSnycast

