package daemon

import (
	"fmt"
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
	Region string
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
                log.Printf("Error getting AZ: %s", err.Error())
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

func (d *Daemon) runHealthChecks() {
	log.Printf("Starting healthchecks")
	for _, v := range d.Config.Healthchecks {
		v.Run()
	}
	log.Printf("Done starting healthchecks")
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
	log.Printf(subnet)
	rt, err := d.RouteTableFetcher.GetRouteTables()
	if err != nil {
		log.Printf("Error %v", err)
		return 1
	}
	for _, val := range rt {
		log.Printf("Route table %v", val)
	}
	d.runHealthChecks()
	return 0
}
