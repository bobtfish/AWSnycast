package daemon

import (
	"github.com/bobtfish/AWSnycast/aws"
	"log"
)

type Daemon struct {
	ConfigFile        string
	Debug             bool
	MetadataFetcher   aws.MetadataFetcher
	RouteTableFetcher aws.RouteTableFetcher
}

func (d *Daemon) Setup() error {
	if d.MetadataFetcher == nil {
		m, err := aws.NewMetadataFetcher(d.Debug)
		if err != nil {
			return err
		}
		d.MetadataFetcher = m
	}
	if d.RouteTableFetcher == nil {
		rtf, err := aws.NewRouteTableFetcher("us-west-1", d.Debug)
		if err != nil {
			return err
		}
		d.RouteTableFetcher = rtf
	}
	return nil
}

func (d *Daemon) Run() int {
	if err := d.Setup(); err != nil {
		log.Printf("Error setting up: %s", err.Error())
		return 1
	}
    rt, err := d.RouteTableFetcher.GetRouteTables()
    if err != nil {
        log.Printf("Error %v", err)
        return 1
    }
    for _, val := range rt {
        log.Printf("Route table %v", val)
    }
	return 0
}
