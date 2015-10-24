package aws

import (
	"errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/bobtfish/AWSnycast/healthcheck"
	"os"
	"testing"
)

var (
	rtb1 = ec2.RouteTable{
		RouteTableId: aws.String("rtb-f0ea3b95"),
		Routes: []*ec2.Route{
			&ec2.Route{
				DestinationCidrBlock: aws.String("172.17.16.0/22"),
				GatewayId:            aws.String("local"),
				Origin:               aws.String("CreateRouteTable"),
				State:                aws.String("active"),
			},
		},
		Tags: []*ec2.Tag{
			&ec2.Tag{
				Key:   aws.String("Name"),
				Value: aws.String("uswest1 devb private insecure"),
			}},
		VpcId: aws.String("vpc-9496cffc"),
	}

	rtb2 = ec2.RouteTable{
		Associations: []*ec2.RouteTableAssociation{
			&ec2.RouteTableAssociation{
				Main: aws.Bool(true),
				RouteTableAssociationId: aws.String("rtbassoc-b1f025d4"),
				RouteTableId:            aws.String("rtb-9696cffe"),
			},
			&ec2.RouteTableAssociation{
				Main: aws.Bool(false),
				RouteTableAssociationId: aws.String("rtbassoc-85c1cbe7"),
				RouteTableId:            aws.String("rtb-9696cffe"),
				SubnetId:                aws.String("subnet-16b0e97e"),
			},
			&ec2.RouteTableAssociation{
				Main: aws.Bool(false),
				RouteTableAssociationId: aws.String("rtbassoc-ba8573df"),
				RouteTableId:            aws.String("rtb-9696cffe"),
				SubnetId:                aws.String("subnet-3fb0e957"),
			},
			&ec2.RouteTableAssociation{
				Main: aws.Bool(false),
				RouteTableAssociationId: aws.String("rtbassoc-84c1cbe6"),
				RouteTableId:            aws.String("rtb-9696cffe"),
				SubnetId:                aws.String("subnet-28b0e940"),
			},
			&ec2.RouteTableAssociation{
				Main: aws.Bool(false),
				RouteTableAssociationId: aws.String("rtbassoc-858573e0"),
				RouteTableId:            aws.String("rtb-9696cffe"),
				SubnetId:                aws.String("subnet-f3b0e99b"),
			},
		},
		PropagatingVgws: []*ec2.PropagatingVgw{
			&ec2.PropagatingVgw{
				GatewayId: aws.String("vgw-d2396a97"),
			},
		},
		RouteTableId: aws.String("rtb-9696cffe"),
		Routes: []*ec2.Route{
			&ec2.Route{
				DestinationCidrBlock: aws.String("10.55.35.43/32"),
				GatewayId:            aws.String("vgw-d2396a97"),
				Origin:               aws.String("CreateRoute"),
				State:                aws.String("active"),
			},
			&ec2.Route{
				DestinationCidrBlock: aws.String("172.17.16.0/22"),
				GatewayId:            aws.String("local"),
				Origin:               aws.String("CreateRouteTable"),
				State:                aws.String("active"),
			},
			&ec2.Route{
				DestinationCidrBlock: aws.String("10.0.0.0/8"),
				InstanceId:           aws.String("i-446b201b"),
				InstanceOwnerId:      aws.String("613514870339"),
				NetworkInterfaceId:   aws.String("eni-ea8a9cac"),
				Origin:               aws.String("CreateRoute"),
				State:                aws.String("active"),
			},
			&ec2.Route{
				DestinationCidrBlock: aws.String("0.0.0.0/0"),
				InstanceId:           aws.String("i-605bd2aa"),
				InstanceOwnerId:      aws.String("613514870339"),
				NetworkInterfaceId:   aws.String("eni-09472250"),
				Origin:               aws.String("CreateRoute"),
				State:                aws.String("active"),
			},
			&ec2.Route{
				DestinationCidrBlock: aws.String("10.0.0.0/8"),
				GatewayId:            aws.String("vgw-d2396a97"),
				Origin:               aws.String("EnableVgwRoutePropagation"),
				State:                aws.String("active"),
			},
		},
		Tags: []*ec2.Tag{
			&ec2.Tag{
				Key:   aws.String("Name"),
				Value: aws.String("uswest1 devb private"),
			}},
		VpcId: aws.String("vpc-9496cffc"),
	}

	rtb3 = ec2.RouteTable{
		Associations: []*ec2.RouteTableAssociation{
			&ec2.RouteTableAssociation{
				Main: aws.Bool(false),
				RouteTableAssociationId: aws.String("rtbassoc-818573e4"),
				RouteTableId:            aws.String("rtb-019cab69"),
				SubnetId:                aws.String("subnet-37b0e95f"),
			},
			&ec2.RouteTableAssociation{
				Main: aws.Bool(false),
				RouteTableAssociationId: aws.String("rtbassoc-fd9cab95"),
				RouteTableId:            aws.String("rtb-019cab69"),
				SubnetId:                aws.String("subnet-44b0e92c"),
			},
		},
		PropagatingVgws: []*ec2.PropagatingVgw{
			&ec2.PropagatingVgw{
				GatewayId: aws.String("vgw-d2396a97"),
			},
		},
		RouteTableId: aws.String("rtb-019cab69"),
		Routes: []*ec2.Route{
			&ec2.Route{
				DestinationCidrBlock: aws.String("10.55.35.43/32"),
				GatewayId:            aws.String("vgw-d2396a97"),
				Origin:               aws.String("CreateRoute"),
				State:                aws.String("active"),
			},
			&ec2.Route{
				DestinationCidrBlock: aws.String("172.17.16.0/22"),
				GatewayId:            aws.String("local"),
				Origin:               aws.String("CreateRouteTable"),
				State:                aws.String("active"),
			},
			&ec2.Route{
				DestinationCidrBlock: aws.String("10.0.0.0/8"),
				InstanceId:           aws.String("i-446b201b"),
				InstanceOwnerId:      aws.String("613514870339"),
				NetworkInterfaceId:   aws.String("eni-ea8a9cac"),
				Origin:               aws.String("CreateRoute"),
				State:                aws.String("active"),
			},
			&ec2.Route{
				DestinationCidrBlock: aws.String("0.0.0.0/0"),
				GatewayId:            aws.String("igw-9ab1e8f2"),
				Origin:               aws.String("CreateRoute"),
				State:                aws.String("active"),
			},
			&ec2.Route{
				DestinationCidrBlock: aws.String("10.0.0.0/8"),
				GatewayId:            aws.String("vgw-d2396a97"),
				Origin:               aws.String("EnableVgwRoutePropagation"),
				State:                aws.String("active"),
			},
		},
		Tags: []*ec2.Tag{
			&ec2.Tag{
				Key:   aws.String("Name"),
				Value: aws.String("uswest1 devb public"),
			},
		},
		VpcId: aws.String("vpc-9496cffc"),
	}

	rtb4 = ec2.RouteTable{
		RouteTableId: aws.String("rtb-f1ea3b94"),
		Routes: []*ec2.Route{
			&ec2.Route{
				DestinationCidrBlock: aws.String("172.17.16.0/22"),
				GatewayId:            aws.String("local"),
				Origin:               aws.String("CreateRouteTable"),
				State:                aws.String("active"),
			},
			&ec2.Route{
				DestinationCidrBlock: aws.String("0.0.0.0/0"),
				GatewayId:            aws.String("igw-9ab1e8f2"),
				Origin:               aws.String("CreateRoute"),
				State:                aws.String("active"),
			},
		},
		Tags: []*ec2.Tag{
			&ec2.Tag{
				Key:   aws.String("Name"),
				Value: aws.String("uswest1 devb public insecure"),
			},
		},
		VpcId: aws.String("vpc-9496cffc"),
	}

	rtb5 = ec2.RouteTable{
		RouteTableId: aws.String("rtb-f0ea3b96"),
		Routes: []*ec2.Route{
			&ec2.Route{
				DestinationCidrBlock: aws.String("0.0.0.0/0"),
				InstanceId:           aws.String("i-605bd2ab"),
				InstanceOwnerId:      aws.String("613514870339"),
				NetworkInterfaceId:   aws.String("eni-09472251"),
				Origin:               aws.String("CreateRoute"),
				State:                aws.String("inactive"),
			},
		},
		VpcId: aws.String("vpc-9496cffc"),
	}
)

