package config

import (
	"errors"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/bobtfish/AWSnycast/aws"
	"github.com/bobtfish/AWSnycast/healthcheck"
	"github.com/hashicorp/go-multierror"
)

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
		if err := v.Validate(instance, manager, name, healthchecks, remotehealthchecks); err != nil {
			result = multierror.Append(result, err)
		}
	}

	return result.ErrorOrNil()
}
