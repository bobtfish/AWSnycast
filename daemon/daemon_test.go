package daemon

import (
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/bobtfish/AWSnycast/config"
	"testing"
)

type FakeMetadataFetcher struct {
	FAvailable bool
	Meta       map[string]string
}

func (m FakeMetadataFetcher) Available() bool {
	return m.FAvailable
}

func (m FakeMetadataFetcher) GetMetadata(key string) (string, error) {
	v, ok := m.Meta[key]
	if ok {
		return v, nil
	}
	return v, errors.New(fmt.Sprintf("Key %s unknown", key))
}

func NewFakeRouteTableFetcher() *FakeRouteTableFetcher {
	f := &FakeRouteTableFetcher{}
	f.Tables = make([]*ec2.RouteTable, 0)
	return f
}

type FakeRouteTableFetcher struct {
	Tables                            []*ec2.RouteTable
	Error                             error
	RouteTable                        ec2.RouteTable
	Cidr                              string
	Instance                          string
	IfUnhealthy                       bool
	Noop                              bool
	CreateOrReplaceInstanceRouteError error
}

func (f *FakeRouteTableFetcher) GetRouteTables() ([]*ec2.RouteTable, error) {
	return f.Tables, f.Error
}

func (f *FakeRouteTableFetcher) CreateOrReplaceInstanceRoute(rtb ec2.RouteTable, cidr string, instance string, ifunhealthy bool, noop bool) error {
	f.RouteTable = rtb
	f.Cidr = cidr
	f.Instance = instance
	f.IfUnhealthy = ifunhealthy
	f.Noop = noop
	return f.CreateOrReplaceInstanceRouteError
}

func TestRunRouteTablesFailGetRouteTables(t *testing.T) {
	d := getD(true)
	rtf := d.RouteTableFetcher.(*FakeRouteTableFetcher)
	rtf.Error = errors.New("Route table get fail")
	err := d.RunRouteTables()
	if err == nil {
		t.Fail()
	} else {
		if err.Error() != "Route table get fail" {
			t.Fail()
		}
	}
}

func TestSetupNoMetadataService(t *testing.T) {
	fakeM := FakeMetadataFetcher{
		FAvailable: false,
	}
	d := Daemon{
		ConfigFile: "../tests/awsnycast.yaml",
	}
	if d.RouteTableFetcher != nil {
		t.Fail()
	}
	d.MetadataFetcher = fakeM

	err := d.Setup()
	if err == nil {
		t.Log(err)
		t.Fail()
	}
	if err.Error() != "No metadata service" {
		t.Fail()
	}
}

func TestSetupNormal(t *testing.T) {
	fakeM := FakeMetadataFetcher{
		FAvailable: true,
	}
	fakeM.Meta = make(map[string]string)
	fakeM.Meta["placement/availability-zone"] = "us-west-1a"
	fakeM.Meta["instance-id"] = "i-1234"
	fakeM.Meta["mac"] = "06:1d:ea:6f:8c:6e"
	fakeM.Meta["network/interfaces/macs/06:1d:ea:6f:8c:6e/subnet-id"] = "subnet-28b0e940"
	d := Daemon{
		ConfigFile: "../tests/awsnycast.yaml",
	}
	if d.RouteTableFetcher != nil {
		t.Fail()
	}
	d.MetadataFetcher = fakeM

	err := d.Setup()
	if err != nil {
		t.Log(err)
		t.Fail()
	}
}

func getD(a bool) Daemon {
	d := Daemon{
		ConfigFile: "../tests/awsnycast.yaml",
		Config:     &config.Config{},
	}
	d.Config.Default()
	fakeM := FakeMetadataFetcher{
		FAvailable: a,
	}
	fakeR := NewFakeRouteTableFetcher()
	fakeM.Meta = make(map[string]string)
	fakeM.Meta["placement/availability-zone"] = "us-west-1a"
	fakeM.Meta["instance-id"] = "i-1234"
	fakeM.Meta["mac"] = "06:1d:ea:6f:8c:6e"
	fakeM.Meta["network/interfaces/macs/06:1d:ea:6f:8c:6e/subnet-id"] = "subnet-28b0e940"
	d.MetadataFetcher = fakeM
	d.RouteTableFetcher = fakeR
	return d
}

func TestSetupBadConfigFile(t *testing.T) {
	d := getD(false)
	d.ConfigFile = "../tests/doesnotexist.yaml"
	err := d.Setup()
	if err == nil {
		t.Fail()
	}
	if err.Error() != "open ../tests/doesnotexist.yaml: no such file or directory" {
		t.Log(err)
		t.Fail()
	}
	success := d.Run(true, false)
	if success != 1 {
		t.Fail()
	}
}

