package daemon

import (
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/bobtfish/AWSnycast/aws"
	"github.com/bobtfish/AWSnycast/config"
	"log"
)

type Daemon struct {
	ConfigFile        string
	Debug             bool
	Config            *config.Config
	MetadataFetcher   aws.MetadataFetcher
	RouteTableFetcher aws.RouteTableFetcher
	Subnet            string
	Instance          string
	Region            string
	quitChan          <-chan bool
}

func (d *Daemon) Setup() error {
	config, err := config.New(d.ConfigFile)
	if err != nil {
		return err
	}
	d.Config = config
	if d.MetadataFetcher == nil {
		m, err := aws.NewMetadataFetcher(d.Debug)
		if err != nil {
			return err
		}
		d.MetadataFetcher = m
	}
	az, err := d.MetadataFetcher.GetMetadata("placement/availability-zone")
	if err != nil {
		return errors.New(fmt.Sprintf("Error getting AZ: %s", err.Error()))
	}
	d.Region = az[:len(az)-1]
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

func (d *Daemon) GetInstance() (string, error) {
	return "i-123", nil
}

func (d *Daemon) runHealthChecks() {
	log.Printf("Starting healthchecks")
	for _, v := range d.Config.Healthchecks {
		v.Run()
	}
	log.Printf("Done starting healthchecks")
}

func (d *Daemon) RunOneRouteTable(rt []*ec2.RouteTable, name string, configRouteTables *config.RouteTable) error {
	filter, err := configRouteTables.Find.GetFilter()
	if err != nil {
		return err
	}
	remaining := aws.FilterRouteTables(filter, rt)
	for _, rtb := range remaining {
		log.Printf("Finder name %s found route table %v", name, rtb)
	RouteLoop:
		for _, upsertRoute := range configRouteTables.UpsertRoutes {
			if upsertRoute.Healthcheck != "" {
				if _, ok := d.Config.Healthchecks[upsertRoute.Healthcheck]; ok {
				} else {
					panic("moo")
				}
				break RouteLoop
			}
			destInstance := upsertRoute.Instance
			if upsertRoute.Instance == "SELF" {
				destInstance = d.Instance
			}
			log.Printf(destInstance)
		}
	}
	return nil
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

func (d *Daemon) Run() int {
	if err := d.Setup(); err != nil {
		log.Printf("Error setting up: %s", err.Error())
		return 1
	}
	subnet, err := d.GetSubnetId()
	if err != nil {
		log.Printf("Error getting metadata: %s", err.Error())
		return 1
	}
	d.Subnet = subnet
	instanceId, err := d.GetInstance()
	if err != nil {
		log.Printf("Error getting metadata: %s", err.Error())
		return 1
	}
	d.Instance = instanceId
	d.quitChan = make(chan bool)
	log.Printf(subnet)
	log.Printf(instanceId)
	d.runHealthChecks()
	err = d.RunRouteTables()
	if err != nil {
		log.Printf("Error: %v", err)
		return 1
	}
	<-d.quitChan
	return 0
}
