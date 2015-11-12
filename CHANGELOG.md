  - Add the ability to setup a route finder which does not bomb out it
    doesn't find anything

Version 0.0.2 - 2015-11-12

  - Make the interval between polls to AWS for the current routing tables
    be configurable with the top level poll_time key. (Defaults to 300s)
  - Add a never_delete flag which can/should be used with the default route
    (0.0.0.0/0) when providing NAT service to not delete the route if connectivity
    to a healthcheck location fails.
  - Expand route filters with two new types 'and' and 'or'. These both take a single
    config key, 'filters', which is a list of other filters to and/or together.
    If you have a few consistent tags on your route tables then you can use this
    to create logic for finding sets of route tables, rather than having to hard
    code route table names/numbers/etc into the config.
  - Update to the breaking changes in the latest version of aws-sdk-go
  - Deb packaging example added to the repository
  - Expand the integration test examples to use all the features above,
    and have an example of an Anycast type service and not just NAT.

Version 0.0.1 - 2015-10-25

  - Well, [@garethr](http://twitter.com/garethr) put this in devops weekly,
    so I guess I should cut a binary release.
  - Currently working feature set:
    - NAT (0.0.0.0/0) or Anycast (/32) routes can be injected into route
      tables found by tags, when healthchecks pass.
    - Simple ping / tcp healthchecks work. (I think maaaybe you can do HTTP with
      this, but haven't tried it in the integration tests yet!)
    - Routes are deleted if owned by the current instance and the
      healthcheck starts failing.
    - Routes are added if not present (or attached to an instance/ENI which
      is dead / blackholing traffic) as soon as healthchecks start passing
    - AWS is polled (DescribeRouteTables) every 300s.
  - Important things which *don't* work.
    - Do not try this if you have multiple VPCs in the same region, with the
      same tags on routing tables, you *will* have a bad time.
    - HA/takeover is *only* done for blackholed traffic (dead instance/disconnected
      ENI). You cannot (yet) healthcheck other instances (e.g. the current
      instance providing a route) - so if an instance gets a route,
      AWSnycast stops running on that instance, and the service breaks, then
      it'll not be fixed until you manually stop or terminate the unhealthy instance!

