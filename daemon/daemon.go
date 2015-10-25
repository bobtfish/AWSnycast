package daemon

import (
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/bobtfish/AWSnycast/aws"
	"github.com/bobtfish/AWSnycast/config"
	"log"
	"time"
)

type Daemon struct {
	oneShot           bool
	noop              bool
	ConfigFile        string
	Debug             bool
	Config            *config.Config
	MetadataFetcher   aws.MetadataFetcher
	RouteTableManager aws.RouteTableManager
	Subnet            string
	Instance          string
	Region            string
	quitChan          chan bool
	loopQuitChan      chan bool
	loopTimerChan     chan bool
	FetchWait         time.Duration
}

func (d *Daemon) setupMetadataFetcher() {
	if d.MetadataFetcher == nil {
		d.MetadataFetcher = aws.NewMetadataFetcher(d.Debug)
	}
}

func (d *Daemon) Setup() error {
	d.setupMetadataFetcher()
	if !d.MetadataFetcher.Available() {
		return errors.New("No metadata service")
	}
	if d.FetchWait == 0 {
		d.FetchWait = time.Second * 300
	}
	az, err := d.MetadataFetcher.GetMetadata("placement/availability-zone")
	if err != nil {
		return errors.New(fmt.Sprintf("Error getting AZ: %s", err.Error()))
	}
	d.Region = az[:len(az)-1]

	instanceId, err := d.MetadataFetcher.GetMetadata("instance-id")
	if err != nil {
		return errors.New(fmt.Sprintf("Error getting instance-id: %s", err.Error()))
	}
	d.Instance = instanceId

	subnet, err := d.GetSubnetId()
	if err != nil {
		return errors.New(fmt.Sprintf("Error getting metadata: %s", err.Error()))
	}
	d.Subnet = subnet

	config, err := config.New(d.ConfigFile, instanceId)
	if err != nil {
		return err
	}
	d.Config = config

	if d.RouteTableManager == nil {
		d.RouteTableManager = aws.NewRouteTableManager(d.Region, d.Debug)
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

func (d *Daemon) GetSubnetId() (string, error) {
	mac, err := d.MetadataFetcher.GetMetadata("mac")
	if err != nil {
		return "", err
	}
	return d.MetadataFetcher.GetMetadata(fmt.Sprintf("network/interfaces/macs/%s/subnet-id", mac))
}

func (d *Daemon) runHealthChecks() {
	if d.Debug {
		log.Printf("Starting healthchecks")
	}
	for _, v := range d.Config.Healthchecks {
		v.Run(d.Debug)
	}
	for _, configRouteTables := range d.Config.RouteTables {
		for _, mr := range configRouteTables.ManageRoutes {
			mr.StartHealthcheckListener(d.noop)
		}
	}
	if d.Debug {
		log.Printf("Done starting healthchecks")
	}
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
		log.Printf("Error setting up: %s", err.Error())
		return 1
	}
	d.quitChan = make(chan bool, 1)
	err := d.RunRouteTables()
	if err != nil {
		log.Printf("Error: %v", err)
		return 1
	}
	d.loopQuitChan = make(chan bool, 1)
	d.loopTimerChan = make(chan bool, 1)
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
		for {
			select {
			case <-d.loopQuitChan:
				return
			case <-d.loopTimerChan:
				err := d.RunRouteTables()
				if err != nil {
					log.Printf("Error: %v", err)
				}
				go func() {
					time.Sleep(d.FetchWait)
					d.loopTimerChan <- true
				}()
			}
		}
	}()
	go func() {
		time.Sleep(d.FetchWait)
		d.loopTimerChan <- true
	}()
}
