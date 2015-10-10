package aws

import (
	"errors"
	awslogger "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/service/ec2"
	"log"
)

type MetadataFetcher interface {
	Available() bool
	GetMetadata(string) (string, error)
}

func NewMetadataFetcher(debug bool) (MetadataFetcher, error) {
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

type RouteTableFetcher interface {
	GetRouteTables() ([]string, error)
}

type RouteTableFetcherEC2 struct {
}

func (rtf RouteTableFetcherEC2) GetRouteTables() ([]string, error) {
	var conn *ec2.EC2
	resp, err := conn.DescribeRouteTables(&ec2.DescribeRouteTablesInput{})
	if err != nil {
		if ec2err, ok := err.(awserr.Error); ok && ec2err.Code() == "InvalidRouteTableID.NotFound" {
			resp = nil
		} else {
			log.Printf("Error on RouteTableStateRefresh: %s", err)
			return []string{}, err
		}
	}
    rt := resp.RouteTables
    log.Printf("%+v", rt)
    ret := make([]string, 0)
    return ret, nil
}

func NewRouteTableFetcher(debug bool) (RouteTableFetcher, error) {
    return RouteTableFetcherEC2{}, nil
}