func TestSetupUnavailable(t *testing.T) {
	d := getD(false)
	err := d.Setup()
	if err == nil {
		t.Fail()
	}
	if d.MetadataFetcher.Available() {
		t.Fail()
	}
}

func TestSetupAvailable(t *testing.T) {
	d := getD(true)
	err := d.Setup()
	if err != nil {
		t.Fail()
	}
	if !d.MetadataFetcher.Available() {
		t.Fail()
	}
}

func TestSetupNoAZ(t *testing.T) {
	d := getD(true)
	delete(d.MetadataFetcher.(FakeMetadataFetcher).Meta, "placement/availability-zone")
	err := d.Setup()
	if err == nil {
		t.Fail()
	}
	if err.Error() != "Error getting AZ: Key placement/availability-zone unknown" {
		t.Log(err)
		t.Fail()
	}
}

func TestSetupNoInstanceId(t *testing.T) {
	d := getD(true)
	delete(d.MetadataFetcher.(FakeMetadataFetcher).Meta, "instance-id")
	err := d.Setup()
	if err == nil {
		t.Fail()
	}
	if err.Error() != "Error getting instance-id: Key instance-id unknown" {
		t.Log(err)
		t.Fail()
	}
}

func TestSetupNoSubnetId(t *testing.T) {
	d := getD(true)
	delete(d.MetadataFetcher.(FakeMetadataFetcher).Meta, "mac")
	err := d.Setup()
	if err == nil {
		t.Fail()
	}
	if err.Error() != "Error getting metadata: Key mac unknown" {
		t.Log(err)
		t.Fail()
	}
}

func TestSetupRegionFromAZ(t *testing.T) {
	d := getD(true)
	err := d.Setup()
	if err != nil {
		t.Fail()
	}
	if d.Region != "us-west-1" {
		t.Fail()
	}
}

func TestSetupInstance(t *testing.T) {
	d := getD(true)
	err := d.Setup()
	if err != nil {
		t.Fail()
	}
	if d.Instance != "i-1234" {
		t.Fail()
	}
}

func TestSetupHealthChecks(t *testing.T) {
	d := getD(true)
	d.Debug = true
	err := d.Setup()
	if err != nil {
		t.Fail()
	}
	if d.Config.Healthchecks["public"].IsRunning() {
		t.Log("HealthChecks already running")
		t.Fail()
	}
	d.runHealthChecks()
	if !d.Config.Healthchecks["public"].IsRunning() {
		t.Log("HealthChecks did not start running")
		t.Fail()
	}
	d.stopHealthChecks()
	if d.Config.Healthchecks["public"].IsRunning() {
		t.Log("HealthChecks still running")
		t.Fail()
	}
}

func TestGetSubnetIdMacFail(t *testing.T) {
	d := getD(true)
	delete(d.MetadataFetcher.(FakeMetadataFetcher).Meta, "mac")
	_, err := d.GetSubnetId()
	if err == nil {
		t.Fail()
	}
}

func TestGetSubnetIdMacFail2(t *testing.T) {
	d := getD(true)
	delete(d.MetadataFetcher.(FakeMetadataFetcher).Meta, "network/interfaces/macs/06:1d:ea:6f:8c:6e/subnet-id")
	_, err := d.GetSubnetId()
	if err == nil {
		t.Fail()
	}
}

func TestGetSubnetIdMacOk(t *testing.T) {
	d := getD(true)
	val, err := d.GetSubnetId()
	if err != nil {
		t.Fail()
	}
	if val != "subnet-28b0e940" {
		t.Fail()
	}
}

func TestHealthCheckOneUpsertRouteNilConfigPanic(t *testing.T) {
	d := getD(true)
	d.Config = nil
	defer func() {
		err := recover()
		if err == nil {
			t.Fail()
		}
		if err.(string) != "No healthchecks, have you run Setup()?" {
			t.Fail()
		}
	}()
	d.HealthCheckOneUpsertRoute("foo", &config.UpsertRoutesSpec{Healthcheck: "foo"})
}

func TestHealthCheckOneUpsertRouteNilConfigHealthchecksPanic(t *testing.T) {
	d := getD(true)
	d.Config.Healthchecks = nil
	defer func() {
		err := recover()
		if err == nil {
			t.Fail()
		}
		if err.(string) != "No healthchecks, have you run Setup()?" {
			t.Fail()
		}
	}()
	d.HealthCheckOneUpsertRoute("foo", &config.UpsertRoutesSpec{Healthcheck: "foo"})
}

func TestHealthCheckOneUpsertRouteOneShot(t *testing.T) {
	d := getD(true)
	d.oneShot = true
	if !d.HealthCheckOneUpsertRoute("foo", &config.UpsertRoutesSpec{Healthcheck: "foo"}) {
		t.Fail()
	}
}

