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
				if err := v.Validate(im.Instance, manager, k, c.Healthchecks, c.RemoteHealthcheckTemplates); err != nil {
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

func (r *RouteTable) Validate(instance string, manager aws.RouteTableManager, name string, healthchecks map[string]*healthcheck.Healthcheck, remotehealthchecks map[string]*healthcheck.Healthcheck) error {
	if r.ManageRoutes == nil {
		r.ManageRoutes = make([]*aws.ManageRoutesSpec, 0)
	}
	var result *multierror.Error
	if len(r.ManageRoutes) == 0 {
		result = multierror.Append(result, errors.New(fmt.Sprintf("No manage_routes key in route table '%s'", name)))
	}
	if err := r.Find.Validate(name); err != nil {
		result = multierror.Append(result, err)
	}
	if r.ec2RouteTables == nil {
		r.ec2RouteTables = make([]*ec2.RouteTable, 0)
	}
	for _, v := range r.ManageRoutes {
		v.Default(instance, manager)
		if err := v.Validate(name, healthchecks, remotehealthchecks); err != nil {
			result = multierror.Append(result, err)
		}
	}

	return result.ErrorOrNil()
}

func New(filename string, im instancemetadata.InstanceMetadata, manager aws.RouteTableManager) (*Config, error) {
	c := new(Config)
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return c, err
	}
	err = yaml.Unmarshal(data, &c)
	if err == nil {
		err = c.Validate(im, manager)
	}
	return c, err
}