func TestMetaDataFetcher(t *testing.T) {
	_ = NewMetadataFetcher(false)
	_ = NewMetadataFetcher(true)
}

type FakeRouteTableFetcher struct {
	Error  error
	Routes []*ec2.RouteTable
}

func (r FakeRouteTableFetcher) GetRouteTables() ([]*ec2.RouteTable, error) {
	return r.Routes, r.Error
}

func (r FakeRouteTableFetcher) ManageInstanceRoute(rtb ec2.RouteTable, rs ManageRoutesSpec, noop bool) error {
	return nil
}

func TestFakeFetcher(t *testing.T) {
	var f RouteTableFetcher
	f = FakeRouteTableFetcher{
		Routes: []*ec2.RouteTable{&rtb1},
	}
	rtb, err := f.GetRouteTables()
	if err != nil {
		t.Fail()
	}
	if len(rtb) != 1 {
		t.Fail()
	}
	if rtb[0] != &rtb1 {
		t.Fail()
	}
}

func TestRouteTableFilterAlways(t *testing.T) {
	f := RouteTableFilterAlways{}
	if f.Keep(&rtb1) {
		t.Fail()
	}
}

func TestRouteTableFilterNever(t *testing.T) {
	f := RouteTableFilterNever{}
	if !f.Keep(&rtb1) {
		t.Fail()
	}
}

