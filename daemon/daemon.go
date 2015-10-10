package daemon

import (
	"github.com/bobtfish/AWSnycast/aws"
	"log"
)

type Daemon struct {
	ConfigFile      string
	Debug           bool
	MetadataFetcher aws.MetadataFetcher
}

func (d *Daemon) Setup() error {
	if d.MetadataFetcher == nil {
		m, err := aws.New(d.Debug)
		if err != nil {
			return err
		}
		d.MetadataFetcher = m
	}
	return nil
}

func (d *Daemon) Run() int {
	if err := d.Setup(); err != nil {
		log.Printf("Error setting up: %s", err.Error())
		return 1
	}
	return 0
}
