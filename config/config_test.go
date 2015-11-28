package config

import (
	"errors"
	"fmt"
	a "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/bobtfish/AWSnycast/aws"
	"github.com/bobtfish/AWSnycast/healthcheck"
	"github.com/bobtfish/AWSnycast/instancemetadata"
	"github.com/hashicorp/go-multierror"
	"testing"
)

var tim instancemetadata.InstanceMetadata
var rtm *FakeRouteTableManager

func init() {
	tim = instancemetadata.InstanceMetadata{
		Instance: "i-1234",
	}
	rtm = &FakeRouteTableManager{}
}

func TestLoadConfig(t *testing.T) {
	c, err := New("../tests/awsnycast.yaml", tim, rtm)
	if err != nil {
		t.Log(err)
		t.Fail()
	}
	if c == nil {
		t.Fail()
	}
}

func TestLoadConfigFails(t *testing.T) {
	_, err := New("../tests/doesnotexist.yaml", tim, rtm)
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
	_, err := New("../tests/invalid.yaml", tim, rtm)
	checkOneMultiError(t, err, "Route table a, Validate for 0.0.0.0/0 cannot find healthcheck 'public'")
}

func TestLoadConfigHealthchecks(t *testing.T) {
	c, err := New("../tests/awsnycast.yaml", tim, rtm)
	if err != nil {
		t.Log("Loading config failed")
		t.Log(err)
		t.Fail()
		return
	}
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
					t.Log("HealthcheckName is not localservice")
					t.Fail()
				}
			}
		} else {
			t.Log("CIDR unknown")
			t.Fail()
		}
	}
	b, ok := c.RouteTables["b"]
	if !ok {
		t.Fail()
	}
	if b.Find.Type != "and" {
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
	c.Validate(tim, rtm)
	if c.Healthchecks == nil {
		t.Fail()
	}
	if c.RouteTables["a"].ManageRoutes[0].Cidr != "127.0.0.1/32" {
		t.Fail()
	}
}

func TestConfigValidateNoRouteTables(t *testing.T) {
	c := Config{}
	err := c.Validate(tim, rtm)
	checkOneMultiError(t, err, "No route_tables key in config")
}

func TestConfigValidateEmptyRouteTables(t *testing.T) {
	r := make(map[string]*RouteTable)
	c := Config{
		RouteTables: r,
	}
	err := c.Validate(tim, rtm)
	checkOneMultiError(t, err, "No route_tables defined in config")
}

func TestConfigValidateBadRouteTables(t *testing.T) {
	r := make(map[string]*RouteTable)
	conf := make(map[string]interface{})
	conf["key"] = "foo"
	conf["value"] = "foo"
	r["foo"] = &RouteTable{
		Find: RouteTableFindSpec{
			Type:   "by_tag",
			Config: conf,
		},
	}
	c := Config{
		RouteTables: r,
	}
	err := c.Validate(tim, rtm)
	checkOneMultiError(t, err, "No manage_routes key in route table 'foo'")
}

func TestConfigValidateBadRouteTableUpserts(t *testing.T) {
	r := make(map[string]*RouteTable)
	urs := make([]*aws.ManageRoutesSpec, 1)
	c := make(map[string]interface{})
	c["key"] = "foo"
	c["value"] = "bar"
	urs[0] = &aws.ManageRoutesSpec{}
	r["foo"] = &RouteTable{
		Find: RouteTableFindSpec{
			Type:   "by_tag",
			Config: c,
		},
		ManageRoutes: urs,
	}
	conf := Config{
		RouteTables: r,
	}
	err := conf.Validate(tim, rtm)
	checkOneMultiError(t, err, "cidr is not defined in foo")
}

func TestConfigValidateBadHealthChecks(t *testing.T) {
	c_disk, _ := New("../tests/awsnycast.yaml", tim, rtm)
	c := Config{
		RouteTables:  c_disk.RouteTables,
		Healthchecks: c_disk.Healthchecks,
	}
	c.Healthchecks["foo"] = &healthcheck.Healthcheck{Type: "tcp"}
	c.Healthchecks["foo"].Validate("foo", false)
	err := c.Validate(tim, rtm)
	checkOneMultiError(t, err, "Healthcheck foo has no destination set")
}

