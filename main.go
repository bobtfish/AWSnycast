package main

import (
	"flag"
	"github.com/bobtfish/AWSnycast/daemon"
	"os"
)

var (
	debug = flag.Bool("debug", false, "Enable debugging")
	f     = flag.String("f", "/etc/awsnycast.yaml", "point configration file, default /etc/awsnycast.yaml")
)

func main() {
	flag.Parse()
	d := new(daemon.Daemon)
	d.Debug = *debug
	d.ConfigFile = *f
	os.Exit(d.Run())
}
