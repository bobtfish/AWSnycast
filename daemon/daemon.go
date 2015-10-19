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
	RouteTableFetcher aws.RouteTableFetcher
	Subnet            string
	Instance          string
	Region            string
	quitChan          chan bool
}

func (d *Daemon) Setup() error {
	config, err := config.New(d.ConfigFile)
	if err != nil {
		return err
	}
	d.Config = config

	if d.MetadataFetcher == nil {
		d.MetadataFetcher = aws.NewMetadataFetcher(d.Debug)
	}
	if !d.MetadataFetcher.Available() {
		return errors.New("No metadata service")
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

	if d.RouteTableFetcher == nil {
		rtf, err := aws.NewRouteTableFetcher(d.Region, d.Debug)
		if err != nil {
			return err
		}
		d.RouteTableFetcher = rtf
	}
	for _, v := range d.Config.Healthchecks {
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
	filter, err := configRouteTable.Find.GetFilter()
	if err != nil {
		return err
	}
	remaining := aws.FilterRouteTables(filter, rt)
	for _, rtb := range remaining {
		log.Printf("Finder name %s found route table %v", name, rtb)
		for _, upsertRoute := range configRouteTable.UpsertRoutes {
			//log.Printf("Trying to upsert route to %s", upsertRoute.Cidr)
			if err := d.RunOneUpsertRoute(rtb, name, upsertRoute); err != nil {
				return err
			}
		}
	}
	return nil
}

func (d *Daemon) HealthCheckOneUpsertRoute(name string, upsertRoute *config.UpsertRoutesSpec) bool {
	if !d.oneShot && upsertRoute.Healthcheck != "" {
		if d.Config == nil || d.Config.Healthchecks == nil {
			panic("No healthchecks, have you run Setup()?")
		}
		if hc, ok := d.Config.Healthchecks[upsertRoute.Healthcheck]; ok {
			//log.Printf("Got healthcheck %s", upsertRoute.Healthcheck)
			if !hc.IsHealthy() {
				log.Printf("Skipping upsert route %s, healthcheck %s isn't healthy yet", name, upsertRoute.Healthcheck)
				return false
			}
		} else {
			panic(fmt.Sprintf("Could not find healthcheck %s", upsertRoute.Healthcheck))
		}
		log.Printf("%s Is healthy", upsertRoute.Healthcheck)
	}
	return true
}

func (d *Daemon) RunOneUpsertRoute(rtb *ec2.RouteTable, name string, upsertRoute *config.UpsertRoutesSpec) error {
	if !d.HealthCheckOneUpsertRoute(name, upsertRoute) {
		return nil
	}

	return d.RouteTableFetcher.(aws.RouteTableFetcherEC2).CreateOrReplaceInstanceRoute(*rtb, upsertRoute.Cidr, upsertRoute.GetInstance(d.Instance), upsertRoute.IfUnhealthy, d.noop)
}

func (d *Daemon) RunRouteTables() error {
	rt, err := d.RouteTableFetcher.GetRouteTables()
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
	if !oneShot {
		d.runHealthChecks()
		defer d.stopHealthChecks()
		time.Sleep(time.Second * 3)
	}
	err := d.RunRouteTables()
	if err != nil {
		log.Printf("Error: %v", err)
		return 1
	}
	if oneShot {
		d.quitChan <- true
	} else {
		go func() {
			time.Sleep(time.Second * 300)
			err := d.RunRouteTables()
			if err != nil {
				log.Printf("Error: %v", err)
			}
		}()
	}
	<-d.quitChan
	return 0
}
