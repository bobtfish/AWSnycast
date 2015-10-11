# AWSnycast

AWSnycast is a routing daemon for AWS route tables, to simulate an Anycast like service, and act as an
extension of in-datacenter Anycast.

In datacenters, a common pattern is to have a /24 network for Anycast, and then in each datacenter,
use systems like [exabgp](https://github.com/Exa-Networks/exabgp) and [bird](http://bird.network.cz/)
on hosts to publish BGP routing information for services they're running.

In AWS, we can configure VPN tunnels, or Direct Connects, and can publish routes to the Amazon
routing tables using BGP, including the Anycast /24. AWSnycast runs locally on your AWS nodes,
healthchecks local services, and publishes more specific routes to them into the AWS route table.

This means that all your systems in AWS can use *the same* network for Anycast services as your
in datacenter machines, *and* talk to services locally, as you bring them up in AWS. This is
super useful for bootstrapping a new VPC (before you have any local services running), or for
providing high availability.

You don't *have* to have a datacenter or VPN connection for AWSnycast to be useful to you, you
still get route publishing based on healthchecks, and HA/failover, just not bootstrapping
from in datacenter.

N.B. This software should be considered *alpha* quality at best, it's currently vapourware / highly experimental.

N.B. Whilst publishing routes to *from* AWS into your datacenter's BGP would be useful, at this
time that is beyond the goals for this project.

# Configuration

Which routes to advertise into which route tables is configured with a YAML config file.

        ---
        healthcheck:
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
                send: HEAD / HTTP/1.0 # String to send
                expect: 200 OK        # Response to get back
        routetables:
            # This is our AZ, so always try to takeover routes always
            a:
                find:
                    type: by_tag
                    config:
                        key: Name
                        value: private a
                upsert_routes:
                  - cidr: 0.0.0.0/0     # NAT box, so manage the default route
                    instance: SELF
                    healthcheck: public
                  - cidr: 192.168.1.1/32 # Manage an AWSnycast service on this machine
                    instance: SELF
                    healthcheck: localservice
            # This is not our AZ, so only takeover routes only if they don't exist already, or
            # the instance serving them is dead (terminated or stopped)
            b:
                find:
                    type: by_tag
                    config:
                        key: Name
                        value: private b
                upsert_routes:
                  - cidr: 0.0.0.0/0
                    if_unhealthy: true
                    instance: SELF
                    healthcheck: public
                  - cidr: 192.168.1.1/32
                    if_unhealthy: true
                    instance: SELF
                    healthcheck: localservice

