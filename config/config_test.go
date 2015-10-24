package config

import (
	"errors"
	"fmt"
	a "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/bobtfish/AWSnycast/aws"
	"github.com/bobtfish/AWSnycast/healthcheck"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	c, err := New("../tests/awsnycast.yaml", "i-1234")
	if err != nil {
		t.Log(err)
		t.Fail()
	}
	if c == nil {
		t.Fail()
	}
}

func TestLoadConfigFails(t *testing.T) {
	_, err := New("../tests/doesnotexist.yaml", "i-1234")
	if err == nil {
		t.Fail()
	} else {
		if err.Error() != "open ../tests/doesnotexist.yaml: no such file or directory" {
			t.Log(err)
			t.Fail()
		}
	}
}

func TestLoadConfigFailsValidation(t *testing.T) {
	_, err := New("../tests/invalid.yaml", "i-1234")
	if err == nil {
		t.Fail()
	} else {
		if err.Error() != "Route table a, upsert 0.0.0.0/0 cannot find healthcheck 'public'" {
			t.Log(err)
			t.Fail()
		}
	}
}

func TestLoadConfigHealthchecks(t *testing.T) {
	c, _ := New("../tests/awsnycast.yaml", "i-1234")
	if c.Healthchecks == nil {
		t.Log("c.Healthchecks == nil")
		t.Fail()
	}
	h, ok := c.Healthchecks["public"]
	if !ok {
		t.Log("c.Healthchecks['public'] not ok")
		t.Fail()
	}
	if h.Type != "ping" {
		t.Log("type not ping")
		t.Fail()
	}
	if h.Destination != "8.8.8.8" {
		t.Log("Destination not 8.8.8.8")
		t.Fail()
	}
	if h.Rise != 2 {
		t.Log("Rise not 2")
		t.Fail()
	}
	if h.Fall != 10 {
		t.Log("fall not 10")
		t.Fail()
	}
	if h.Every != 1 {
		t.Log("every not 1")
		t.Fail()
	}
	a, ok := c.RouteTables["a"]
	if !ok {
		t.Log("RouteTables a not ok")
		t.Fail()
	}
	if a.Find.Type != "by_tag" {
		t.Log("Not by_tag")
		t.Fail()
	}
	if v, ok := a.Find.Config["key"]; ok {
		if v != "Name" {
			t.Log("Config key Name not found")
			t.Fail()
		}
	} else {
		t.Log(fmt.Sprintf("Config key not found: %+v", a.Find.Config))
		t.Fail()
	}
	if v, ok := a.Find.Config["value"]; ok {
		if v != "private a" {
			t.Log("Config value not private a")
			t.Fail()
		}
	} else {
		t.Log("Config value not present")
		t.Fail()
	}
	routes := a.ManageRoutes
	if len(routes) != 2 {
		t.Log("Route len not 2")
		t.Fail()
	}
	for _, route := range routes {
		if route.Cidr == "0.0.0.0/0" || route.Cidr == "192.168.1.1/32" {
			if route.Instance != "i-1234" {
				t.Log("route.Instance not SELF")
				t.Fail()
			}
			if route.Cidr == "0.0.0.0/0" {
				if route.HealthcheckName != "public" {
					t.Log("Healthcheck not public")
					t.Fail()
				}
			} else {
				if route.HealthcheckName != "localservice" {
					t.Fail()
				}
			}
		} else {
			t.Fail()
		}
	}
	b, ok := c.RouteTables["b"]
	if !ok {
		t.Fail()
	}
	if b.Find.Type != "by_tag" {
		t.Fail()
	}
}

func TestConfigDefault(t *testing.T) {
	r := make(map[string]*RouteTable)
	r["a"] = &RouteTable{
		ManageRoutes: []*aws.ManageRoutesSpec{&aws.ManageRoutesSpec{Cidr: "127.0.0.1"}},
	}
	c := Config{
		RouteTables: r,
	}
	c.Default("i-1234")
	if c.Healthchecks == nil {
		t.Fail()
	}
	if c.RouteTables["a"].ManageRoutes[0].Cidr != "127.0.0.1/32" {
		t.Fail()
	}
}

