package daemon

import (
	"errors"
	"fmt"
	"github.com/bobtfish/AWSnycast/aws"
)

type InstanceMetadata struct {
	Subnet           string
	Instance         string
	AvailabilityZone string
	Region           string
}

func (d *Daemon) setupMetadataFetcher() {
	if d.MetadataFetcher == nil {
		d.MetadataFetcher = aws.NewMetadataFetcher(d.Debug)
	}
}

func fetchMetadata(mdf aws.MetadataFetcher) (InstanceMetadata, error) {
	m := InstanceMetadata{}
	if !mdf.Available() {
		return m, errors.New("No metadata service")
	}
	az, err := mdf.GetMetadata("placement/availability-zone")
	if err != nil {
		return m, errors.New(fmt.Sprintf("Error getting AZ: %s", err.Error()))
	}
	m.AvailabilityZone = az
	m.Region = az[:len(az)-1]

	instanceId, err := mdf.GetMetadata("instance-id")
	if err != nil {
		return m, errors.New(fmt.Sprintf("Error getting instance-id: %s", err.Error()))
	}
	m.Instance = instanceId

	subnet, err := getSubnetId(mdf)
	if err != nil {
		return m, errors.New(fmt.Sprintf("Error getting metadata: %s", err.Error()))
	}
	m.Subnet = subnet

	return m, nil
}

func getSubnetId(mdf aws.MetadataFetcher) (string, error) {
	mac, err := mdf.GetMetadata("mac")
	if err != nil {
		return "", err
	}
	return mdf.GetMetadata(fmt.Sprintf("network/interfaces/macs/%s/subnet-id", mac))
}
