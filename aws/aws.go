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
	return resp.RouteTables, nil
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

type RouteTableFilter interface {
	Keep(*ec2.RouteTable) bool
}

type RouteTableFilterAlways struct{}

func (fs RouteTableFilterAlways) Keep(rt *ec2.RouteTable) bool {
	return false
}

type RouteTableFilterNever struct{}

func (fs RouteTableFilterNever) Keep(rt *ec2.RouteTable) bool {
	return true
}

type RouteTableFilterAnd struct {
	RouteTableFilters []RouteTableFilter
}

func (fs RouteTableFilterAnd) Keep(rt *ec2.RouteTable) bool {
	for _, f := range fs.RouteTableFilters {
		if !f.Keep(rt) {
			return false
		}
	}
	return true
}

type RouteTableFilterOr struct {
	RouteTableFilters []RouteTableFilter
}

func (fs RouteTableFilterOr) Keep(rt *ec2.RouteTable) bool {
	for _, f := range fs.RouteTableFilters {
		if f.Keep(rt) {
			return true
		}
	}
	return false
}

type RouteTableFilterMain struct{}

func (fs RouteTableFilterMain) Keep(rt *ec2.RouteTable) bool {
	for _, a := range rt.Associations {
		if *(a.Main) {
			return true
		}
	}
	return false
}

func FilterRouteTables(f RouteTableFilter, tables []*ec2.RouteTable) []*ec2.RouteTable {
	out := make([]*ec2.RouteTable, 0, len(tables))
	for _, rtb := range tables {
		if f.Keep(rtb) {
			out = append(out, rtb)
		}
	}
	return out
}

func RouteTableForSubnet(subnet string, tables []*ec2.RouteTable) *ec2.RouteTable {
	subnet_rtb := FilterRouteTables(RouteTableFilterSubnet{SubnetId: subnet}, tables)
	if len(subnet_rtb) == 0 {
		main_rtbs := FilterRouteTables(RouteTableFilterMain{}, tables)
		if len(main_rtbs) == 0 {
			return nil
		}
		return main_rtbs[0]
	}
	return subnet_rtb[0]
}

type RouteTableFilterSubnet struct {
	SubnetId string
}

func (fs RouteTableFilterSubnet) Keep(rt *ec2.RouteTable) bool {
	for _, a := range rt.Associations {
		if a.SubnetId != nil && *(a.SubnetId) == fs.SubnetId {
			return true
		}
	}
	return false
}
