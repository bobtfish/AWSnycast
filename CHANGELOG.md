Version 0.1.3 - 2016-12-19
 - Add version to the user agent sent.
 - Upgrade AWS SDK
 - Additional paranoia and debug logging

Version 0.1.2 - 2016-04-15
 - Initial support for running hooks on creating/deleting
   routes. Not yet documented or fully tested, but
   this is the first part of fixing issue #1
 - Switch from Godep to Gom to manage dependencies.
 - Upgrade aws-sdk-go to the latest upstream version.

Version 0.1.1 - 2016-03-05
 - Add a 'command' healthcheck type.
 - Documentation updates / improvements
 - Better type conversions in the "config" attribute for health
   checks (Fixes #6)

Version 0.1.0 - 2016-02-01
 - Add the ability to negotiate SSL connections when
   doing healthchecks.
 - Remove some of the scary warning text from README

Version 0.0.10 - 2015-12-05
 - First version with working remote_healthchecks

Version 0.0.9 - 2015-12-04
 - Fix various crash cases
 - Almost working remote_healthchecks
 - More improvements to error reporting in configuration

Version 0.0.8 - 2015-11-29
 - Fix crash on route tables which don't have an instance-id
   attached.
 - More improvements to error reporting in configuration
 - Make the error reporting of healthcheck configuration better by
   reporting all errors at once, rather than the first one.

Version 0.0.7 - 2015-11-28 *BROKEN DO NOT USE*
 - Make config tell you about all the errors found when dieing at startup,
   rather than just the first one found.
 - Now requires the ec2:DescribeInstanceAttribute permission to run.
 - Local instance is checked for src/destination check being disabled
   at startup. If not disabled AWSnycast will quit with an error message
   (as you need this disabled for either NAT boxes or those handinging
   Anycast addresses)
 - Add a -syslog CLI flag, which will cause the logs printed on STDOUT
   to be duplicated to syslog.
 - When replacing an existing route, log the old and new instance IDs,
   and the route state.
 - Improved unit tests - added coverage.
 - Additional work towards the remote_healthchecks feature working.

Version 0.0.6 - 2015-11-13
  - Build .deb and .rpm packages for Linux as part of the release process

Version 0.0.5 - 2015-11-13
  - Build a .deb package for amd64 as part of the release process
  - Add additional methods to find route tables
  - More logging improvements
  - Fix the integration tests to not run Terrafom themselves

Version 0.0.4 - 2015-11-13
  - Make the logging much better, with more context to log messages
    - Make debug mode output more verbose debug logging
  - Add integration tests of Anycast functionality
  - Make the integration test environment useful again
  - Start using the 'godep' tool to freeze dependencies

Version 0.0.3 - 2015-11-12

  - Add much better documentation
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