func TestRouteTableFilterAndTwoNever(t *testing.T) {
	f := RouteTableFilterAnd{
		RouteTableFilters: []RouteTableFilter{
			RouteTableFilterNever{},
			RouteTableFilterNever{},
		},
	}
	if !f.Keep(&rtb1) {
		t.Fail()
	}
}

func TestRouteTableFilterAndOneNever(t *testing.T) {
	f := RouteTableFilterAnd{
		RouteTableFilters: []RouteTableFilter{
			RouteTableFilterNever{},
			RouteTableFilterAlways{},
		},
	}
	if f.Keep(&rtb1) {
		t.Fail()
	}
}

func TestRouteTableFilterOrOneNever(t *testing.T) {
	f := RouteTableFilterOr{
		RouteTableFilters: []RouteTableFilter{
			RouteTableFilterNever{},
			RouteTableFilterAlways{},
		},
	}
	if !f.Keep(&rtb1) {
		t.Fail()
	}
}

func TestRouteTableFilterOrOneNever2(t *testing.T) {
	f := RouteTableFilterOr{
		RouteTableFilters: []RouteTableFilter{
			RouteTableFilterAlways{},
			RouteTableFilterNever{},
		},
	}
	if !f.Keep(&rtb1) {
		t.Fail()
	}
}

func TestRouteTableFilterOrAlways(t *testing.T) {
	f := RouteTableFilterOr{
		RouteTableFilters: []RouteTableFilter{
			RouteTableFilterAlways{},
			RouteTableFilterAlways{},
		},
	}
	if f.Keep(&rtb1) {
		t.Fail()
	}
}

func TestFilterRouteTables(t *testing.T) {
	rtb := FilterRouteTables(RouteTableFilterNever{}, []*ec2.RouteTable{&rtb1})
	if len(rtb) != 1 {
		t.Fail()
	}
	if rtb[0] != &rtb1 {
		t.Fail()
	}
}

func TestRouteTableFilterMain(t *testing.T) {
	f := RouteTableFilterMain{}
	if f.Keep(&rtb1) {
		t.Fail()
	}
	if !f.Keep(&rtb2) {
		t.Fail()
	}
}

func TestRoutTableFilterSubnet(t *testing.T) {
	f := RouteTableFilterSubnet{
		SubnetId: "subnet-28b0e940",
	}
	if f.Keep(&rtb1) {
		t.Fail()
	}
	if !f.Keep(&rtb2) {
		t.Fail()
	}
}

func TestRouteTableForSubnetExplicitAssociation(t *testing.T) {
	rt := RouteTableForSubnet("subnet-37b0e95f", []*ec2.RouteTable{&rtb1, &rtb2, &rtb3, &rtb4})
	if rt == nil || rt != &rtb3 {
		t.Fail()
	}
}

func TestRouteTableForSubnetDefaultMain(t *testing.T) {
	rt := RouteTableForSubnet("subnet-38b0e95f", []*ec2.RouteTable{&rtb1, &rtb2, &rtb3, &rtb4})
	if rt == nil || rt != &rtb2 {
		t.Fail()
	}
}

func TestRouteTableForSubnetNone(t *testing.T) {
	rt := RouteTableForSubnet("subnet-38b0e95f", []*ec2.RouteTable{&rtb1, &rtb3, &rtb4})
	if rt != nil {
		t.Fail()
	}
}

func TestRouteTableFilterDestinationCidrBlock(t *testing.T) {
	f := RouteTableFilterDestinationCidrBlock{
		DestinationCidrBlock: "0.0.0.0/0",
	}
	if f.Keep(&rtb1) {
		t.Fail()
	}
	if !f.Keep(&rtb2) {
		t.Fail()
	}
}

func TestRouteTableFilterDestinationCidrBlockViaIGW(t *testing.T) {
	f := RouteTableFilterDestinationCidrBlock{
		DestinationCidrBlock: "0.0.0.0/0",
		ViaIGW:               true,
	}
	if f.Keep(&rtb2) {
		t.Fail()
	}
	if !f.Keep(&rtb4) {
		t.Fail()
	}
}

