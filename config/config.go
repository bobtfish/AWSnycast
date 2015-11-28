package config

import (
	"errors"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/bobtfish/AWSnycast/aws"
	"github.com/bobtfish/AWSnycast/healthcheck"
	"github.com/bobtfish/AWSnycast/instancemetadata"
	"github.com/hashicorp/go-multierror"
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

type Config struct {
	PollTime                   uint                                `yaml:"poll_time"`
	Healthchecks               map[string]*healthcheck.Healthcheck `yaml:"healthchecks"`
	RemoteHealthcheckTemplates map[string]*healthcheck.Healthcheck `yaml:"remote_healthchecks"`
	RouteTables                map[string]*RouteTable              `yaml:"routetables"`
}

type RouteTable struct {
	Find           RouteTableFindSpec      `yaml:"find"`
	ManageRoutes   []*aws.ManageRoutesSpec `yaml:"manage_routes"`
	ec2RouteTables []*ec2.RouteTable
}

type RouteTableFindSpec struct {
	NoResultsOk bool                   `yaml:"no_results_ok"`
	Type        string                 `yaml:"type"`
	Not         bool                   `yaml:"not"`
	Config      map[string]interface{} `yaml:"config"`
}

var routeFindTypes map[string]func(RouteTableFindSpec) (aws.RouteTableFilter, error)

func init() {
	routeFindTypes = make(map[string]func(RouteTableFindSpec) (aws.RouteTableFilter, error))
	routeFindTypes["by_tag"] = func(spec RouteTableFindSpec) (aws.RouteTableFilter, error) {
		var result *multierror.Error
		if _, ok := spec.Config["key"]; !ok {
			result = multierror.Append(result, errors.New("No key in config for by_tag route table finder"))
		}
		if _, ok := spec.Config["value"]; !ok {
			result = multierror.Append(result, errors.New("No value in config for by_tag route table finder"))
		}
		if err := result.ErrorOrNil(); err != nil {
			return nil, result
		}
		return aws.RouteTableFilterTagMatch{
			Key:   spec.Config["key"].(string),
			Value: spec.Config["value"].(string),
		}, nil
	}
	routeFindTypes["and"] = func(spec RouteTableFindSpec) (aws.RouteTableFilter, error) {
		filters, err := getFiltersListForSpec(spec)
		if err != nil {
			return nil, appendMultiError(err, "for and route table finder")
		}
		return aws.RouteTableFilterAnd{filters}, nil
	}
	routeFindTypes["or"] = func(spec RouteTableFindSpec) (aws.RouteTableFilter, error) {
		filters, err := getFiltersListForSpec(spec)
		if err != nil {
			return nil, appendMultiError(err, "for or route table finder")
		}
		return aws.RouteTableFilterOr{filters}, nil
	}
	routeFindTypes["main"] = func(spec RouteTableFindSpec) (aws.RouteTableFilter, error) {
		return aws.RouteTableFilterMain{}, nil
	}
	routeFindTypes["subnet"] = func(spec RouteTableFindSpec) (aws.RouteTableFilter, error) {
		if _, ok := spec.Config["subnet_id"]; !ok {
			return nil, errors.New("No subnet_id in config for subnet route table finder")
		}
		return aws.RouteTableFilterSubnet{spec.Config["subnet_id"].(string)}, nil
	}
	routeFindTypes["has_route_to"] = func(spec RouteTableFindSpec) (aws.RouteTableFilter, error) {
		if _, ok := spec.Config["cidr"]; !ok {
			return nil, errors.New("No cidr in config for has_route_to route table finder")
		}
		return aws.RouteTableFilterDestinationCidrBlock{DestinationCidrBlock: spec.Config["cidr"].(string)}, nil
	}
}

func appendMultiError(in *multierror.Error, a string) *multierror.Error {
	var result *multierror.Error
	for _, element := range in.Errors {
		result = multierror.Append(result, errors.New(element.Error()+" "+a))
	}
	return result
}

func getFiltersListForSpec(spec RouteTableFindSpec) ([]aws.RouteTableFilter, *multierror.Error) {
	var result *multierror.Error
	v, ok := spec.Config["filters"]
	if !ok {
		result = multierror.Append(errors.New("No filters in config"))
		return nil, result
	}
	var filters []aws.RouteTableFilter
	switch t := v.(type) {
	default:
		result = multierror.Append(result, errors.New(fmt.Sprintf("unexpected type %T for 'filters' key", t)))
	case []interface{}:
		for _, filter := range t { // I REGRET NOTHING
			filterRepacked, err := yaml.Marshal(filter)
			if err != nil {
				result = multierror.Append(result, err)
				continue
			}
			var spec RouteTableFindSpec
			err = yaml.Unmarshal(filterRepacked, &spec)
			if err != nil {
				result = multierror.Append(result, err)
				continue
			}
			filter, err := spec.GetFilter()
			if err != nil {
				result = multierror.Append(result, err)
				continue
			}
			filters = append(filters, filter)
		} // End lack of regret
	}
	return filters, result
}

func (spec RouteTableFindSpec) GetFilter() (aws.RouteTableFilter, error) {
	if genFilter, found := routeFindTypes[spec.Type]; found {
		filter, err := genFilter(spec)
		if err != nil {
			return filter, err
		}
		if spec.Not {
			return aws.RouteTableFilterNot{filter}, nil
		}
		return filter, nil
	}
	return nil, errors.New(fmt.Sprintf("Route table finder type '%s' not found in the registry", spec.Type))
}

func (r *RouteTable) UpdateEc2RouteTables(rt []*ec2.RouteTable) error {
	filter, err := r.Find.GetFilter()
	if err != nil {
		return err
	}
	r.ec2RouteTables = aws.FilterRouteTables(filter, rt)
	if len(r.ec2RouteTables) == 0 {
		if r.Find.NoResultsOk {
			return nil
		}
		return errors.New("No route table in AWS matched filter spec")
	}
	for _, manage := range r.ManageRoutes {
		manage.UpdateEc2RouteTables(r.ec2RouteTables)
	}
	return nil
}

func (r *RouteTable) RunEc2Updates(manager aws.RouteTableManager, noop bool) error {
	for _, rtb := range r.ec2RouteTables {
		contextLogger := log.WithFields(log.Fields{
			"rtb": *(rtb.RouteTableId),
		})
		contextLogger.Debug("Finder found route table")
		for _, manageRoute := range r.ManageRoutes {
			contextLogger.WithFields(log.Fields{"cidr": manageRoute.Cidr}).Debug("Trying to manage route")
			if err := manager.ManageInstanceRoute(*rtb, *manageRoute, noop); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *Config) Default(im instancemetadata.InstanceMetadata, manager aws.RouteTableManager) {
	if c.PollTime == 0 {
		c.PollTime = 300 // Default to every 5m
	}
	if c.Healthchecks == nil {
		c.Healthchecks = make(map[string]*healthcheck.Healthcheck)
	}
	if c.RemoteHealthcheckTemplates == nil {
		c.RemoteHealthcheckTemplates = make(map[string]*healthcheck.Healthcheck)
	}
	if c.RouteTables != nil {
		for _, v := range c.RouteTables {
			v.Default(im.Instance, manager)
		}
	} else {
		c.RouteTables = make(map[string]*RouteTable)
	}
	for _, v := range c.Healthchecks {
		v.Default()
	}
	for _, v := range c.RemoteHealthcheckTemplates {
		v.Default()
	}
}

func (c Config) Validate() error {
	var result *multierror.Error
	if c.RouteTables == nil {
		result = multierror.Append(result, errors.New("No route_tables key in config"))
	} else {
		if len(c.RouteTables) == 0 {
			result = multierror.Append(result, errors.New("No route_tables defined in config"))
		}
	}
	if c.Healthchecks != nil {
		for k, v := range c.Healthchecks {
			if err := v.Validate(k, false); err != nil {
				result = multierror.Append(result, err)
			}
		}
	}
	if c.RemoteHealthcheckTemplates != nil {
		for k, v := range c.RemoteHealthcheckTemplates {
			if err := v.Validate(k, true); err != nil {
				result = multierror.Append(result, err)
			}
		}
	}
	if c.RouteTables != nil {
		for k, v := range c.RouteTables {
			if err := v.Validate(k, c.Healthchecks, c.RemoteHealthcheckTemplates); err != nil {
				result = multierror.Append(result, err)
			}
		}
	}
	return result.ErrorOrNil()
}

func (r *RouteTableFindSpec) Default() {
	if r.Config == nil {
		r.Config = make(map[string]interface{})
	}
}
func (r *RouteTableFindSpec) Validate(name string) error {
	if r.Type == "" {
		return errors.New(fmt.Sprintf("Route find spec %s needs a type key", name))
	}
	if r.Type != "by_tag" {
		return errors.New(fmt.Sprintf("Route find spec %s type '%s' not known", name, r.Type))
	}
	if r.Config == nil {
		return errors.New("No config supplied")
	}
	return nil
}

func (r *RouteTable) Default(instance string, manager aws.RouteTableManager) {
	r.Find.Default()
	if r.ManageRoutes == nil {
		r.ManageRoutes = make([]*aws.ManageRoutesSpec, 0)
	}
	for _, v := range r.ManageRoutes {
		v.Default(instance, manager)
	}
	if r.ec2RouteTables == nil {
		r.ec2RouteTables = make([]*ec2.RouteTable, 0)
	}
}
func (r RouteTable) Validate(name string, healthchecks map[string]*healthcheck.Healthcheck, remotehealthchecks map[string]*healthcheck.Healthcheck) error {
	if r.ManageRoutes == nil || len(r.ManageRoutes) == 0 {
		return errors.New(fmt.Sprintf("No manage_routes key in route table '%s'", name))
	}
	for _, v := range r.ManageRoutes {
		if err := v.Validate(name, healthchecks, remotehealthchecks); err != nil {
			return err
		}
	}
	return nil
}

func New(filename string, im instancemetadata.InstanceMetadata, manager aws.RouteTableManager) (*Config, error) {
	c := new(Config)
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return c, err
	}
	err = yaml.Unmarshal(data, &c)
	if err == nil {
		c.Default(im, manager)
		err = c.Validate()
	}
	return c, err
}
