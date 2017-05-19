# AWSnycast

[![Build Status](https://travis-ci.org/bobtfish/AWSnycast.svg)](https://travis-ci.org/bobtfish/AWSnycast) [![Coverage Status](https://coveralls.io/repos/bobtfish/AWSnycast/badge.svg?branch=master&service=github)](https://coveralls.io/github/bobtfish/AWSnycast?branch=master)

AWSnycast is a routing daemon for AWS route tables, to simulate an Anycast like service, and act as an
extension of in-datacenter Anycast. It can also be used to provide HA NAT service.

# WARNING

Please use release version 0.1.4 rather than master.

# Anycast in AWS?

AWSnycast allows you to implement an Anycast-like method of route publishing - based on healthchecks,
and HA/failover. This is very similar to using systems like [exabgp](https://github.com/Exa-Networks/exabgp)
or [bird](http://bird.network.cz/) on traditional datacenter hosts to publish [BGP](https://en.wikipedia.org/wiki/Border_Gateway_Protocol) routing information
for individual /32 IPs of services they're running.

This means that all your systems in AWS (no matter what region or account) can use *the same* IP/subnet for
highly available services (avoiding the need to reconfigure things at boot time if you're constructing
AMIs).

If you have a physical datacenter with Anycast already, you can configure [VPN tunnels](http://docs.aws.amazon.com/AmazonVPC/latest/UserGuide/VPC_VPN.html),
or [Direct Connects](https://aws.amazon.com/directconnect/) to AWS then publish routes to the Amazon routing tables using BGP,
including an Anycast /24 or /16 - things you bring up in AWS will have more specific routes
locally, but services which only reside in your datacenter will automatically be routed there.

This is super useful for bootstrapping a new VPC (before you have any local services running), or for
providing high availability.

N.B. Whilst publishing routes *from* AWS into your datacenter's BGP would be useful, at this
time that is beyond the goals for this project.

# NAT

A common pattern in AWS is to route egressing traffic (from private subnets) through a NAT instance
in a public subnet. AWSnycast can publish routes for one or more NAT instances for you automatically, allowing
you to deploy one or more NAT instances per VPC.

AWS have recently added a [NAT gateway](http://docs.aws.amazon.com/AmazonVPC/latest/UserGuide/vpc-nat-gateway.html)
product, which simplifies redunent NAT, and allows you to burst to 10G (which needs high spec NAT instances to do).

For small amounts of traffic, NAT gateway is cheaper than NAT instancesm for large amounts of traffic it becomes
more expensive - however NAT instances with AWSnycast probably have a lower reliability, as HA/failover won't happen
until the failed machine is stopped/blackholed.

If you choose to use NAT instances, and AWSnycast to manage their routes, you can deploy 2 (or more) NAT machines, in different availability zones, and configure a private routing table
for each AZ. AWSnycast can then be used to advertise the routes to these NAT machines for High Availability,
such that by default, both machines will be active (for their AZ - to avoid cross AZ transfer costs + latency),
however if one machine fails, then the remaining machine will take over it's route for the duration that
it's down.

# How does this even work?

You can setup routes in AWS to go to an individual instance. This means that you inject routes
that are *outside* the addesses space for your VPC, and point them to an instance, failing over
to a different instance if the machine providing the service fails.

All you have to do on the instance itself is setup a network interface which can deal with this traffic;
for example, alias lo0:0 as 192.168.1.1 and disable source/destination checking for that instance in AWS.

By advertising a larger range via BGP (from VPN or Direct connect) and then injecting /32 routes for
the AWS instances of individual services, you get both bootstrapping (being able to bootstrap new AWS
regions from a VPN connection, as critical services such as dns and puppet/chef are always at a fixed
IP address) and HA.

## Why not manage / move ENIs?

Good question! You *can* provide HA in AWS by assigning each service an ENI
([Elastic Network Interface](http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/using-eni.html)), and
[moving them between healthy instances](http://www.cakesolutions.net/teamblogs/making-aws-nat-instances-highly-available-without-the-compromises).

There are a number of reasons I chose not to do this:
  * Most AWS machine classes/sizes are fairly restricted about the number of ENIs they can have attached.
  * ENIs need to be detached then re-attached - so failover isn't atomic and it makes writing reliable
    distributed software to do this hard without a strong consensus store. (Think detach-fight!)
  * I explicitly didn't want to depend on a strong consensus store (to make this useful for VPC
    bootstrapping).
  * ENIs are an AWS only solution, and don't/can't provide parity with existing Anycast implementations
    in datacenter.

## What do you recommend using this for?

Basic infrastructure services, like DNS (if you use your own DNS already), puppet or chef servers, package
repositories, etc. This is *not* a load balancing or SOA service discovery solution.

I'd *highly* recommend putting any TCP service you use AWSnycast for behind [haproxy](http://www.haproxy.org/), for load balancing (as
only one route can be active at a time - i.e. AWS doesn't support any form of ECMP), and to make AWSnycast's
failover only needed when an instance dies, rather than an individual service instance.

If you're building an SOA, and for 'application level' rather than 'infrastructure level' services, I'd
recommend checking out Consul, Smartstack and similar technologies.

# Trying it out

In the [tests/integration](tests/integration) folder, there is [Terraform](http://terraform.io) code which will build
a simple AWS VPC, with 2 AZs and 2 NAT machines (with HA and failover), and AWSnycast setup.

To try this, you can install the binary and then _terraform apply_ in that directory to build
the example network, then _make sshnat_ to log into one of the machines,
and _journalctl -u awsnycast_ to view the logs.

Try terminating one of the machines and watch routes fail over!

FIXME!! More details about how to test / curl things here..

# Installing (binary)

You can install binary release versions onto x86 Linux
directly from github, e.g.

    sudo wget https://github.com/bobtfish/AWSnycast/releases/download/v0.1.0/AWSnycast -O /usr/local/bin/AWSnycast
    sudo chmod 700 /usr/local/bin/AWSnycast

or you can install the .deb or .rpm packages found at the same location

# Building from source

You need go installed to build this project (tested on go 1.5+1.6). 

Once you have go installed, and the GOPATH environment variable setup
(to, for example, /Users/tdoran/go), you should be able to install with:

    go get github.com/mattn/gom
    go get github.com/bobtfish/AWSnycast
    cd /Users/tdoran/go/src/github.com/bobtfish/AWSnycast
    make

This will build the binary at the top level of the checkout
(i.e. in /Users/tdoran/go/src/github.com/bobtfish/AWSnycast/AWSnycast in my example)

# Running it

You can run AWSnycast -h to get a list of helpful options:

    Usage of AWSnycast:
      -debug
            Enable debugging
      -f string
            Configration file (default "/etc/awsnycast.yaml")
      -noop
            Don't actually *do* anything, just print what would be done
      -oneshot
            Run route table manipulation exactly once, ignoring healthchecks, then exit

Once you've everything is fully set up, you shouldn't need any options.

To run AWSnycast also needs permissions to access the AWS API. This can be done either by
supplying the standard *AWS_ACCESS_KEY_ID* and *AWS_SECRET_ACCESS_KEY* environment
variables, or by applying an IAM Role to the instance running AWSnycast (recommended).

An example IAM Policy is shown below:

    {
      "Version": "2012-10-17",
      "Statement": [
            {
                "Action": [
                    "ec2:ReplaceRoute",
                    "ec2:CreateRoute",
                    "ec2:DeleteRoute",
                    "ec2:DescribeRouteTables",
                    "ec2:DescribeNetworkInterfaces",
                    "ec2:DescribeInstanceAttribute"
                ],
                "Effect": "Allow",
                "Resource": "*"
            }
        ]
    }

DescribeRouteTables + Create/Delete/ReplaceRoute are used for the core functionality,
DescribeNetworkInterfaces is used to get the primary IP address of other instances
which are found to be owning a route (for remote health checks), and DescribeInstanceAttribute
is used to check that the local machine has src/dest checking disabled (and refusing
to start if it doesn't as a safety precaution).

Note that this software *does not* need root permissions, and therefore *should not* be
run as root on your system. Please run it as a normal user (or even as nobody if you're
using an IAM Role).

# Configuration

AWSnycast reads a YAML config file (by default /etc/awsnycast.yaml) to learn which routes to advertise.

An example config is shown below:

        ---
        poll_time: 300 # How often to poll AWS route tables
        healthchecks:
            public:
                type: ping
                destination: 8.8.8.8
                rise: 2  # How many need to succeed in a row to be up
                fall: 10 # How many need to fail in a row to be down
                every: 1 # How often, in seconds
            localservice:
                type: tcp
                destination: 192.168.1.1
                rise: 2
                fall: 2
                every: 30
                config:
                    send: HEAD / HTTP/1.0 # String to send (optional)
                    expect: 200 OK        # Response to get back (optional)
                    port: 80
        remote_healthchecks:
            service:
                type: tcp
                rise: 20 # Note we set these a lot higher than local healthchecks
                fall: 20
                every: 30
                config:
                    send: HEAD / HTTP/1.0 # String to send (optional)
                    expect: 200 OK        # Response to get back (optional)
                    port: 80
        routetables:
            # This is our AZ, so always try to takeover routes always
            our_az:
                find:
                    type: and
                    config:
                      - type: by_tag
                        config:
                            key: az 
                            value: eu-west-1a
                      - type: by_tag
                            config:
                                key: type
                                value: private
                manage_routes:
                  - cidr: 0.0.0.0/0     # NAT box, so manage the default route
                    instance: SELF
                    healthcheck: public
                    never_delete: true   # Don't delete the default route if google goes down ;)
                  - cidr: 192.168.1.1/32 # Manage an AWSnycast service on this machine
                    instance: SELF
                    healthcheck: localservice
                    remote_healthcheck: service
            # This is not our AZ, so only takeover routes only if they don't exist already, or
            # the instance serving them is dead (terminated or stopped).
            # Every backup AWSnycast instance should have if_unhealthy: true set for route tables
            # it is the backup server for, otherwise multiple instances can cause routes to flap
            other_azs:
                find:
                    no_results_ok: true # This allows you to deploy the same config in regions you only have 1 AZ
                    type: and
                    config:
                        filters:
                          - type: by_tag
                            not: true
                            config:
                                key: az
                                value: eu-west-1a
                          - type: by_tag
                            config:
                                key: type
                                value: private
                manage_routes:
                  - cidr: 0.0.0.0/0
                    if_unhealthy: true # Note this is what causes routes only to be taken over not present currently, or the instance with them has failed
                    instance: SELF
                  - cidr: 192.168.1.1/32
                    if_unhealthy: true
                    instance: SELF
                    healthcheck: localservice
                    remote_healthcheck: service

## Healthchecks

Healthchecks are indicated by the top level 'healthchecks' key. Values are a hash of name / definition.

The definition is composed of a few fields:

 * type - required
 * destination - required. The destination IP for the healthcheck. This *must* be an IP.
 * rise - optional, how many checks need to pass in a row to become healthy. Default 2
 * fall - optional, how many checks need to fail in a row to become unhealthy. Default 2
 * every - required, how often in seconds to run the healthcheck
 * config - optional, A hash of keys/values for the specific healthcheck type you are using
 * run_on_healthy - optional. An array holding a script/command to run when the healthcheck becomes healthy.
 * run_on_unhealthy - optional. An array holding a script/command to run when the healthcheck becomes unhealthy.

### ping

Does an ICMP ping against the destination.

### tcp

Makes a TCP connection against the destination on a port. Optionally sends data and checks
returned data.

Takes a number of config parameters:

  * port - required, the port number to connect on
  * send - optional, a string to send to the remote side
  * expect - optional, a string to expect back in the
             response from the remote side
  * ssl - optional, a bool for if to use TLS to connect
  * certPath - optional, path to a file containing a certificate
  * cert - optional, the certificate as a string
  * skipVerify - optional, a bool which if true will skip certificate verification
  * serverName - FIXME

### command

Run an arbitrary command. Exit status 0 is success, anything else is a failure.

Takes the following config parameters:

  * command - required, the command to run
  * arguments - options, a list of arguments to supply to the command.

If any of the arguments contains the string %DESTINATION% then it will be
replaced by the healthcheck destination.

For example, you can replicate the ping healthcheck with the following arguments:

  * command - ping
  * arguments - -c, 1, %DESTINATION%

## Route tables

Indicated by the top level 'route_tables' key. Values are a hash of name / definition.

The definition is composed of a few fields:

 * find (see Finding them below)
 * manage_routes (see Managing them below)

### Finding them

Route tables are found by various 'filters' which you can configure. Each finder has a 'type' and some 'config'.

Depending on the type of your finder, different config keys are required / valid.

All filters can optionally take a 'not' key, which if true inverts the result

Top level filters can optionally take a 'no_results_ok' key, which if true stops AWSnycast from failing if no
route tables can be found. Use this when you want a setup with a backup server / az in *some* regions, but
you want to deploy the same config in all regions.

Currently supported filters are:

#### by_tag

Does a simple equality match on a tag. Expects config keys 'key' and 'value', whos values are exaclty matched
with the tag's key and value.

#### and

Runs a series of other filters, and only matches if *all* of it's filters match.

Supply a list to the 'filters' config key, with values being hashes of other finders

#### or

Runs a series of other filters, and matches if *any* of it's filters match.

Supply a list to the 'filters' config key, with values being hashes of other finders

### main

Matches the main route table only. Takes no config parmeters

### subnet

Matches the route table associated with the subnet given in the 'subnet_id' config key

### has_route_to

Matches any route tables which have a route to a specific (and exact) cidr (given by the 'cidr'
config key).

### Managing them

Routes to be managed are a list of hashes, with the following keys:

  * cidr - required. The address to advertise into the route table
  * instance - required. The Amazon instance ID to route this cidr to. Can be
    SELF to mean this instance
  * healthcheck - optional. The string name of the healthcheck to associate
    with this route. If the healthcheck doesn't pass then the route will be
    removed from the routing table (allowing you to failover to a wider scope
    Anycast route advertised from your datacenter)
  * if_unhealthy - true. Only take this route over if the instance currently
    associated with it is unhealthy in the AWS route table (i.e. black holing
    traffic). This is used for backup servers in a multi-az deployment.
  * remote_healthcheck - FIXME
  * run_before_replace_route - FIXME
  * run_after_replace_route - FIXME
  * run_before_add_route - FIXME
  * run_after_add_route - FIXME

# Releases

Release (stable) versions of AWSnycast are tagged in the repository, and go binaries (generated by Travis CI)
are uploaded to github - see the CHANGELOG.md for information about what is in each release.

Note that currently this project is pre-1.0, so I reserve the right to make massive sweeping changes
between versions.

Note also that incorrect use of this project can completely mess up your AWS routing tables, and make your instances inaccessible! You are *HIGHLY* recommended to become confident using the _-noop_
mode before running this for real!

# TODO

This project is currently under heavy development.

Here's a list of the features that I'm planning to work on next, in approximate order:

  * Autodetect this machine's AZ
  * Make us autodetect the VPC this instance is running in, and refuse to adjust routing tables in other VPCs
  * Enable the use of multiple different healthchecks for a route (to only consider it down if multiple checks fail)
  * Add serf gossip between instances, to allow faster and reliable failover/STONITH
  * Add the ability to have external clients participate in healthchecks in the serf network.
  * Add a web interface to be able to get the state as HTML or JSON (and manually initiate failovers?)

# Contributing

Bug reports, success(or failure) stories, questions, suggestions, feature requests and (documentation or code) patches are all very welcome.

Please feel free to ping t0m on Freenode or @bobtfish on Twitter if you'd like help configuring/using/debugging/improving this software.

Please note however that this project is released with a Contributor Code of Conduct. By participating in this project you agree to abide by its terms.

# Copyright

Copyright Tomas Doran 2015

# License

Apache2 - See the included LICENSE file for more details.