func TestRouteTableFilterDestinationCidrBlockViaInstance(t *testing.T) {
	f := RouteTableFilterDestinationCidrBlock{
		DestinationCidrBlock: "0.0.0.0/0",
		ViaInstance:          true,
	}
	/* Via IGW */
	if f.Keep(&rtb4) {
		t.Fail()
	}
	/* Via instance */
	if !f.Keep(&rtb2) {
		t.Fail()
	}
}

func TestRouteTableFilterDestinationCidrBlockViaInstanceInactive(t *testing.T) {
	f := RouteTableFilterDestinationCidrBlock{
		DestinationCidrBlock: "0.0.0.0/0",
		ViaInstance:          true,
		InstanceNotActive:    true,
	}
	if f.Keep(&rtb2) {
		t.Fail()
	}
	if !f.Keep(&rtb5) {
		t.Fail()
	}
}

func TestRouteTableFilterTagMatch(t *testing.T) {
	f := RouteTableFilterTagMatch{
		Key:   "Name",
		Value: "uswest1 devb private insecure",
	}
	if f.Keep(&rtb2) {
		t.Fail()
	}
	if !f.Keep(&rtb1) {
		t.Fail()
	}
}

func TestGetCreateRouteInput(t *testing.T) {
	rtb := ec2.RouteTable{RouteTableId: aws.String("rtb-1234")}
	in := getCreateRouteInput(rtb, "0.0.0.0/0", "i-12345", false)
	if *(in.RouteTableId) != "rtb-1234" {
		t.Fail()
	}
	if *(in.DestinationCidrBlock) != "0.0.0.0/0" {
		t.Fail()
	}
	if *(in.InstanceId) != "i-12345" {
		t.Fail()
	}
	if *(in.DryRun) != false {
		t.Fail()
	}
}

func TestGetCreateRouteInputDryRun(t *testing.T) {
	rtb := ec2.RouteTable{RouteTableId: aws.String("rtb-1234")}
	in := getCreateRouteInput(rtb, "0.0.0.0/0", "i-12345", true)
	if *(in.DryRun) != true {
		t.Fail()
	}
}

func NewFakeEC2Conn() *FakeEC2Conn {
	return &FakeEC2Conn{
		DescribeRouteTablesOutput: &ec2.DescribeRouteTablesOutput{
			RouteTables: make([]*ec2.RouteTable, 0),
		},
	}
}

type FakeEC2Conn struct {
	CreateRouteOutput         *ec2.CreateRouteOutput
	CreateRouteError          error
	CreateRouteInput          *ec2.CreateRouteInput
	ReplaceRouteOutput        *ec2.ReplaceRouteOutput
	ReplaceRouteError         error
	ReplaceRouteInput         *ec2.ReplaceRouteInput
	DeleteRouteInput          *ec2.DeleteRouteInput
	DeleteRouteOutput         *ec2.DeleteRouteOutput
	DeleteRouteError          error
	DescribeRouteTablesInput  *ec2.DescribeRouteTablesInput
	DescribeRouteTablesOutput *ec2.DescribeRouteTablesOutput
	DescribeRouteTablesError  error
}

func (f *FakeEC2Conn) CreateRoute(i *ec2.CreateRouteInput) (*ec2.CreateRouteOutput, error) {
	f.CreateRouteInput = i
	return f.CreateRouteOutput, f.CreateRouteError
}
func (f *FakeEC2Conn) ReplaceRoute(i *ec2.ReplaceRouteInput) (*ec2.ReplaceRouteOutput, error) {
	f.ReplaceRouteInput = i
	return f.ReplaceRouteOutput, f.ReplaceRouteError
}
func (f *FakeEC2Conn) DeleteRoute(i *ec2.DeleteRouteInput) (*ec2.DeleteRouteOutput, error) {
	f.DeleteRouteInput = i
	return f.DeleteRouteOutput, f.DeleteRouteError
}
func (f *FakeEC2Conn) DescribeRouteTables(i *ec2.DescribeRouteTablesInput) (*ec2.DescribeRouteTablesOutput, error) {
	f.DescribeRouteTablesInput = i
	return f.DescribeRouteTablesOutput, f.DescribeRouteTablesError
}

