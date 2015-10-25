
Version 0.0.1
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