func TestConfigValidateNoRouteTables(t *testing.T) {
	c := Config{}
	err := c.Validate()
	if err == nil {
		t.Fail()
	}
	if err.Error() != "No route_tables key in config" {
		t.Log(err.Error())
		t.Fail()
	}
}

func TestConfigValidateEmptyRouteTables(t *testing.T) {
	r := make(map[string]*RouteTable)
	c := Config{
		RouteTables: r,
	}
	err := c.Validate()
	if err == nil {
		t.Fail()
	}
	if err.Error() != "No route_tables defined in config" {
		t.Log(err.Error())
		t.Fail()
	}
}

func TestConfigValidateBadRouteTables(t *testing.T) {
	r := make(map[string]*RouteTable)
	r["foo"] = &RouteTable{}
	c := Config{
		RouteTables: r,
	}
	err := c.Validate()
	if err == nil {
		t.Fail()
	}
	if err.Error() != "No manage_routes key in route table 'foo'" {
		t.Log(err.Error())
		t.Fail()
	}
}

func TestConfigValidateBadRouteTableUpserts(t *testing.T) {
	r := make(map[string]*RouteTable)
	urs := make([]*aws.ManageRoutesSpec, 1)
	urs[0] = &aws.ManageRoutesSpec{}
	r["foo"] = &RouteTable{
		ManageRoutes: urs,
	}
	c := Config{
		RouteTables: r,
	}
	err := c.Validate()
	if err == nil {
		t.Fail()
	}
	if err.Error() != "cidr is not defined in foo" {
		t.Log(err.Error())
		t.Fail()
	}
}

func TestConfigValidateBadHealthChecks(t *testing.T) {
	c_disk, _ := New("../tests/awsnycast.yaml", "i-1234")
	h := make(map[string]*healthcheck.Healthcheck)
	h["foo"] = &healthcheck.Healthcheck{}
	c := Config{
		RouteTables:  c_disk.RouteTables,
		Healthchecks: h,
	}
	err := c.Validate()
	if err == nil {
		t.Fail()
	}
	if err.Error() != "Healthcheck foo has no destination set" {
		t.Log(err.Error())
		t.Fail()
	}
}

func TestConfigValidateNoHealthChecks(t *testing.T) {
	c_disk, _ := New("../tests/awsnycast.yaml", "i-124")
	c := Config{
		RouteTables: c_disk.RouteTables,
	}
	c.Default("i-1234")
	err := c.Validate()
	if err == nil {
		t.Fail()
	}
}

func TestConfigValidate(t *testing.T) {
	u := make([]*aws.ManageRoutesSpec, 1)
	u[0] = &aws.ManageRoutesSpec{
		Cidr: "127.0.0.1",
	}
	r := make(map[string]*RouteTable)
	r["a"] = &RouteTable{
		ManageRoutes: u,
	}
	c := Config{
		RouteTables: r,
	}
	c.Default("i-1234")
	err := c.Validate()
	if err != nil {
		t.Log(err)
		t.Fail()
	}
	rt := c.RouteTables["a"]
	ur := rt.ManageRoutes[0]
	if ur.Cidr != "127.0.0.1/32" {
		t.Fail()
	}
}

func TestConfigValidateEmpty(t *testing.T) {
	c := Config{}
	c.Default("i-1234")
	err := c.Validate()
	if err == nil {
		t.Fail()
	}
	if err.Error() != "No route_tables defined in config" {
		t.Log(err.Error())
		t.Fail()
	}
}

func TestRouteTableFindSpecDefault(t *testing.T) {
	r := RouteTableFindSpec{}
	r.Default()
	if r.Config == nil {
		t.Fail()
	}
}
func TestRouteTableFindSpecValidate(t *testing.T) {
	c := make(map[string]string)
	c["key"] = "Name"
	c["value"] = "private a"
	r := RouteTableFindSpec{
		Type:   "by_tag",
		Config: c,
	}
	err := r.Validate("foo")
	if err != nil {
		t.Log(err)
		t.Fail()
	}
}

