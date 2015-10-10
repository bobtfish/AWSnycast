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
	GetRouteTables() ([]string, error)
}

type RouteTableFetcherEC2 struct {
}

func (rtf RouteTableFetcherEC2) GetRouteTables() ([]string, error) {
	providers := []credentials.Provider{
		&credentials.EnvProvider{},
		&ec2rolecreds.EC2RoleProvider{},
	}
	cred := credentials.NewChainCredentials(providers)
	_, credErr := cred.Get()
	if credErr != nil {
		return []string{}, credErr
	}
	//	credVal.AccessKeyID
	//	credVal.SecretAccessKey
	//	credVal.SessionToken
	awsConfig := &aws.Config{
		Credentials: cred,
		Region:      aws.String("us-west-1"),
		MaxRetries:  aws.Int(3),
	}
	iamconn := iam.New(awsConfig)
	_, err := iamconn.GetUser(nil)

	if awsErr, ok := err.(awserr.Error); ok {
		if awsErr.Code() == "SignatureDoesNotMatch" {
			return []string{}, fmt.Errorf("Failed authenticating with AWS: please verify credentials")
		}
	}
	ec2conn := ec2.New(awsConfig)
	resp, err := ec2conn.DescribeRouteTables(&ec2.DescribeRouteTablesInput{})
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