func TestRouteTableFetcherEC2ReplaceInstanceRouteNoop(t *testing.T) {
	rtf := RouteTableFetcherEC2{conn: NewFakeEC2Conn()}
	route := findRouteFromRouteTable(rtb2, "0.0.0.0/0")
	if route == nil {
		t.Fail()
	}
	err := rtf.ReplaceInstanceRoute(rtb2.RouteTableId, route, "0.0.0.0/0", "i-1234", false, true)
	if err != nil {
		t.Log(err)
		t.Fail()
	}
	if rtf.conn.(*FakeEC2Conn).ReplaceRouteInput == nil {
		t.Fail()
	}
	// Should *not* have actually tried to replace the route - dry run mode
	r := rtf.conn.(*FakeEC2Conn).ReplaceRouteInput
	if *(r.DryRun) != true {
		t.Fail()
	}
}

func TestRouteTableFetcherEC2ReplaceInstanceRoute(t *testing.T) {
	rtf := RouteTableFetcherEC2{conn: NewFakeEC2Conn()}
	route := findRouteFromRouteTable(rtb2, "0.0.0.0/0")
	if route == nil {
		t.Fail()
	}
	err := rtf.ReplaceInstanceRoute(rtb2.RouteTableId, route, "0.0.0.0/0", "i-1234", false, false)
	if err != nil {
		t.Fail()
	}
	if rtf.conn.(*FakeEC2Conn).ReplaceRouteInput == nil {
		t.Log("ReplaceRouteInput == nil")
		t.Fail()
	}
	r := rtf.conn.(*FakeEC2Conn).ReplaceRouteInput
	if *(r.DestinationCidrBlock) != "0.0.0.0/0" {
		t.Fail()
	}
	if *(r.RouteTableId) != *(rtb2.RouteTableId) {
		t.Fail()
	}
	if *(r.InstanceId) != "i-1234" {
		t.Fail()
	}
}

func TestRouteTableFetcherEC2ReplaceInstanceRouteFails(t *testing.T) {
	rtf := RouteTableFetcherEC2{conn: NewFakeEC2Conn()}
	rtf.conn.(*FakeEC2Conn).ReplaceRouteError = errors.New("Whoops, AWS blew up")
	route := findRouteFromRouteTable(rtb2, "0.0.0.0/0")
	if route == nil {
		t.Fail()
	}
	err := rtf.ReplaceInstanceRoute(rtb2.RouteTableId, route, "0.0.0.0/0", "i-1234", false, false)
	if err == nil {
		t.Fail()
	}
	if err.Error() != "Whoops, AWS blew up" {
		t.Log(err)
		t.Fail()
	}
	if rtf.conn.(*FakeEC2Conn).ReplaceRouteInput == nil {
		t.Log("ReplaceRouteInput == nil")
		t.Fail()
	}
}

func TestRouteTableFetcherEC2ReplaceInstanceRouteNotIfHealthy(t *testing.T) {
	rtf := RouteTableFetcherEC2{conn: NewFakeEC2Conn()}
	route := findRouteFromRouteTable(rtb2, "0.0.0.0/0")
	if route == nil {
		t.Fail()
	}
	err := rtf.ReplaceInstanceRoute(rtb2.RouteTableId, route, "0.0.0.0/0", "i-1234", true, false)
	if err != nil {
		t.Fail()
	}
	if rtf.conn.(*FakeEC2Conn).ReplaceRouteInput != nil {
		t.Log("ReplaceRouteInput != nil")
		t.Fail()
	}
}

func TestRouteTableFetcherEC2ManageInstanceRouteAlreadyThisInstance(t *testing.T) {
	rtf := RouteTableFetcherEC2{conn: NewFakeEC2Conn()}
	s := ManageRoutesSpec{
		Cidr:        "0.0.0.0/0",
		Instance:    "i-605bd2aa",
		IfUnhealthy: false,
	}
	err := rtf.ManageInstanceRoute(rtb2, s, false)
	if err != nil {
		t.Fail()
	}
	if rtf.conn.(*FakeEC2Conn).ReplaceRouteInput != nil {
		t.Log("ReplaceRouteInput != nil")
		t.Fail()
	}
}

func TestManageInstanceRoute(t *testing.T) {
	rtf := RouteTableFetcherEC2{conn: NewFakeEC2Conn()}
	s := ManageRoutesSpec{
		Cidr:        "0.0.0.0/0",
		Instance:    "i-1234",
		IfUnhealthy: false,
	}
	err := rtf.ManageInstanceRoute(rtb2, s, false)
	if err != nil {
		t.Fail()
	}
	if rtf.conn.(*FakeEC2Conn).ReplaceRouteInput == nil {
		t.Log("ReplaceRouteInput == nil")
		t.Fail()
	}
	r := rtf.conn.(*FakeEC2Conn).ReplaceRouteInput
	if *(r.DestinationCidrBlock) != "0.0.0.0/0" {
		t.Fail()
	}
	if *(r.RouteTableId) != *(rtb2.RouteTableId) {
		t.Fail()
	}
	if *(r.InstanceId) != "i-1234" {
		t.Fail()
	}
}

