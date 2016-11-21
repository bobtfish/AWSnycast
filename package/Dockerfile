FROM docker-dev.yelpcorp.com/trusty_yelp
MAINTAINER Tomas Doran <bobtfish@bobtfish.net>

RUN apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -yq \
    go \
    git \
    build-essential \
    ruby2.0 \
    ruby-dev \
    --no-install-recommends

ENV GOPATH /go
ENV PATH /go/bin:/usr/bin:/bin:/usr/local/bin
RUN gem install json --version 1.8.3
RUN gem install fpm --version 1.3.3
RUN go get github.com/mattn/gom