func TestHealthCheckOneUpsertRoute(t *testing.T) {
	d := getD(true)
	d.Setup()
	if d.HealthCheckOneUpsertRoute("foo", &config.UpsertRoutesSpec{Healthcheck: "public"}) {
		t.Fail()
	}
	d.Config.Healthchecks["public"].PerformHealthcheck() // Run the healthcheck twice to become healthy
	if d.HealthCheckOneUpsertRoute("foo", &config.UpsertRoutesSpec{Healthcheck: "public"}) {
		t.Fail()
	}
	d.Config.Healthchecks["public"].PerformHealthcheck()
	if !d.HealthCheckOneUpsertRoute("foo", &config.UpsertRoutesSpec{Healthcheck: "public"}) {
		t.Fail()
	}
}

func TestRunOneUpsertRouteFailingHealthcheck(t *testing.T) {
	d := getD(true)
	d.Setup()
	if d.HealthCheckOneUpsertRoute("foo", &config.UpsertRoutesSpec{Healthcheck: "public"}) {
		t.Fail()
	}
	if d.RunOneUpsertRoute(&ec2.RouteTable{}, "a", d.Config.RouteTables["a"].UpsertRoutes[0]) != nil {
		t.Fail()
	}
}
func TestHealthCheckOneUpsertRouteUnknown(t *testing.T) {
	d := getD(true)
	d.Setup()
	defer func() {
		err := recover()
		if err == nil {
			t.Fail()
		}
		if err.(string) != "Could not find healthcheck unknown" {
			t.Fail()
		}
	}()
	d.HealthCheckOneUpsertRoute("foo", &config.UpsertRoutesSpec{Healthcheck: "unknown"})
}

func TestHealthCheckOneUpsertRouteNoHealthcheck(t *testing.T) {
	d := getD(true)
	d.oneShot = false
	if !d.HealthCheckOneUpsertRoute("foo", &config.UpsertRoutesSpec{Healthcheck: ""}) {
		t.Fail()
	}
}

func TestRunOneShot(t *testing.T) {
	d := getD(true)
	if d.Run(true, true) != 0 {
		t.Fail()
	}
}

func TestRunOneRouteTableGetFilterFail(t *testing.T) {
	d := getD(true)
	awsRt := make([]*ec2.RouteTable, 0)
	rt := &config.RouteTable{}
	err := d.RunOneRouteTable(awsRt, "public", rt)
	if err == nil {
		t.Fail()
	} else {
		if err.Error() != "Healthcheck type '' not found in the healthcheck registry" {
			t.Log(err)
			t.Fail()
		}
	}
}

func TestRunOneRouteTableNoRouteTablesInAWS(t *testing.T) {
	d := getD(true)
	awsRt := make([]*ec2.RouteTable, 0)
	c := make(map[string]string)
	c["key"] = "Name"
	c["value"] = "private a"
	rt := &config.RouteTable{
		Find: config.RouteTableFindSpec{
			Type:   "by_tag",
			Config: c,
		},
	}
	err := d.RunOneRouteTable(awsRt, "public", rt)
	if err != nil {
		t.Fail()
	}
}

func TestRunOneRouteTable(t *testing.T) {
	d := getD(true)
	awsRt := make([]*ec2.RouteTable, 1)
	awsRt[0] = &ec2.RouteTable{
		Associations: []*ec2.RouteTableAssociation{},
		RouteTableId: aws.String("rtb-9696cffe"),
		Routes:       []*ec2.Route{},
		Tags: []*ec2.Tag{
			&ec2.Tag{
				Key:   aws.String("Name"),
				Value: aws.String("private a"),
			},
		},
	}
	c := make(map[string]string)
	c["key"] = "Name"
	c["value"] = "private a"
	rt := &config.RouteTable{
		Find: config.RouteTableFindSpec{
			Type:   "by_tag",
			Config: c,
		},
	}
	err := d.RunOneRouteTable(awsRt, "public", rt)
	if err != nil {
		t.Log(err)
		t.Fail()
	}
}

/*
func TestHealthCheckOneUpsertRouteHealthcheckFail(t *testing.T) {
	d := getD(true)
	if d.HealthCheckOneUpsertRoute("foo", &config.UpsertRoutesSpec{Healthcheck: "foo"}) {
		t.Fail()
	}
}

func TestHealthCheckOneUpsertRouteHealthcheckSuceed(t *testing.T) {
	d := getD(true)
	if !d.HealthCheckOneUpsertRoute("foo", &config.UpsertRoutesSpec{Healthcheck: "foo"}) {
		t.Fail()
	}
} */
