package config

import (
	"errors"
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

type Healthcheck struct {
	Type        string `yaml:"type"`
	Destination string `yaml:"destination"`
	Rise        uint   `yaml:"rise"`
	Fall        uint   `yaml:"fall"`
	Every       uint   `yaml:"every"`
}

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
	Healthchecks map[string]Healthcheck `yaml:"healthchecks"`
	RouteTables  map[string]RouteTable  `yaml:"routetables"`
}

func (c *Config) Default() {
}
func (c Config) Validate(name string) error {
	return nil
}

func (r *RouteFindSpec) Default() {
}
func (r *RouteFindSpec) Validate(name string) error {
	return nil
}

func (r *UpsertRoutesSpec) Default() {
	if r.Cidr == "127.0.0.1" { // FIXME
		r.Cidr = "127.0.0.1/32"
	}
}
func (r *UpsertRoutesSpec) Validate(name string) error {
	if r.Cidr == "" {
		return errors.New(fmt.Sprintf("cidr is not defined in %s", name))
	}
	// FIXME
	return nil
}

func (r *RouteTable) Default() {
}
func (r RouteTable) Validate(name string) error {
	if r.UpsertRoutes == nil || len(r.UpsertRoutes) == 0 {
		return errors.New(fmt.Sprintf("No upsert_routes key in route table '%s'", name))
	}
	return nil
}

func (h *Healthcheck) Default() {
	if h.Rise == 0 {
		h.Rise = 2
	}
	if h.Fall == 0 {
		h.Fall = 3
	}
}

func (h Healthcheck) Validate() error {
	if h.Type != "ping" {
		return errors.New(fmt.Sprintf("Unknown healthcheck type '%s'", h.Type))
	}
	if h.Rise == 0 {
		return errors.New("rise must be > 0")
	}
	if h.Fall == 0 {
		return errors.New("fall must be > 0")
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