func TestConfigValidateNoHealthChecks(t *testing.T) {
	c_disk, _ := New("../tests/awsnycast.yaml", tim, rtm)
	c := Config{
		RouteTables: c_disk.RouteTables,
	}
	err := c.Validate(tim, rtm)
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
	conf := make(map[string]interface{})
	r["a"] = &RouteTable{
		Find:         RouteTableFindSpec{Type: "by_tag", Config: conf},
		ManageRoutes: u,
	}
	c := Config{
		RouteTables: r,
	}
	err := c.Validate(tim, rtm)
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
	err := c.Validate(tim, rtm)
	checkOneMultiError(t, err, "No route_tables key in config")
}

func TestRouteTableFindSpecDefault(t *testing.T) {
	r := RouteTableFindSpec{}
	r.Validate("foo")
	if r.Config == nil {
		t.Fail()
	}
}
func TestRouteTableFindSpecValidate(t *testing.T) {
	c := make(map[string]interface{})
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
	c := make(map[string]interface{})
	c["key"] = "Name"
	c["value"] = "private a"
	r := RouteTableFindSpec{
		Config: c,
	}
	err := r.Validate("foo")
	checkOneMultiError(t, err, "Route find spec foo needs a type key")
}

func TestRouteTableFindSpecValidateUnknownType(t *testing.T) {
	c := make(map[string]interface{})
	c["key"] = "Name"
	c["value"] = "private a"
	r := RouteTableFindSpec{
		Type:   "doesnotexist",
		Config: c,
	}
	err := r.Validate("foo")
	checkOneMultiError(t, err, "Route find spec foo type 'doesnotexist' not known")
}

func TestRouteTableFindSpecValidateNoConfig(t *testing.T) {
	r := RouteTableFindSpec{
		Type: "by_tag",
	}
	err := r.Validate("foo")
	checkOneMultiError(t, err, "Route find spec foo needs config")
}

func TestRouteTableDefaultEmpty(t *testing.T) {
	r := RouteTable{}
	r.Validate("i-1234", rtm, "foo", emptyHealthchecks, emptyHealthchecks)
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
	r.Validate("i-1234", rtm, "foo", emptyHealthchecks, emptyHealthchecks)
	if len(r.ManageRoutes) != 1 {
		t.Fail()
	}
	routeSpec := r.ManageRoutes[0]
	if routeSpec.Cidr != "127.0.0.1/32" {
		t.Fail()
	}
}

func TestRouteTableValidateNoRoutes(t *testing.T) {
	c := make(map[string]interface{})
	c["key"] = "Name"
	c["value"] = "private a"
	rfs := RouteTableFindSpec{
		Type:   "by_tag",
		Config: c,
	}
	r := RouteTable{
		Find:         rfs,
		ManageRoutes: make([]*aws.ManageRoutesSpec, 0),
	}
	err := r.Validate("i-1234", rtm, "foo", emptyHealthchecks, emptyHealthchecks)
	checkOneMultiError(t, err, "No manage_routes key in route table 'foo'")
}

var emptyHealthchecks map[string]*healthcheck.Healthcheck

func init() {
	emptyHealthchecks = make(map[string]*healthcheck.Healthcheck)
}

func TestRouteTableValidate(t *testing.T) {
	routes := make([]*aws.ManageRoutesSpec, 1)
	routes[0] = &aws.ManageRoutesSpec{
		Cidr: "127.0.0.1",
	}
	conf := make(map[string]interface{})
	conf["key"] = "foo"
	conf["value"] = "foo"
	r := &RouteTable{
		Find: RouteTableFindSpec{
			Type:   "by_tag",
			Config: conf,
		},
		ManageRoutes: routes,
	}
	err := r.Validate("i-1234", rtm, "foo", emptyHealthchecks, emptyHealthchecks)
	if err != nil {
		t.Log(err)
		t.Fail()
	}
}