func TestRouteTableFindSpecValidateNoType(t *testing.T) {
	c := make(map[string]string)
	c["key"] = "Name"
	c["value"] = "private a"
	r := RouteTableFindSpec{
		Config: c,
	}
	err := r.Validate("foo")
	if err == nil {
		t.Fail()
	}
	if err.Error() != "Route find spec foo needs a type key" {
		t.Log(err.Error())
		t.Fail()
	}
}

func TestRouteTableFindSpecValidateUnknownType(t *testing.T) {
	c := make(map[string]string)
	c["key"] = "Name"
	c["value"] = "private a"
	r := RouteTableFindSpec{
		Type:   "doesnotexist",
		Config: c,
	}
	err := r.Validate("foo")
	if err == nil {
		t.Fail()
	}
	if err.Error() != "Route find spec foo type 'doesnotexist' not known" {
		t.Log(err.Error())
		t.Fail()
	}
}

func TestRouteTableFindSpecValidateNoConfig(t *testing.T) {
	r := RouteTableFindSpec{
		Type: "by_tag",
	}
	err := r.Validate("foo")
	if err == nil {
		t.Fail()
	}
	if err.Error() != "No config supplied" {
		t.Log(err.Error())
		t.Fail()
	}
}

func TestRouteTableDefaultEmpty(t *testing.T) {
	r := RouteTable{}
	r.Default("i-1234")
	if r.ManageRoutes == nil {
		t.Fail()
	}
	if r.ec2RouteTables == nil {
		t.Fail()
	}
}

func TestRouteTableDefault(t *testing.T) {
	routes := make([]*aws.ManageRoutesSpec, 1)
	routes[0] = &aws.ManageRoutesSpec{
		Cidr: "127.0.0.1",
	}
	r := RouteTable{
		ManageRoutes: routes,
	}
	r.Default("i-1234")
	if len(r.ManageRoutes) != 1 {
		t.Fail()
	}
	routeSpec := r.ManageRoutes[0]
	if routeSpec.Cidr != "127.0.0.1/32" {
		t.Fail()
	}
}

func TestRouteTableValidateNullRoutes(t *testing.T) {
	r := RouteTable{}
	h := make(map[string]*healthcheck.Healthcheck)
	err := r.Validate("foo", h)
	if err == nil {
		t.Fail()
	}
	if err.Error() != "No manage_routes key in route table 'foo'" {
		t.Log(err.Error())
		t.Fail()
	}
}

func TestRouteTableValidateNoRoutes(t *testing.T) {
	r := RouteTable{
		ManageRoutes: make([]*aws.ManageRoutesSpec, 0),
	}
	h := make(map[string]*healthcheck.Healthcheck)
	err := r.Validate("foo", h)
	if err == nil {
		t.Fail()
	}
	if err.Error() != "No manage_routes key in route table 'foo'" {
		t.Log(err.Error())
		t.Fail()
	}
}

func TestRouteTableValidate(t *testing.T) {
	routes := make([]*aws.ManageRoutesSpec, 1)
	routes[0] = &aws.ManageRoutesSpec{
		Cidr: "127.0.0.1",
	}
	r := RouteTable{
		ManageRoutes: routes,
	}
	r.Default("i-1234")
	h := make(map[string]*healthcheck.Healthcheck)
	err := r.Validate("foo", h)
	if err != nil {
		t.Fail()
	}
}

func TestByTagRouteTableFindMissingKey(t *testing.T) {
	c := make(map[string]string)
	rts := RouteTableFindSpec{
		Type:   "by_tag",
		Config: c,
	}
	rtf, err := rts.GetFilter()
	if rtf != nil {
		t.Fail()
	}
	if err == nil {
		t.Fail()
	}
	if err.Error() != "No key in config for by_tag route table finder" {
		t.Fail()
	}
}

func TestByTagRouteTableFindMissingValue(t *testing.T) {
	c := make(map[string]string)
	c["key"] = "Name"
	rts := RouteTableFindSpec{
		Type:   "by_tag",
		Config: c,
	}
	rtf, err := rts.GetFilter()
	if rtf != nil {
		t.Fail()
	}
	if err == nil {
		t.Fail()
	}
	if err.Error() != "No value in config for by_tag route table finder" {
		t.Fail()
	}
}

