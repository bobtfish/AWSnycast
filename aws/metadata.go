package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
)

type MetadataFetcher interface {
	Available() bool
	GetMetadata(string) (string, error)
}

func NewMetadataFetcher(debug bool) MetadataFetcher {
	c := ec2metadata.Config{}
	if debug {
		c.LogLevel = aws.LogLevel(aws.LogDebug)
	}
	return ec2metadata.New(&c)
}