func TestByTagRouteTableFindMissingKey(t *testing.T) {
	c := make(map[string]interface{})
	c["value"] = "foo"
	rts := RouteTableFindSpec{
		Type:   "by_tag",
		Config: c,
	}
	rtf, err := rts.GetFilter()
	if rtf != nil {
		t.Fail()
	}
	checkOneMultiError(t, err, "No key in config for by_tag route table finder")
}

func TestByTagRouteTableFindMissingValue(t *testing.T) {
	c := make(map[string]interface{})
	c["key"] = "Name"
	rts := RouteTableFindSpec{
		Type:   "by_tag",
		Config: c,
	}
	rtf, err := rts.GetFilter()
	if rtf != nil {
		t.Fail()
	}
	checkOneMultiError(t, err, "No value in config for by_tag route table finder")
}

func TestByTagRouteTableFind(t *testing.T) {
	c := make(map[string]interface{})
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
	c := make(map[string]interface{})
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
		if err.Error() != "Route table finder type '' not found in the registry" {
			t.Log(err)
			t.Fail()
		}
	}
}

func TestUpdateEc2RouteTablesNoRouteTablesInAWS(t *testing.T) {
	awsRt := make([]*ec2.RouteTable, 0)
	c := make(map[string]interface{})
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
	rt.Find.NoResultsOk = true
	err = rt.UpdateEc2RouteTables(awsRt)
	if err != nil {
		t.Log(err)
		t.Fail()
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
	c := make(map[string]interface{})
	c["key"] = "Name"
	c["value"] = "private a"
	rt := &RouteTable{
		Find: RouteTableFindSpec{
			Type:   "by_tag",
			Config: c,
		},
		ManageRoutes: []*aws.ManageRoutesSpec{
			&aws.ManageRoutesSpec{
				Cidr: "127.0.0.1/32",
			},
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

func (r *FakeRouteTableManager) InstanceIsRouter(id string) bool {
	return true
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
	err := rt.Validate("i-1234", rtm, "foo", emptyHealthchecks, emptyHealthchecks)
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
	err = rt.RunEc2Updates(frtm, true)
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

func checkOneMultiError(t *testing.T, err error, validate string) {
	if err == nil {
		t.Fail()
	}
	if merr, ok := err.(*multierror.Error); ok {
		if len(merr.Errors) != 1 {
			t.Log(fmt.Sprintf("%v not 1 errors", len(merr.Errors)))
			for i, err := range merr.Errors {
				t.Log(fmt.Sprintf("Error %v is %s", i, err.Error()))
			}
			t.Fail()
			return
		}
		if merr.Errors[0].Error() != validate {
			t.Log("'" + merr.Errors[0].Error() + "' not '" + validate + "'")
			t.Fail()
		}
	} else {
		t.Log("Not multierror")
		t.Log(err)
		t.Fail()
	}
}
func TestRouteTableFindSpecAndNoFilters(t *testing.T) {
	c := make(map[string]interface{})
	_, err := RouteTableFindSpec{Config: c, Type: "and"}.GetFilter()
	checkOneMultiError(t, err, "No filters in config for and route table finder")
}

func TestRouteTableFindSpecOrNoFilters(t *testing.T) {
	c := make(map[string]interface{})
	_, err := RouteTableFindSpec{Config: c, Type: "or"}.GetFilter()
	checkOneMultiError(t, err, "No filters in config for or route table finder")
}

func TestRouteTableFindSpecSubnetNoSubnet(t *testing.T) {
	c := make(map[string]interface{})
	_, err := RouteTableFindSpec{Config: c, Type: "subnet"}.GetFilter()
	if err == nil {
		t.Fail()
	} else {
		if err.Error() != "No subnet_id in config for subnet route table finder" {
			t.Log(err.Error())
			t.Fail()
		}
	}
}

func TestRouteTableFindSpecHasRouteToNoCidr(t *testing.T) {
	c := make(map[string]interface{})
	_, err := RouteTableFindSpec{Config: c, Type: "has_route_to"}.GetFilter()
	if err == nil {
		t.Fail()
	} else {
		if err.Error() != "No cidr in config for has_route_to route table finder" {
			t.Log(err.Error())
			t.Fail()
		}
	}
}

func TestRouteTableFindSpecSubnet(t *testing.T) {
	c := make(map[string]interface{})
	c["subnet_id"] = "subnet-12345"
	_, err := RouteTableFindSpec{Config: c, Type: "subnet"}.GetFilter()
	if err != nil {
		t.Fail()
	}
}

func TestRouteTableFindSpecHasRouteTo(t *testing.T) {
	c := make(map[string]interface{})
	c["cidr"] = "0.0.0.0/0"
	_, err := RouteTableFindSpec{Config: c, Type: "has_route_to"}.GetFilter()
	if err != nil {
		t.Fail()
	}
}

func TestRouteTableFindSpecMain(t *testing.T) {
	c := make(map[string]interface{})
	spec := RouteTableFindSpec{Config: c, Type: "main", Not: true}
	f, err := spec.GetFilter()
	if f == nil {
		t.Fail()
	}
	if err != nil {
		t.Log(err)
		t.Fail()
	}
}

func TestGetFiltersListForSpec(t *testing.T) {
	d := make(map[string]interface{})
	d["key"] = "example tag"
	d["value"] = "foo"
	filterStuff := make([]interface{}, 2)
	filterStuff[0] = RouteTableFindSpec{
		Type:   "by_tag",
		Config: d,
	}
	filterStuff[1] = RouteTableFindSpec{
		Type: "main",
	}
	c := make(map[string]interface{})
	c["filters"] = filterStuff
	spec := RouteTableFindSpec{Config: c}
	filters, err := getFiltersListForSpec(spec)
	if err != nil {
		t.Log(err)
		t.Fail()
	}
	if len(filters) != 2 {
		t.Fail()
	}
}

func TestTableFindSpecAndOr(t *testing.T) {
	d := make(map[string]interface{})
	d["key"] = "example tag"
	d["value"] = "foo"
	filterStuff := make([]interface{}, 2)
	filterStuff[0] = RouteTableFindSpec{
		Type:   "by_tag",
		Config: d,
	}
	filterStuff[1] = RouteTableFindSpec{
		Type: "main",
	}
	c := make(map[string]interface{})
	c["filters"] = filterStuff
	spec := RouteTableFindSpec{Config: c, Type: "and"}
	f, err := spec.GetFilter()
	if err != nil {
		t.Log(err)
		t.Fail()
	}
	if f == nil {
		t.Fail()
	}
	spec2 := RouteTableFindSpec{Config: c, Type: "or"}
	f, err = spec2.GetFilter()
	if err != nil {
		t.Log(err)
		t.Fail()
	}
	if f == nil {
		t.Fail()
	}
}

func TestGetFiltersListForSpecWrongType(t *testing.T) {
	c := make(map[string]interface{})
	c["filters"] = "foo"
	spec := RouteTableFindSpec{Config: c}
	_, err := getFiltersListForSpec(spec)
	checkOneMultiError(t, err, "unexpected type string for 'filters' key")
}
func TestGetFiltersListForSpecInnerFails(t *testing.T) {
	d := make(map[string]interface{})
	filterStuff := make([]interface{}, 1)
	filterStuff[0] = RouteTableFindSpec{
		Type:   "by_tag",
		Config: d,
	}
	c := make(map[string]interface{})
	c["filters"] = filterStuff
	spec := RouteTableFindSpec{Config: c, Type: "or"}
	_, err := spec.GetFilter()
	if err == nil {
		t.Fail()
	} else {
		if merr, ok := err.(*multierror.Error); ok {
			if len(merr.Errors) != 2 {
				t.Log(fmt.Printf("%d not 2 errors", len(merr.Errors)))
				t.Fail()
			}
			if merr.Errors[1].Error() != "No value in config for by_tag route table finder for or route table finder" {
				t.Log(merr.Errors[1])
				t.Fail()
			}
		} else {
			t.Log("Not multierror")
			t.Log(err)
			t.Fail()
		}
	}
}
