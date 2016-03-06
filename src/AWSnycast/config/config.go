package config

import (
	"AWSnycast/aws"
	"AWSnycast/healthcheck"
	"AWSnycast/instancemetadata"
	"errors"
	"github.com/hashicorp/go-multierror"
	"github.com/spf13/viper"
)

type Config struct {
	PollTime                   uint                                `yaml:"poll_time"`
	Healthchecks               map[string]*healthcheck.Healthcheck `yaml:"healthchecks"`
	RemoteHealthcheckTemplates map[string]*healthcheck.Healthcheck `yaml:"remote_healthchecks"`
	RouteTables                map[string]*RouteTable              `yaml:"routetables"`
}

func New(filename string, im instancemetadata.InstanceMetadata, manager aws.RouteTableManager) (*Config, error) {
	var c *Config

	viper.SetConfigType("yaml")
	viper.SetConfigFile(filename)
	err := viper.ReadInConfig()
	if err != nil {
		return c, err
	}
	err = viper.Unmarshal(&c)
	if err == nil {
		err = c.Validate(im, manager)
	}
	return c, err
}

func (c *Config) Validate(im instancemetadata.InstanceMetadata, manager aws.RouteTableManager) error {
	if c.PollTime == 0 {
		c.PollTime = 300 // Default to every 5m
	}
	var result *multierror.Error
	if c.RouteTables == nil {
		result = multierror.Append(result, errors.New("No route_tables key in config"))
	} else {
		if len(c.RouteTables) == 0 {
			result = multierror.Append(result, errors.New("No route_tables defined in config"))
		} else {
			for k, v := range c.RouteTables {
				if err := v.Validate(im, manager, k, c.Healthchecks, c.RemoteHealthcheckTemplates); err != nil {
					result = multierror.Append(result, err)
				}
			}
		}
	}
	if c.Healthchecks != nil {
		for k, v := range c.Healthchecks {
			if err := v.Validate(k, false); err != nil {
				result = multierror.Append(result, err)
			}
		}
	} else {
		c.Healthchecks = make(map[string]*healthcheck.Healthcheck)
	}
	if c.RemoteHealthcheckTemplates != nil {
		for k, v := range c.RemoteHealthcheckTemplates {
			if err := v.Validate(k, true); err != nil {
				result = multierror.Append(result, err)
			}
		}
	} else {
		c.RemoteHealthcheckTemplates = make(map[string]*healthcheck.Healthcheck)
	}
	return result.ErrorOrNil()
}
