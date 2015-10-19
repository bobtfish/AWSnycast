package daemon

import (
	"errors"
	"fmt"
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
	return v, errors.New(fmt.Sprintf("Key %s unknown"))
}

type FakeRouteTableFetcher struct{}

func (r FakeRouteTableFetcher) GetRouteTables() ([]*ec2.RouteTable, error) {
	return []*ec2.RouteTable{}, nil
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
	fakeR := FakeRouteTableFetcher{}
	fakeM.Meta = make(map[string]string)
	fakeM.Meta["placement/availability-zone"] = "us-west-1a"
	fakeM.Meta["instance-id"] = "i-1234"
	fakeM.Meta["mac"] = "06:1d:ea:6f:8c:6e"
	fakeM.Meta["network/interfaces/macs/06:1d:ea:6f:8c:6e/subnet-id"] = "subnet-28b0e940"
	d.MetadataFetcher = fakeM
	d.RouteTableFetcher = fakeR
	return d
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

func TestHealthCheckOneUpsertRouteOneShot(t *testing.T) {
	d := getD(true)
	d.oneShot = true
	if !d.HealthCheckOneUpsertRoute("foo", &config.UpsertRoutesSpec{Healthcheck: "foo"}) {
		t.Fail()
	}
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