func TestManageInstanceRouteAWSFailOnReplace(t *testing.T) {
	rtf := RouteTableFetcherEC2{conn: NewFakeEC2Conn()}
	rtf.conn.(*FakeEC2Conn).ReplaceRouteError = errors.New("Whoops, AWS blew up")
	s := ManageRoutesSpec{
		Cidr:        "0.0.0.0/0",
		Instance:    "i-1234",
		IfUnhealthy: false,
	}
	err := rtf.ManageInstanceRoute(rtb2, s, false)
	if err == nil {
		t.Fail()
	}
	if err.Error() != "Whoops, AWS blew up" {
		t.Fail()
	}
}

func TestManageInstanceRouteAWSFailOnCreate(t *testing.T) {
	rtf := RouteTableFetcherEC2{conn: NewFakeEC2Conn()}
	rtf.conn.(*FakeEC2Conn).CreateRouteError = errors.New("Whoops, AWS blew up")
	s := ManageRoutesSpec{
		Cidr:        "0.0.0.0/0",
		Instance:    "i-1234",
		IfUnhealthy: false,
	}
	err := rtf.ManageInstanceRoute(rtb1, s, false)
	if err == nil {
		t.Fail()
	}
	if err.Error() != "Whoops, AWS blew up" {
		t.Fail()
	}
}

func TestManageInstanceRouteCreateRoute(t *testing.T) {
	rtf := RouteTableFetcherEC2{conn: NewFakeEC2Conn()}
	s := ManageRoutesSpec{
		Cidr:        "0.0.0.0/0",
		Instance:    "i-1234",
		IfUnhealthy: false,
	}
	err := rtf.ManageInstanceRoute(rtb1, s, false)
	if err != nil {
		t.Fail()
	}
	if rtf.conn.(*FakeEC2Conn).CreateRouteInput == nil {
		t.Log("rtf.conn.(*FakeEC2Conn).CreateRoute was never called")
		t.Fail()
	}
	in := rtf.conn.(*FakeEC2Conn).CreateRouteInput
	if *(in.RouteTableId) != *(rtb1.RouteTableId) {
		t.Fail()
	}
	if *(in.DestinationCidrBlock) != "0.0.0.0/0" {
		t.Fail()
	}
	if *(in.InstanceId) != "i-1234" {
		t.Fail()
	}
}

func TestGetRouteTables(t *testing.T) {
	rtf := RouteTableFetcherEC2{conn: NewFakeEC2Conn()}
	_, err := rtf.GetRouteTables()
	if err != nil {
		t.Fail()
	}
	if rtf.conn.(*FakeEC2Conn).DescribeRouteTablesInput == nil {
		t.Log("rtf.conn.(*FakeEC2Conn).DescribeRouteTablesInput was never set")
		t.Fail()
	}
}

func TestGetRouteTablesAWSFail(t *testing.T) {
	rtf := RouteTableFetcherEC2{conn: NewFakeEC2Conn()}
	rtf.conn.(*FakeEC2Conn).DescribeRouteTablesError = errors.New("Whoops, AWS blew up")
	_, err := rtf.GetRouteTables()
	if err == nil {
		t.Fail()
	}
	if err.Error() != "Whoops, AWS blew up" {
		t.Log(err)
		t.Fail()
	}
	if rtf.conn.(*FakeEC2Conn).DescribeRouteTablesInput == nil {
		t.Log("rtf.conn.(*FakeEC2Conn).DescribeRouteTablesInput was never called")
		t.Fail()
	}
}

func TestNewRouteTableFetcher(t *testing.T) {
	rtf := NewRouteTableFetcher("us-west-1", false)
	if rtf == nil {
		t.Fail()
	}
	if rtf.(RouteTableFetcherEC2).conn == nil {
		t.Fail()
	}
}

func TestManageRoutesSpecDefault(t *testing.T) {
	u := &ManageRoutesSpec{
		Cidr: "127.0.0.1",
	}
	u.Default("i-1234")
	if u.Cidr != "127.0.0.1/32" {
		t.Log("Not canonicalized in ManageRoutesSpecDefault")
		t.Fail()
	}
	if u.Instance != "SELF" {
		t.Log("Instance not defaulted to SELF")
	}
}

