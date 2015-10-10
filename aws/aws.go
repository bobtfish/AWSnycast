package aws

import (
	"errors"
	awslogger "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"log"
)

type MetadataFetcher interface {
	Available() bool
	GetMetadata(string) (string, error)
}

func New(debug bool) (MetadataFetcher, error) {
	c := ec2metadata.Config{}
	if debug {
		c.LogLevel = awslogger.LogLevel(awslogger.LogDebug)
	}
	m := ec2metadata.New(&c)
	if m.Available() {
		log.Printf("Have metadata service")
		return m, nil
	} else {
		log.Printf("No metadata service")
		return m, errors.New("No metadata service")
	}
}
