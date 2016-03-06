#!/bin/bash

project=$1
version=$2
iteration=$3

export PATH=$PATH:/go/bin:/usr/local/go/bin

cd /go/src/github.com/bobtfish/AWSnycast
CGO_ENABLED=0 go get
CGO_ENABLED=0 go build -a -tags netgo -ldflags '-w' .
strip AWSnycast
mkdir /dist && cd /dist
cp /go/bin/AWSnycast .
fpm -s dir -t deb --name ${project} \
    --iteration ${iteration} --version ${version} \
    --prefix /usr/bin \
    AWSnycast
rm AWSnycast
