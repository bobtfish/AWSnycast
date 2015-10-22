package config

import (
	"errors"
	"fmt"
	"github.com/bobtfish/AWSnycast/aws"
	"github.com/bobtfish/AWSnycast/healthcheck"
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

type RouteTableFindSpec struct {
	Type   string            `yaml:"type"`
	Config map[string]string `yaml:"config"`
}

var routeFindTypes map[string]func(RouteTableFindSpec) (aws.RouteTableFilter, error)

func init() {
	routeFindTypes = make(map[string]func(RouteTableFindSpec) (aws.RouteTableFilter, error))
	routeFindTypes["by_tag"] = func(spec RouteTableFindSpec) (aws.RouteTableFilter, error) {
		if _, ok := spec.Config["key"]; !ok {
			return nil, errors.New("No key in config for by_tag route table finder")
		}
		if _, ok := spec.Config["value"]; !ok {
			return nil, errors.New("No value in config for by_tag route table finder")
		}
		return aws.RouteTableFilterTagMatch{
			Key:   spec.Config["key"],
			Value: spec.Config["value"],
		}, nil
	}
}

func (spec RouteTableFindSpec) GetFilter() (aws.RouteTableFilter, error) {
	if genFilter, found := routeFindTypes[spec.Type]; found {
		return genFilter(spec)
	}
	return nil, errors.New(fmt.Sprintf("Healthcheck type '%s' not found in the healthcheck registry", spec.Type))
}

type RouteTable struct {
	Find         RouteTableFindSpec      `yaml:"find"`
	ManageRoutes []*aws.ManageRoutesSpec `yaml:"manage_routes"`
}

type Config struct {
	Healthchecks map[string]*healthcheck.Healthcheck `yaml:"healthchecks"`
	RouteTables  map[string]*RouteTable              `yaml:"routetables"`
}

func (c *Config) Default(instance string) {
	if c.Healthchecks == nil {
		c.Healthchecks = make(map[string]*healthcheck.Healthcheck)
	}
	if c.RouteTables != nil {
		for _, v := range c.RouteTables {
			v.Default(instance)
		}
	} else {
		c.RouteTables = make(map[string]*RouteTable)
	}
	for _, v := range c.Healthchecks {
		v.Default()
	}
}
func (c Config) Validate() error {
	if c.RouteTables == nil {
		return errors.New("No route_tables key in config")
	}
	if len(c.RouteTables) == 0 {
		return errors.New("No route_tables defined in config")
	}
	if c.Healthchecks != nil {
		for k, v := range c.Healthchecks {
			if err := v.Validate(k); err != nil {
				return err
			}
		}
	}
	for k, v := range c.RouteTables {
		if err := v.Validate(k, c.Healthchecks); err != nil {
			return err
		}
	}
	return nil
}

func (r *RouteTableFindSpec) Default() {
	if r.Config == nil {
		r.Config = make(map[string]string)
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

func (r *RouteTable) Default(instance string) {
	r.Find.Default()
	if r.ManageRoutes == nil {
		r.ManageRoutes = make([]*aws.ManageRoutesSpec, 0)
	}
	for _, v := range r.ManageRoutes {
		v.Default(instance)
	}
}
func (r RouteTable) Validate(name string, healthchecks map[string]*healthcheck.Healthcheck) error {
	if r.ManageRoutes == nil || len(r.ManageRoutes) == 0 {
		return errors.New(fmt.Sprintf("No manage_routes key in route table '%s'", name))
	}
	for _, v := range r.ManageRoutes {
		if err := v.Validate(name, healthchecks); err != nil {
			return err
		}
	}
	return nil
}

func New(filename string, instance string) (*Config, error) {
	c := new(Config)
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return c, err
	}
	err = yaml.Unmarshal(data, &c)
	if err == nil {
		c.Default(instance)
	}
	return c, err
}
