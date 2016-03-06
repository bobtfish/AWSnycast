package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
)

type MetadataFetcher interface {
	Available() bool
	GetMetadata(string) (string, error)
}

func NewMetadataFetcher(debug bool) MetadataFetcher {
	sess := session.New()
	if debug {
		sess.Config.LogLevel = aws.LogLevel(aws.LogDebug)
	}
	return ec2metadata.New(sess)
}