func TestManageRoutesSpecValidateBadInstance(t *testing.T) {
	r := &ManageRoutesSpec{
		Instance: "vpc-1234",
		Cidr:     "127.0.0.1",
	}
	h := make(map[string]*healthcheck.Healthcheck)
	err := r.Validate("foo", h)
	if err == nil {
		t.Fail()
	}
	if err.Error() != "Could not parse invalid CIDR address: 127.0.0.1 in foo" {
		t.Log(err.Error())
		t.Fail()
	}
}

func TestManageRoutesSpecValidateMissingCidr(t *testing.T) {
	r := ManageRoutesSpec{
		Instance: "SELF",
	}
	h := make(map[string]*healthcheck.Healthcheck)
	err := r.Validate("foo", h)
	if err == nil {
		t.Fail()
	}
	if err.Error() != "cidr is not defined in foo" {
		t.Log(err.Error())
		t.Fail()
	}
}

func TestManageRoutesSpecValidateBadCidr1(t *testing.T) {
	r := ManageRoutesSpec{
		Cidr:     "300.0.0.0/16",
		Instance: "SELF",
	}
	h := make(map[string]*healthcheck.Healthcheck)
	err := r.Validate("foo", h)
	if err == nil {
		t.Fail()
	}
	if err.Error() != "Could not parse invalid CIDR address: 300.0.0.0/16 in foo" {
		t.Log(err.Error())
		t.Fail()
	}
}

func TestManageRoutesSpecValidateBadCidr2(t *testing.T) {
	r := ManageRoutesSpec{
		Cidr:     "3.0.0.0/160",
		Instance: "SELF",
	}
	h := make(map[string]*healthcheck.Healthcheck)
	err := r.Validate("foo", h)
	if err == nil {
		t.Fail()
	}
	if err.Error() != "Could not parse invalid CIDR address: 3.0.0.0/160 in foo" {
		t.Log(err.Error())
		t.Fail()
	}
}

func TestManageRoutesSpecValidateBadCidr3(t *testing.T) {
	r := ManageRoutesSpec{
		Cidr:     "foo",
		Instance: "SELF",
	}
	h := make(map[string]*healthcheck.Healthcheck)
	err := r.Validate("bar", h)
	if err == nil {
		t.Fail()
	}
	if err.Error() != "Could not parse invalid CIDR address: foo in bar" {
		t.Log(err.Error())
		t.Fail()
	}
}

func TestManageRoutesSpecValidate(t *testing.T) {
	r := ManageRoutesSpec{
		Cidr:     "0.0.0.0/0",
		Instance: "SELF",
	}
	h := make(map[string]*healthcheck.Healthcheck)
	err := r.Validate("foo", h)
	if err != nil {
		t.Log(err)
		t.Fail()
	}
}

func TestManageRoutesSpecValidateMissingHealthcheck(t *testing.T) {
	r := ManageRoutesSpec{
		Cidr:            "0.0.0.0/0",
		Instance:        "SELF",
		HealthcheckName: "test",
	}
	h := make(map[string]*healthcheck.Healthcheck)
	err := r.Validate("foo", h)
	if err == nil {
		t.Fail()
	} else {
		if err.Error() != "Route table foo, upsert 0.0.0.0/0 cannot find healthcheck 'test'" {
			t.Log(err)
			t.Fail()
		}
	}
}

func TestManageRoutesSpecValidateWithHealthcheck(t *testing.T) {
	r := ManageRoutesSpec{
		Cidr:            "0.0.0.0/0",
		Instance:        "SELF",
		HealthcheckName: "test",
	}
	h := make(map[string]*healthcheck.Healthcheck)
	h["test"] = &healthcheck.Healthcheck{}
	err := r.Validate("foo", h)
	if err != nil {
		t.Log(err)
		t.Fail()
	} else {
		if h["test"] != r.healthcheck {
			t.Log("r.healthcheck not set")
			t.Fail()
		}
	}
}

func TestManageRouteSpecDefaultInstanceSELF(t *testing.T) {
	urs := ManageRoutesSpec{
		Cidr:     "127.0.0.1",
		Instance: "SELF",
	}
	urs.Default("i-other")
	if urs.Instance != "i-other" {
		t.Fail()
	}
}

func TestManageRouteSpecDefaultInstanceOther(t *testing.T) {
	urs := ManageRoutesSpec{
		Cidr:     "127.0.0.1",
		Instance: "i-foo",
	}
	urs.Default("i-other")
	if urs.Instance != "i-foo" {
		t.Fail()
	}
}

type FakeHealthCheck struct {
	isHealthy bool
}

func (h *FakeHealthCheck) IsHealthy() bool {
	return h.isHealthy
}

func (h *FakeHealthCheck) GetListener() <-chan bool {
	return make(chan bool)
}

