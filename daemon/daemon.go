package daemon

import (
	log "github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/bobtfish/AWSnycast/aws"
	"github.com/bobtfish/AWSnycast/config"
	"github.com/bobtfish/AWSnycast/instancemetadata"
	"time"
)

type Daemon struct {
	oneShot           bool
	noop              bool
	ConfigFile        string
	Debug             bool
	Config            *config.Config
	MetadataFetcher   instancemetadata.MetadataFetcher
	RouteTableManager aws.RouteTableManager
	quitChan          chan bool
	loopQuitChan      chan bool
	FetchWait         time.Duration
	instancemetadata.InstanceMetadata
}

func (d *Daemon) setupMetadataFetcher() {
	if d.MetadataFetcher == nil {
		d.MetadataFetcher = instancemetadata.New(d.Debug)
	}
}

func (d *Daemon) Setup() error {
	d.setupMetadataFetcher()
	im, err := instancemetadata.FetchMetadata(d.MetadataFetcher)
	if err != nil {
		return err
	}
	d.InstanceMetadata = im

	if d.RouteTableManager == nil {
		d.RouteTableManager = aws.NewRouteTableManager(d.Region, d.Debug)
	}

	config, err := config.New(d.ConfigFile, d.InstanceMetadata, d.RouteTableManager)
	if err != nil {
		return err
	}
	d.Config = config

	if d.FetchWait == 0 {
		d.FetchWait = time.Second * time.Duration(config.PollTime)
	}

	return setupHealthchecks(d.Config)
}

func setupHealthchecks(c *config.Config) error {
	for _, v := range c.Healthchecks {
		err := v.Setup()
		if err != nil {
			return err
		}
	}
	return nil
}

func (d *Daemon) runHealthChecks() {
	log.Debug("Starting healthchecks")
	for _, v := range d.Config.Healthchecks {
		v.Run(d.Debug)
	}
	for _, configRouteTables := range d.Config.RouteTables {
		for _, mr := range configRouteTables.ManageRoutes {
			mr.StartHealthcheckListener(d.noop)
		}
	}
	log.Debug("Started all healthchecks")
}

func (d *Daemon) stopHealthChecks() {
	for _, v := range d.Config.Healthchecks {
		v.Stop()
	}
}

func (d *Daemon) RunOneRouteTable(rt []*ec2.RouteTable, name string, configRouteTable *config.RouteTable) error {
	if err := configRouteTable.UpdateEc2RouteTables(rt); err != nil {
		return err
	}
	return configRouteTable.RunEc2Updates(d.RouteTableManager, d.noop)
}

func (d *Daemon) RunRouteTables() error {
	rt, err := d.RouteTableManager.GetRouteTables()
	if err != nil {
		return err
	}
	for name, configRouteTables := range d.Config.RouteTables {
		if err := d.RunOneRouteTable(rt, name, configRouteTables); err != nil {
			return err
		}
	}
	return nil
}

func (d *Daemon) Run(oneShot bool, noop bool) int {
	d.oneShot = oneShot
	d.noop = noop
	if err := d.Setup(); err != nil {
		log.WithFields(log.Fields{"err": err.Error()}).Error("Error in initial setup")
		return 1
	}

	if !d.RouteTableManager.InstanceIsRouter(d.Instance) {
		log.WithFields(log.Fields{"instance_id": d.Instance}).Error("I am not a router (do not have src/destination checking disabled)")
		return 1
	}

	d.quitChan = make(chan bool, 1)
	err := d.RunRouteTables()
	if err != nil {
		log.WithFields(log.Fields{"err": err.Error()}).Error("Error in initial route table run")
		return 1
	}
	d.loopQuitChan = make(chan bool, 1)
	if oneShot {
		d.quitChan <- true
	} else {
		d.runHealthChecks()
		defer d.stopHealthChecks()
		d.RunSleepLoop()
	}
	<-d.quitChan
	d.loopQuitChan <- true
	return 0
}

func (d *Daemon) RunSleepLoop() {
	go func() {

		ticker := time.NewTicker(d.FetchWai)
		fetch := ticker.C

		for {
			select {
			case <-d.loopQuitChan:
				return
			case <-fetch:
				err := d.RunRouteTables()
				if err != nil {
					log.WithFields(log.Fields{"err": err.Error()}).Warn("Error in route table poll run")
				}
			}
		}
	}()
}
