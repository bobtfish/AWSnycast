package aws

import (
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/awslabs/aws-sdk-go/aws/credentials/ec2rolecreds"
	"log"
)

type MetadataFetcher interface {
	Available() bool
	GetMetadata(string) (string, error)
}

func NewMetadataFetcher(debug bool) (MetadataFetcher, error) {
	c := ec2metadata.Config{}
	if debug {
		c.LogLevel = aws.LogLevel(aws.LogDebug)
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
	GetRouteTables() ([]*ec2.RouteTable, error)
}

type RouteTableFetcherEC2 struct {
	Region string
	conn   *ec2.EC2
}

func (r RouteTableFetcherEC2) GetRouteTables() ([]*ec2.RouteTable, error) {
	resp, err := r.conn.DescribeRouteTables(&ec2.DescribeRouteTablesInput{})
	if err != nil {
		log.Printf("Error on RouteTableStateRefresh: %s", err)
		return []*ec2.RouteTable{}, err
	}
	rt := resp.RouteTables
	log.Printf("%+v", rt)
	return rt, nil
}

func NewRouteTableFetcher(region string, debug bool) (RouteTableFetcher, error) {
	r := RouteTableFetcherEC2{}
	providers := []credentials.Provider{
		&credentials.EnvProvider{},
		&ec2rolecreds.EC2RoleProvider{},
	}
	cred := credentials.NewChainCredentials(providers)
	_, credErr := cred.Get()
	if credErr != nil {
		return r, credErr
	}
	awsConfig := &aws.Config{
		Credentials: cred,
		Region:      aws.String(region),
		MaxRetries:  aws.Int(3),
	}
	iamconn := iam.New(awsConfig)
	_, err := iamconn.GetUser(nil)

	if awsErr, ok := err.(awserr.Error); ok {
		if awsErr.Code() == "SignatureDoesNotMatch" {
			return r, fmt.Errorf("Failed authenticating with AWS: please verify credentials")
		}
	}
	r.conn = ec2.New(awsConfig)
	return r, nil
}
