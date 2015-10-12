package config

import (
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

func New(filename string) (*Config, error) {
	c := new(Config)
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return c, err
	}
	err = yaml.Unmarshal(data, &c)
	return c, err
}