func TestByTagRouteTableFind(t *testing.T) {
	c := make(map[string]string)
	c["key"] = "Name"
	c["value"] = "private b"
	rts := RouteTableFindSpec{
		Type:   "by_tag",
		Config: c,
	}
	rtf, err := rts.GetFilter()
	if rtf == nil {
		t.Fail()
	}
	if err != nil {
		t.Fail()
	}
}

func TestRouteTableFindUnknownType(t *testing.T) {
	c := make(map[string]string)
	rts := RouteTableFindSpec{
		Type:   "unknown",
		Config: c,
	}
	rtf, err := rts.GetFilter()
	if rtf != nil {
		t.Fail()
	}
	if err == nil {
		t.Fail()
	}
}

func TestUpdateEc2RouteTablesRouteTablesGetFilterFail(t *testing.T) {
	awsRt := make([]*ec2.RouteTable, 0)
	rt := &RouteTable{}
	err := rt.UpdateEc2RouteTables(awsRt)
	if err == nil {
		t.Fail()
	} else {
		if err.Error() != "Healthcheck type '' not found in the healthcheck registry" {
			t.Log(err)
			t.Fail()
		}
	}
}

func TestUpdateEc2RouteTablesNoRouteTablesInAWS(t *testing.T) {
	awsRt := make([]*ec2.RouteTable, 0)
	c := make(map[string]string)
	c["key"] = "Name"
	c["value"] = "private a"
	rt := &RouteTable{
		Find: RouteTableFindSpec{
			Type:   "by_tag",
			Config: c,
		},
	}
	err := rt.UpdateEc2RouteTables(awsRt)
	if err == nil {
		t.Fail()
	} else {
		if err.Error() != "No route table in AWS matched filter spec" {
			t.Log(err)
			t.Fail()
		}
	}
}

func TestUpdateEc2RouteTablesFindRouteTablesInAWS(t *testing.T) {
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
	c := make(map[string]string)
	c["key"] = "Name"
	c["value"] = "private a"
	rt := &RouteTable{
		Find: RouteTableFindSpec{
			Type:   "by_tag",
			Config: c,
		},
	}
	err := rt.UpdateEc2RouteTables(awsRt)
	if err != nil {
		t.Log(err)
		t.Fail()
	}
}

type FakeRouteTableManager struct {
	Error            error
	RouteTable       ec2.RouteTable
	ManageRoutesSpec aws.ManageRoutesSpec
	Noop             bool
}

func (r *FakeRouteTableManager) GetRouteTables() ([]*ec2.RouteTable, error) {
	return nil, nil
}

func (r *FakeRouteTableManager) ManageInstanceRoute(rtb ec2.RouteTable, rs aws.ManageRoutesSpec, noop bool) error {
	r.RouteTable = rtb
	r.ManageRoutesSpec = rs
	r.Noop = noop
	return r.Error
}

func TestRunEc2Updates(t *testing.T) {
	rt := &RouteTable{
		ManageRoutes: []*aws.ManageRoutesSpec{&aws.ManageRoutesSpec{Cidr: "127.0.0.1"}},
	}
	rt.Default("i-1234")
	rt.ec2RouteTables = append(rt.ec2RouteTables, &ec2.RouteTable{
		Associations: []*ec2.RouteTableAssociation{},
		RouteTableId: a.String("rtb-9696cffe"),
		Routes:       []*ec2.Route{},
		Tags: []*ec2.Tag{
			&ec2.Tag{
				Key:   a.String("Name"),
				Value: a.String("private a"),
			},
		},
	})
	frtm := &FakeRouteTableManager{}
	err := rt.RunEc2Updates(frtm, true)
	if err != nil {
		t.Log(err)
		t.Fail()
	}
	if *(frtm.RouteTable.RouteTableId) != "rtb-9696cffe" {
		t.Log(*(frtm.RouteTable.RouteTableId))
		t.Fail()
	}
	if frtm.ManageRoutesSpec.Cidr != "127.0.0.1/32" {
		t.Fail()
	}
	frtm.Error = errors.New("Test error")
	err = rt.RunEc2Updates(frtm, true)
	if err == nil {
		t.Fail()
	} else {
		if err.Error() != "Test error" {
			t.Log(err)
			t.Fail()
		}
	}
}
