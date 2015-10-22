package config

import (
	"fmt"
	"github.com/bobtfish/AWSnycast/healthcheck"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	c, err := New("../tests/awsnycast.yaml")
	if err != nil {
		t.Log(err)
		t.Fail()
	}
	if c == nil {
		t.Fail()
	}
}

func TestLoadConfigFails(t *testing.T) {
	_, err := New("../tests/doesnotexist.yaml")
	if err == nil {
		t.Fail()
	}
}

func TestLoadConfigHealthchecks(t *testing.T) {
	c, _ := New("../tests/awsnycast.yaml")
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
			if route.Instance != "SELF" {
				t.Log("route.Instance not SELF")
				t.Fail()
			}
			if route.Cidr == "0.0.0.0/0" {
				if route.Healthcheck != "public" {
					t.Log("Healthcheck not public")
					t.Fail()
				}
			} else {
				if route.Healthcheck != "localservice" {
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
		ManageRoutes: []*ManageRoutesSpec{&ManageRoutesSpec{Cidr: "127.0.0.1"}},
	}
	c := Config{
		RouteTables: r,
	}
	c.Default()
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
	urs := make([]*ManageRoutesSpec, 1)
	urs[0] = &ManageRoutesSpec{}
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
	c_disk, _ := New("../tests/awsnycast.yaml")
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
	c_disk, _ := New("../tests/awsnycast.yaml")
	c := Config{
		RouteTables: c_disk.RouteTables,
	}
	c.Default()
	err := c.Validate()
	if err == nil {
		t.Fail()
	}
}

func TestConfigValidate(t *testing.T) {
	u := make([]*ManageRoutesSpec, 1)
	u[0] = &ManageRoutesSpec{
		Cidr: "127.0.0.1",
	}
	r := make(map[string]*RouteTable)
	r["a"] = &RouteTable{
		ManageRoutes: u,
	}
	c := Config{
		RouteTables: r,
	}
	c.Default()
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
	c.Default()
	err := c.Validate()
	if err == nil {
		t.Fail()
	}
	if err.Error() != "No route_tables defined in config" {
		t.Log(err.Error())
		t.Fail()
	}
}

// FIXME - need tests for each part of config failing, and check errors.

func TestManageRoutesSpecDefault(t *testing.T) {
	u := &ManageRoutesSpec{
		Cidr: "127.0.0.1",
	}
	u.Default()
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
	r.Default()
}

func TestRouteTableDefault(t *testing.T) {
	routes := make([]*ManageRoutesSpec, 1)
	routes[0] = &ManageRoutesSpec{
		Cidr: "127.0.0.1",
	}
	r := RouteTable{
		ManageRoutes: routes,
	}
	r.Default()
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
		ManageRoutes: make([]*ManageRoutesSpec, 0),
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
	routes := make([]*ManageRoutesSpec, 1)
	routes[0] = &ManageRoutesSpec{
		Cidr: "127.0.0.1",
	}
	r := RouteTable{
		ManageRoutes: routes,
	}
	r.Default()
	h := make(map[string]*healthcheck.Healthcheck)
	err := r.Validate("foo", h)
	if err != nil {
		t.Fail()
	}
}

func TestUpsertRouteSpecGetInstanceSELF(t *testing.T) {
	urs := ManageRoutesSpec{
		Cidr:     "127.0.0.1",
		Instance: "SELF",
	}
	if urs.GetInstance("i-other") != "i-other" {
		t.Fail()
	}
}

func TestUpsertRouteSpecGetInstanceOther(t *testing.T) {
	urs := ManageRoutesSpec{
		Cidr:     "127.0.0.1",
		Instance: "i-foo",
	}
	if urs.GetInstance("i-other") != "i-foo" {
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
