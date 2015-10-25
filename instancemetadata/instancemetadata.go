package instancemetadata

import (
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
)

type MetadataFetcher interface {
	Available() bool
	GetMetadata(string) (string, error)
}

func New(debug bool) MetadataFetcher {
	c := ec2metadata.Config{}
	if debug {
		c.LogLevel = aws.LogLevel(aws.LogDebug)
	}
	return ec2metadata.New(&c)
}

type InstanceMetadata struct {
	Subnet           string
	Instance         string
	AvailabilityZone string
	Region           string
}

func FetchMetadata(mdf MetadataFetcher) (InstanceMetadata, error) {
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

func getSubnetId(mdf MetadataFetcher) (string, error) {
	mac, err := mdf.GetMetadata("mac")
	if err != nil {
		return "", err
	}
	return mdf.GetMetadata(fmt.Sprintf("network/interfaces/macs/%s/subnet-id", mac))
}
