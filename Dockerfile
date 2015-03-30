FROM ubuntu:trusty
MAINTAINER Tomas Doran <bobtfish@bobtfish.net>

RUN apt-get -y update
RUN apt-get -y install unzip
ADD https://dl.bintray.com/mitchellh/consul/0.3.1_linux_amd64.zip /tmp/consul.zip
RUN cd /usr/local/sbin && \
    unzip /tmp/consul.zip
RUN apt-get update && apt-get install -y python-pip
RUN pip install exabgp && pip install awscli
RUN mkdir /etc/consul.conf.d
ENTRYPOINT ["/usr/local/sbin/consul", "agent", "-server", "-data-dir=/tmp/consul", "-client=0.0.0.0", "-config-file", "/etc/consul.conf", "-config-dir", "/etc/consul.conf.d"]
EXPOSE 8400 8500 8600/udp

