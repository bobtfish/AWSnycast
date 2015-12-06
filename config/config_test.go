package config

import (
	"errors"
	"fmt"
	a "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/bobtfish/AWSnycast/aws"
	"github.com/bobtfish/AWSnycast/healthcheck"
	"github.com/bobtfish/AWSnycast/instancemetadata"
	"github.com/bobtfish/AWSnycast/testhelpers"
	"github.com/hashicorp/go-multierror"
	"github.com/stretchr/testify/assert"
	"testing"
)

var tim instancemetadata.InstanceMetadata
var rtm *FakeRouteTableManager

var emptyHealthchecks map[string]*healthcheck.Healthcheck

func init() {
	emptyHealthchecks = make(map[string]*healthcheck.Healthcheck)
	tim = instancemetadata.InstanceMetadata{
		Instance: "i-1234",
	}
	rtm = &FakeRouteTableManager{}
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

func TestLoadConfig(t *testing.T) {
	c, err := New("../tests/awsnycast.yaml", tim, rtm)
	assert.Nil(t, err)
	assert.NotNil(t, c)
}

func TestLoadConfigFails(t *testing.T) {
	_, err := New("../tests/doesnotexist.yaml", tim, rtm)
	if assert.NotNil(t, err) {
		assert.Equal(t, err.Error(), "open ../tests/doesnotexist.yaml: no such file or directory")
	}
}

func TestLoadConfigFailsValidation(t *testing.T) {
	_, err := New("../tests/invalid.yaml", tim, rtm)
	testhelpers.CheckOneMultiError(t, err, "Route tables a, route 0.0.0.0/0 cannot find healthcheck 'public'")
}

func TestLoadConfigHealthchecks(t *testing.T) {
	c, err := New("../tests/awsnycast.yaml", tim, rtm)
	assert.Nil(t, err, "Loading config failed")
	if assert.NotNil(t, c.Healthchecks) {
		h, ok := c.Healthchecks["public"]
		assert.Equal(t, ok, true, "c.Healthchecks['public'] not ok")
		assert.Equal(t, h.Type, "ping")
		assert.Equal(t, h.Destination, "8.8.8.8")
		assert.Equal(t, h.Rise, uint(2))
		assert.Equal(t, h.Fall, uint(10))
		assert.Equal(t, h.Every, uint(1))
	}
	if assert.NotNil(t, c.RouteTables) {
		a, ok := c.RouteTables["a"]
		assert.Equal(t, ok, true, "RouteTables a not ok")
		assert.Equal(t, a.Find.Type, "by_tag")
		v, ok := a.Find.Config["key"]
		assert.Equal(t, ok, true)
		assert.Equal(t, v, "Name")
		v, ok = a.Find.Config["value"]
		assert.Equal(t, ok, true)
		assert.Equal(t, v, "private a")
		routes := a.ManageRoutes
		assert.Equal(t, len(routes), 2)
		for _, route := range routes {
			if route.Cidr == "0.0.0.0/0" || route.Cidr == "192.168.1.1/32" {
				assert.Equal(t, route.Instance, "i-1234")
				if route.Cidr == "0.0.0.0/0" {
					assert.Equal(t, route.HealthcheckName, "public")
				} else {
					assert.Equal(t, route.HealthcheckName, "localservice")
				}
			} else {
				t.Log("CIDR unknown")
				t.Fail()
			}
		}
	}
	b, ok := c.RouteTables["b"]
	assert.Equal(t, ok, true)
	assert.Equal(t, b.Find.Type, "and")
}

func TestConfigDefault(t *testing.T) {
	r := make(map[string]*RouteTable)
	r["a"] = &RouteTable{
		ManageRoutes: []*aws.ManageRoutesSpec{&aws.ManageRoutesSpec{Cidr: "127.0.0.1"}},
	}
	c := Config{
		RouteTables: r,
	}
	assert.NotNil(t, c.Validate(tim, rtm))
	assert.NotNil(t, c.Healthchecks)
	assert.Equal(t, c.RouteTables["a"].ManageRoutes[0].Cidr, "127.0.0.1/32")
}

func TestConfigValidateNoRouteTables(t *testing.T) {
	c := Config{}
	err := c.Validate(tim, rtm)
	testhelpers.CheckOneMultiError(t, err, "No route_tables key in config")
}

func TestConfigValidateEmptyRouteTables(t *testing.T) {
	r := make(map[string]*RouteTable)
	c := Config{
		RouteTables: r,
	}
	err := c.Validate(tim, rtm)
	testhelpers.CheckOneMultiError(t, err, "No route_tables defined in config")
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
	testhelpers.CheckOneMultiError(t, err, "No manage_routes key in route table 'foo'")
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
	testhelpers.CheckOneMultiError(t, err, "cidr is not defined in foo")
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
	testhelpers.CheckOneMultiError(t, err, "Healthcheck foo has no destination set")
}

func TestConfigValidateNoHealthChecks(t *testing.T) {
	c_disk, _ := New("../tests/awsnycast.yaml", tim, rtm)
	c := Config{
		RouteTables: c_disk.RouteTables,
	}
	err := c.Validate(tim, rtm)
	assert.NotNil(t, err)
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
	assert.Nil(t, c.Validate(tim, rtm))
	rt := c.RouteTables["a"]
	ur := rt.ManageRoutes[0]
	assert.Equal(t, ur.Cidr, "127.0.0.1/32")
}

func TestConfigValidateEmpty(t *testing.T) {
	c := Config{}
	err := c.Validate(tim, rtm)
	testhelpers.CheckOneMultiError(t, err, "No route_tables key in config")
}

func TestRouteTableFindSpecDefault(t *testing.T) {
	r := RouteTableFindSpec{}
	r.Validate("foo")
	assert.NotNil(t, r.Config)
}
func TestRouteTableFindSpecValidate(t *testing.T) {
	c := make(map[string]interface{})
	c["key"] = "Name"
	c["value"] = "private a"
	r := RouteTableFindSpec{
		Type:   "by_tag",
		Config: c,
	}
	assert.Nil(t, r.Validate("foo"))
}

func TestRouteTableFindSpecValidateNoType(t *testing.T) {
	c := make(map[string]interface{})
	c["key"] = "Name"
	c["value"] = "private a"
	r := RouteTableFindSpec{
		Config: c,
	}
	err := r.Validate("foo")
	testhelpers.CheckOneMultiError(t, err, "Route find spec foo needs a type key")
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
	testhelpers.CheckOneMultiError(t, err, "Route find spec foo type 'doesnotexist' not known")
}

func TestRouteTableFindSpecValidateNoConfig(t *testing.T) {
	r := RouteTableFindSpec{
		Type: "by_tag",
	}
	err := r.Validate("foo")
	testhelpers.CheckOneMultiError(t, err, "Route find spec foo needs config")
}

func TestRouteTableDefaultEmpty(t *testing.T) {
	r := RouteTable{}
	r.Validate(tim, rtm, "foo", emptyHealthchecks, emptyHealthchecks)
	assert.NotNil(t, r.ManageRoutes)
	assert.NotNil(t, r.ec2RouteTables)
}

func TestRouteTableDefault(t *testing.T) {
	routes := make([]*aws.ManageRoutesSpec, 1)
	routes[0] = &aws.ManageRoutesSpec{
		Cidr: "127.0.0.1",
	}
	r := RouteTable{
		ManageRoutes: routes,
	}
	r.Validate(tim, rtm, "foo", emptyHealthchecks, emptyHealthchecks)
	assert.Equal(t, len(r.ManageRoutes), 1)
	routeSpec := r.ManageRoutes[0]
	assert.Equal(t, routeSpec.Cidr, "127.0.0.1/32")
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
	err := r.Validate(tim, rtm, "foo", emptyHealthchecks, emptyHealthchecks)
	testhelpers.CheckOneMultiError(t, err, "No manage_routes key in route table 'foo'")
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
	assert.Nil(t, r.Validate(tim, rtm, "foo", emptyHealthchecks, emptyHealthchecks))
}

func TestByTagRouteTableFindMissingKey(t *testing.T) {
	c := make(map[string]interface{})
	c["value"] = "foo"
	rts := RouteTableFindSpec{
		Type:   "by_tag",
		Config: c,
	}
	rtf, err := rts.GetFilter()
	assert.Nil(t, rtf)
	testhelpers.CheckOneMultiError(t, err, "No key in config for by_tag route table finder")
}

func TestByTagRouteTableFindMissingValue(t *testing.T) {
	c := make(map[string]interface{})
	c["key"] = "Name"
	rts := RouteTableFindSpec{
		Type:   "by_tag",
		Config: c,
	}
	rtf, err := rts.GetFilter()
	assert.Nil(t, rtf)
	testhelpers.CheckOneMultiError(t, err, "No value in config for by_tag route table finder")
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
	assert.NotNil(t, rtf)
	assert.Nil(t, err)
}

func TestRouteTableFindUnknownType(t *testing.T) {
	c := make(map[string]interface{})
	rts := RouteTableFindSpec{
		Type:   "unknown",
		Config: c,
	}
	rtf, err := rts.GetFilter()
	assert.Nil(t, rtf)
	assert.NotNil(t, err)
}

func TestUpdateEc2RouteTablesRouteTablesGetFilterFail(t *testing.T) {
	awsRt := make([]*ec2.RouteTable, 0)
	rt := &RouteTable{}
	err := rt.UpdateEc2RouteTables(awsRt)
	if assert.NotNil(t, err) {
		assert.Equal(t, err.Error(), "Route table finder type '' not found in the registry")
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
	if assert.NotNil(t, err) {
		assert.Equal(t, err.Error(), "No route table in AWS matched filter spec")
	}
	rt.Find.NoResultsOk = true
	err = rt.UpdateEc2RouteTables(awsRt)
	assert.Nil(t, err)
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
	assert.Nil(t, rt.UpdateEc2RouteTables(awsRt))
}

func TestRunEc2Updates(t *testing.T) {
	rt := &RouteTable{
		ManageRoutes: []*aws.ManageRoutesSpec{&aws.ManageRoutesSpec{Cidr: "127.0.0.1"}},
	}
	err := rt.Validate(tim, rtm, "foo", emptyHealthchecks, emptyHealthchecks)
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
	if assert.Nil(t, rt.RunEc2Updates(frtm, true)) {
		assert.Equal(t, *(frtm.RouteTable.RouteTableId), "rtb-9696cffe")
		assert.Equal(t, frtm.ManageRoutesSpec.Cidr, "127.0.0.1/32")
	}
	frtm.Error = errors.New("Test error")
	err = rt.RunEc2Updates(frtm, true)
	if assert.NotNil(t, err) {
		assert.Equal(t, err.Error(), "Test error")
	}
}

func TestRouteTableFindSpecAndNoFilters(t *testing.T) {
	c := make(map[string]interface{})
	_, err := RouteTableFindSpec{Config: c, Type: "and"}.GetFilter()
	testhelpers.CheckOneMultiError(t, err, "No filters in config for and route table finder")
}

func TestRouteTableFindSpecOrNoFilters(t *testing.T) {
	c := make(map[string]interface{})
	_, err := RouteTableFindSpec{Config: c, Type: "or"}.GetFilter()
	testhelpers.CheckOneMultiError(t, err, "No filters in config for or route table finder")
}

func TestRouteTableFindSpecSubnetNoSubnet(t *testing.T) {
	c := make(map[string]interface{})
	_, err := RouteTableFindSpec{Config: c, Type: "subnet"}.GetFilter()
	if assert.NotNil(t, err) {
		assert.Equal(t, err.Error(), "No subnet_id in config for subnet route table finder")
	}
}

func TestRouteTableFindSpecHasRouteToNoCidr(t *testing.T) {
	c := make(map[string]interface{})
	_, err := RouteTableFindSpec{Config: c, Type: "has_route_to"}.GetFilter()
	if assert.NotNil(t, err) {
		assert.Equal(t, err.Error(), "No cidr in config for has_route_to route table finder")
	}
}

func TestRouteTableFindSpecSubnet(t *testing.T) {
	c := make(map[string]interface{})
	c["subnet_id"] = "subnet-12345"
	_, err := RouteTableFindSpec{Config: c, Type: "subnet"}.GetFilter()
	assert.Nil(t, err)
}

func TestRouteTableFindSpecHasRouteTo(t *testing.T) {
	c := make(map[string]interface{})
	c["cidr"] = "0.0.0.0/0"
	_, err := RouteTableFindSpec{Config: c, Type: "has_route_to"}.GetFilter()
	assert.Nil(t, err)
}

func TestRouteTableFindSpecMain(t *testing.T) {
	c := make(map[string]interface{})
	spec := RouteTableFindSpec{Config: c, Type: "main", Not: true}
	f, err := spec.GetFilter()
	assert.Nil(t, err)
	assert.NotNil(t, f)
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
	assert.Nil(t, err)
	assert.Equal(t, len(filters), 2)
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
	assert.Nil(t, err)
	assert.NotNil(t, f)
	spec2 := RouteTableFindSpec{Config: c, Type: "or"}
	f, err = spec2.GetFilter()
	assert.Nil(t, err)
	assert.NotNil(t, f)
}

func TestGetFiltersListForSpecWrongType(t *testing.T) {
	c := make(map[string]interface{})
	c["filters"] = "foo"
	spec := RouteTableFindSpec{Config: c}
	_, err := getFiltersListForSpec(spec)
	testhelpers.CheckOneMultiError(t, err, "unexpected type string for 'filters' key")
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
	if assert.NotNil(t, err) {
		merr, ok := err.(*multierror.Error)
		if assert.Equal(t, ok, true) {
			assert.Equal(t, len(merr.Errors), 2, fmt.Sprintf("%d not 2 errors", len(merr.Errors)))
			assert.Equal(t, merr.Errors[1].Error(), "No value in config for by_tag route table finder for or route table finder")
		}
	}
}
