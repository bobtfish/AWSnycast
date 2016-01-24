package daemon

import (
	"errors"
	"fmt"
	a "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/bobtfish/AWSnycast/aws"
	"github.com/bobtfish/AWSnycast/config"
	"github.com/bobtfish/AWSnycast/healthcheck"
	"github.com/bobtfish/AWSnycast/instancemetadata"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
	"time"
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

func NewFakeRouteTableManager() *FakeRouteTableManager {
	f := &FakeRouteTableManager{}
	f.Tables = make([]*ec2.RouteTable, 0)
	return f
}

func (r *FakeRouteTableManager) InstanceIsRouter(id string) bool {
	return true
}

type FakeRouteTableManager struct {
	Tables                   []*ec2.RouteTable
	Error                    error
	RouteTable               ec2.RouteTable
	Cidr                     string
	Instance                 string
	IfUnhealthy              bool
	Noop                     bool
	ManageInstanceRouteError error
}

func (f *FakeRouteTableManager) GetRouteTables() ([]*ec2.RouteTable, error) {
	return f.Tables, f.Error
}

func (f *FakeRouteTableManager) ManageInstanceRoute(rtb ec2.RouteTable, rs aws.ManageRoutesSpec, noop bool) error {
	f.RouteTable = rtb
	f.Cidr = rs.Cidr
	f.Instance = rs.Instance
	f.IfUnhealthy = rs.IfUnhealthy
	f.Noop = noop
	return f.ManageInstanceRouteError
}

func getFakeMetadataFetcher(a bool) aws.MetadataFetcher {
	fakeM := FakeMetadataFetcher{
		FAvailable: a,
	}
	fakeM.Meta = make(map[string]string)
	fakeM.Meta["placement/availability-zone"] = "us-west-1a"
	fakeM.Meta["instance-id"] = "i-1234"
	fakeM.Meta["mac"] = "06:1d:ea:6f:8c:6e"
	fakeM.Meta["local-ipv4"] = "127.0.0.1"
	fakeM.Meta["network/interfaces/macs/06:1d:ea:6f:8c:6e/subnet-id"] = "subnet-28b0e940"
	return fakeM
}

func getD(a bool) Daemon {
	d := Daemon{
		ConfigFile: "../tests/awsnycast.yaml",
		Config:     &config.Config{},
	}
	d.Config.Validate(instancemetadata.InstanceMetadata{Instance: "i-1234"}, NewFakeRouteTableManager()) // FIXME error handling
	fakeR := NewFakeRouteTableManager()
	d.MetadataFetcher = getFakeMetadataFetcher(a)
	d.RouteTableManager = fakeR
	return d
}

func TestRunRouteTablesFailGetRouteTables(t *testing.T) {
	assert := assert.New(t)
	d := getD(true)
	rtf := d.RouteTableManager.(*FakeRouteTableManager)
	rtf.Error = errors.New("Route table get fail")
	err := d.RunRouteTables()
	if assert.NotNil(err) {
		assert.Equal(err.Error(), "Route table get fail")
	}
}

func TestSetupNoMetadataService(t *testing.T) {
	assert := assert.New(t)
	fakeM := FakeMetadataFetcher{
		FAvailable: false,
	}
	d := Daemon{
		ConfigFile: "../tests/awsnycast.yaml",
	}
	assert.NotNil(t, d.RouteTableManager)
	d.MetadataFetcher = fakeM

	err := d.Setup()

	if assert.NotNil(err) {
		assert.Equal(err.Error(), "No metadata service")
	}
}

func TestSetupNormalMetadataService(t *testing.T) {
	assert := assert.New(t)
	d := Daemon{
		ConfigFile: "../tests/awsnycast.yaml",
	}
	d.setupMetadataFetcher()
	assert.NotNil(d.MetadataFetcher)
}

func myHealthCheckConstructorFail(h healthcheck.Healthcheck) (healthcheck.HealthChecker, error) {
	return nil, errors.New("Test")
}

func TestConfigBadHealthcheck(t *testing.T) {
	healthcheck.RegisterHealthcheck("testconstructorfail", myHealthCheckConstructorFail)
	c := &config.Config{}
	c.Validate(instancemetadata.InstanceMetadata{Instance: "i-1234"}, NewFakeRouteTableManager())
	c.Healthchecks["one"] = &healthcheck.Healthcheck{
		Type:        "testconstructorfail",
		Destination: "127.0.0.1",
	}
	err := setupHealthchecks(c, false)
	if assert.NotNil(t, err) {
		assert.Equal(t, err.Error(), "Test")
	}
}

func TestSetupNormal(t *testing.T) {
	assert.Nil(t, os.Setenv("AWS_ACCESS_KEY_ID", "AKIAJRYDH3TP2D3WKRNQ"))
	assert.Nil(t, os.Setenv("AWS_SECRET_ACCESS_KEY", "8Dbur5oqKACVDzpE/CA6g+XXAmyxmYEShVG7w4XF"))
	fakeM := FakeMetadataFetcher{
		FAvailable: true,
	}
	fakeM.Meta = make(map[string]string)
	fakeM.Meta["placement/availability-zone"] = "us-west-1a"
	fakeM.Meta["instance-id"] = "i-1234"
	fakeM.Meta["mac"] = "06:1d:ea:6f:8c:6e"
	fakeM.Meta["local-ipv4"] = "127.0.0.1"
	fakeM.Meta["network/interfaces/macs/06:1d:ea:6f:8c:6e/subnet-id"] = "subnet-28b0e940"
	d := Daemon{
		ConfigFile: "../tests/awsnycast.yaml",
	}
	assert.Nil(t, d.RouteTableManager)
	d.MetadataFetcher = fakeM
	err := d.Setup()
	assert.Nil(t, err)
}

func TestSetupBadConfigFile(t *testing.T) {
	d := getD(true)
	d.ConfigFile = "../tests/doesnotexist.yaml"
	err := d.Setup()
	if assert.NotNil(t, err) {
		assert.Equal(t, err.Error(), "open ../tests/doesnotexist.yaml: no such file or directory")
	}
	assert.Equal(t, d.Run(true, false, false), 1)
}

func TestSetupHealthChecks(t *testing.T) {
	d := getD(true)
	d.Debug = true
	err := d.Setup()
	assert.Nil(t, err, "Setup failed")
	assert.Equal(t, d.Config.Healthchecks["public"].IsRunning(), false, "HealthChecks already running")
	d.runHealthChecks()
	assert.Equal(t, d.Config.Healthchecks["public"].IsRunning(), true, "HealthChecks did not start running")
	d.stopHealthChecks()
	assert.Equal(t, d.Config.Healthchecks["public"].IsRunning(), false, "HealthChecks still running")
}

func TestRunOneShotFail(t *testing.T) {
	d := getD(true)
	assert.Equal(t, d.Run(true, true, false), 1)
}

func TestRunOneShot(t *testing.T) {
	d := getD(true)
	awsRt := make([]*ec2.RouteTable, 2)
	awsRt[0] = &ec2.RouteTable{
		Associations: []*ec2.RouteTableAssociation{},
		RouteTableId: a.String("rtb-9696cffe"),
		Routes:       []*ec2.Route{},
		Tags: []*ec2.Tag{
			&ec2.Tag{
				Key:   a.String("Name"),
				Value: a.String("private a"),
			},
		},
	}
	awsRt[1] = &ec2.RouteTable{
		Associations: []*ec2.RouteTableAssociation{},
		RouteTableId: a.String("rtb-deadbeef"),
		Routes:       []*ec2.Route{},
		Tags: []*ec2.Tag{
			&ec2.Tag{
				Key:   a.String("type"),
				Value: a.String("private"),
			},
			&ec2.Tag{
				Key:   a.String("az"),
				Value: a.String("eu-west-1b"),
			},
		},
	}
	d.RouteTableManager.(*FakeRouteTableManager).Tables = awsRt
	assert.Equal(t, d.Run(true, true, false), 0)
}

func TestRunOneRouteTableGetFilterFail(t *testing.T) {
	d := getD(true)
	awsRt := make([]*ec2.RouteTable, 0)
	rt := &config.RouteTable{}
	err := d.RunOneRouteTable(awsRt, "public", rt)
	if assert.NotNil(t, err) {
		assert.Equal(t, err.Error(), "Route table finder type '' not found in the registry")
	}
}

func TestRunOneRouteTableNoRouteTablesInAWS(t *testing.T) {
	d := getD(true)
	awsRt := make([]*ec2.RouteTable, 0)
	c := make(map[string]interface{})
	c["key"] = "Name"
	c["value"] = "private a"
	rt := &config.RouteTable{
		Find: config.RouteTableFindSpec{
			Type:   "by_tag",
			Config: c,
		},
	}
	err := d.RunOneRouteTable(awsRt, "public", rt)
	if assert.NotNil(t, err) {
		assert.Equal(t, err.Error(), "No route table in AWS matched filter spec")
	}
}

func TestRunOneRouteTableNoManageRoutes(t *testing.T) {
	d := getD(true)
	awsRt := make([]*ec2.RouteTable, 1)
	awsRt[0] = &ec2.RouteTable{
		Associations: []*ec2.RouteTableAssociation{},
		RouteTableId: a.String("rtb-9696cffe"),
		Routes:       []*ec2.Route{},
		Tags: []*ec2.Tag{
			&ec2.Tag{
				Key:   a.String("Name"),
				Value: a.String("private a"),
			},
		},
	}
	c := make(map[string]interface{})
	c["key"] = "Name"
	c["value"] = "private a"
	rt := &config.RouteTable{
		Find: config.RouteTableFindSpec{
			Type:   "by_tag",
			Config: c,
		},
	}
	err := d.RunOneRouteTable(awsRt, "public", rt)
	assert.Nil(t, err)
}

func TestRunOneRouteTable(t *testing.T) {
	d := getD(true)
	awsRt := make([]*ec2.RouteTable, 1)
	awsRt[0] = &ec2.RouteTable{
		Associations: []*ec2.RouteTableAssociation{},
		RouteTableId: a.String("rtb-9696cffe"),
		Routes:       []*ec2.Route{},
		Tags: []*ec2.Tag{
			&ec2.Tag{
				Key:   a.String("Name"),
				Value: a.String("private a"),
			},
		},
	}
	c := make(map[string]interface{})
	c["key"] = "Name"
	c["value"] = "private a"
	u := make([]*aws.ManageRoutesSpec, 1)
	u[0] = &aws.ManageRoutesSpec{
		Cidr:     "0.0.0.0/0",
		Instance: "i-12345",
	}
	rt := &config.RouteTable{
		Find: config.RouteTableFindSpec{
			Type:   "by_tag",
			Config: c,
		},
		ManageRoutes: u,
	}
	assert.Nil(t, d.RunOneRouteTable(awsRt, "public", rt))
}

func TestRunOneRouteTableUpsertRouteFail(t *testing.T) {
	d := getD(true)
	rtf := d.RouteTableManager.(*FakeRouteTableManager)
	rtf.ManageInstanceRouteError = errors.New("Test")
	awsRt := make([]*ec2.RouteTable, 1)
	awsRt[0] = &ec2.RouteTable{
		Associations: []*ec2.RouteTableAssociation{},
		RouteTableId: a.String("rtb-9696cffe"),
		Routes:       []*ec2.Route{},
		Tags: []*ec2.Tag{
			&ec2.Tag{
				Key:   a.String("Name"),
				Value: a.String("private a"),
			},
		},
	}
	c := make(map[string]interface{})
	c["key"] = "Name"
	c["value"] = "private a"
	u := make([]*aws.ManageRoutesSpec, 1)
	u[0] = &aws.ManageRoutesSpec{
		Cidr:     "0.0.0.0/0",
		Instance: "i-12345",
	}
	rt := &config.RouteTable{
		Find: config.RouteTableFindSpec{
			Type:   "by_tag",
			Config: c,
		},
		ManageRoutes: u,
	}
	err := d.RunOneRouteTable(awsRt, "public", rt)
	if assert.NotNil(t, err) {
		assert.Equal(t, err.Error(), "Test")
	}
}

func TestRunSleepLoop(t *testing.T) {
	d := getD(true)
	assert.Nil(t, d.Setup())
	d.FetchWait = time.Nanosecond
	d.loopQuitChan = make(chan bool, 10)
	d.loopTimerChan = make(chan bool, 10)
	d.RunSleepLoop()
	time.Sleep(time.Millisecond)
	d.loopQuitChan <- true
	time.Sleep(time.Millisecond)
}

func TestRunOneReal(t *testing.T) {
	d := getD(true)
	d.FetchWait = time.Nanosecond
	awsRt := make([]*ec2.RouteTable, 2)
	awsRt[0] = &ec2.RouteTable{
		Associations: []*ec2.RouteTableAssociation{},
		RouteTableId: a.String("rtb-9696cffe"),
		Routes:       []*ec2.Route{},
		Tags: []*ec2.Tag{
			&ec2.Tag{
				Key:   a.String("Name"),
				Value: a.String("private a"),
			},
		},
	}
	awsRt[1] = &ec2.RouteTable{
		Associations: []*ec2.RouteTableAssociation{},
		RouteTableId: a.String("rtb-deadbeef"),
		Routes:       []*ec2.Route{},
		Tags: []*ec2.Tag{
			&ec2.Tag{
				Key:   a.String("az"),
				Value: a.String("eu-west-1b"),
			},
			&ec2.Tag{
				Key:   a.String("type"),
				Value: a.String("private"),
			},
		},
	}
	d.RouteTableManager.(*FakeRouteTableManager).Tables = awsRt
	hasFinishedRunLoop := make(chan bool, 1)
	go func() {
		assert.Equal(t, d.Run(false, true, false), 0, "Run was not successful")
		hasFinishedRunLoop <- true
	}()
	time.Sleep(time.Millisecond)
	d.quitChan <- true
	finished := <-hasFinishedRunLoop
	assert.Equal(t, finished, true)
}
