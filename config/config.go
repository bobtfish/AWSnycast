package config

import (
	"errors"
	"fmt"
	"github.com/bobtfish/AWSnycast/healthcheck"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"net"
	"strings"
)

type RouteFindSpec struct {
	Type   string            `yaml:"type"`
	Config map[string]string `yaml:"config"`
}

type UpsertRoutesSpec struct {
	Cidr        string `yaml:"cidr"`
	Instance    string `yaml:"instance"`
	Healthcheck string `yaml:"healthcheck"`
}

type RouteTable struct {
	Find         RouteFindSpec      `yaml:"find"`
	UpsertRoutes []UpsertRoutesSpec `yaml:"upsert_routes"`
}

type Config struct {
	Healthchecks map[string]healthcheck.Healthcheck `yaml:"healthchecks"`
	RouteTables  map[string]RouteTable              `yaml:"routetables"`
}

func (c *Config) Default() {
	if c.Healthchecks == nil {
		c.Healthchecks = make(map[string]healthcheck.Healthcheck)
	}
	if c.RouteTables != nil {
		for k, v := range c.RouteTables {
			v.Default()
			c.RouteTables[k] = v
		}
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
	for k, v := range c.RouteTables {
		if err := v.Validate(k); err != nil {
			return err
		}
	}
	for k, v := range c.Healthchecks {
		if err := v.Validate(k); err != nil {
			return err
		}
	}
	return nil
}

func (r *RouteFindSpec) Default() {
	if r.Config == nil {
		r.Config = make(map[string]string)
	}
}
func (r *RouteFindSpec) Validate(name string) error {
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

func (r *UpsertRoutesSpec) Default() {
	if !strings.Contains(r.Cidr, "/") {
		r.Cidr = fmt.Sprintf("%s/32", r.Cidr)
	}
}
func (r *UpsertRoutesSpec) Validate(name string) error {
	if r.Cidr == "" {
		return errors.New(fmt.Sprintf("cidr is not defined in %s", name))
	}
	if _, _, err := net.ParseCIDR(r.Cidr); err != nil {
		return errors.New(fmt.Sprintf("Could not parse %s in %s", err.Error(), name))
	}
	return nil
}

func (r *RouteTable) Default() {
	r.Find.Default()
	if r.UpsertRoutes == nil {
		r.UpsertRoutes = make([]UpsertRoutesSpec, 0)
	}
	n := make([]UpsertRoutesSpec, len(r.UpsertRoutes))
	for i, v := range r.UpsertRoutes {
		v.Default()
		n[i] = v
	}
	r.UpsertRoutes = n
}
func (r RouteTable) Validate(name string) error {
	if r.UpsertRoutes == nil || len(r.UpsertRoutes) == 0 {
		return errors.New(fmt.Sprintf("No upsert_routes key in route table '%s'", name))
	}
	for _, v := range r.UpsertRoutes {
		if err := v.Validate(name); err != nil {
			return err
		}
	}
	return nil
}

func New(filename string) (*Config, error) {
	c := new(Config)
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return c, err
	}
	err = yaml.Unmarshal(data, &c)
	if err == nil {
		c.Default()
	}
	return c, err
}