func TestManageInstanceRouteNoCreateRouteBadHealthcheck(t *testing.T) {
	rtf := RouteTableFetcherEC2{conn: NewFakeEC2Conn()}
	s := ManageRoutesSpec{
		Cidr:            "0.0.0.0/0",
		Instance:        "i-1234",
		IfUnhealthy:     false,
		HealthcheckName: "foo",
		healthcheck:     &FakeHealthCheck{isHealthy: false},
	}
	err := rtf.ManageInstanceRoute(rtb1, s, false)
	if err != nil {
		t.Fail()
	}
	if rtf.conn.(*FakeEC2Conn).CreateRouteInput != nil {
		t.Log("rtf.conn.(*FakeEC2Conn).CreateRoute was called")
		t.Fail()
	}
}

func TestManageInstanceRouteCreateRouteGoodHealthcheck(t *testing.T) {
	rtf := RouteTableFetcherEC2{conn: NewFakeEC2Conn()}
	s := ManageRoutesSpec{
		Cidr:            "0.0.0.0/0",
		Instance:        "i-1234",
		IfUnhealthy:     false,
		HealthcheckName: "foo",
		healthcheck:     &FakeHealthCheck{isHealthy: true},
	}
	err := rtf.ManageInstanceRoute(rtb1, s, false)
	if err != nil {
		t.Fail()
	}
	if rtf.conn.(*FakeEC2Conn).CreateRouteInput == nil {
		t.Log("rtf.conn.(*FakeEC2Conn).CreateRoute was not called")
		t.Fail()
	}
}

func TestManageInstanceRouteDeleteInstanceRouteThisInstanceUnhealthy(t *testing.T) {
	rtf := RouteTableFetcherEC2{conn: NewFakeEC2Conn()}
	s := ManageRoutesSpec{
		Cidr:            "0.0.0.0/0",
		Instance:        "i-605bd2aa",
		IfUnhealthy:     false,
		HealthcheckName: "localhealthcheck",
		healthcheck:     &FakeHealthCheck{isHealthy: false},
	}
	err := rtf.ManageInstanceRoute(rtb2, s, false)
	if err != nil {
		t.Fail()
	}
	if rtf.conn.(*FakeEC2Conn).ReplaceRouteInput != nil {
		t.Log("ReplaceRouteInput was called")
		t.Fail()
	}
	if rtf.conn.(*FakeEC2Conn).DeleteRouteInput == nil {
		t.Log("DeleteRouteInput was never called")
		t.Fail()
	}
	r := rtf.conn.(*FakeEC2Conn).DeleteRouteInput
	if *(r.DestinationCidrBlock) != "0.0.0.0/0" {
		t.Fail()
	}
	if *(r.RouteTableId) != *(rtb2.RouteTableId) {
		t.Fail()
	}
}

func TestManageInstanceRouteDeleteInstanceRouteThisInstanceUnhealthyAWSFail(t *testing.T) {
	rtf := RouteTableFetcherEC2{conn: NewFakeEC2Conn()}
	rtf.conn.(*FakeEC2Conn).DeleteRouteError = errors.New("Whoops, AWS blew up")
	s := ManageRoutesSpec{
		Cidr:            "0.0.0.0/0",
		Instance:        "i-605bd2aa",
		IfUnhealthy:     false,
		HealthcheckName: "localhealthcheck",
		healthcheck:     &FakeHealthCheck{isHealthy: false},
	}
	err := rtf.ManageInstanceRoute(rtb2, s, false)
	if err == nil {
		t.Fail()
	} else {
		if err.Error() != "Whoops, AWS blew up" {
			t.Log(err)
			t.Fail()
		}
	}
}

func TestGetCredFail(t *testing.T) {
	os.Setenv("AWS_ACCESS_KEY_ID", "")
	os.Setenv("AWS_ACCESS_KEY", "")
	defer func() {
		err := recover()
		if err == nil {
			t.Fail()
		}
		if err.(awserr.Error).Error() != "NoCredentialProviders: no valid providers in chain" {
			t.Log(err)
			t.Fail()
		}
	}()
	getCred([]credentials.Provider{&credentials.EnvProvider{}})
}

func TestEc2RouteTablesDefault(t *testing.T) {
	rs := &ManageRoutesSpec{
		Cidr: "127.0.0.1",
	}
	rs.Default("i-1234")
	if rs.ec2RouteTables == nil {
		t.Fail()
	}
}

func TestUpdateEc2RouteTables(t *testing.T) {
	rs := &ManageRoutesSpec{}
	rs.UpdateEc2RouteTables([]*ec2.RouteTable{})
	if rs.ec2RouteTables == nil {
		t.Fail()
	}
}
