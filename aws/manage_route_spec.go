package aws

import (
	"errors"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/bobtfish/AWSnycast/healthcheck"
	"github.com/bobtfish/AWSnycast/instancemetadata"
	"github.com/hashicorp/go-multierror"
	"net"
	"strings"
)

type ManageRoutesSpec struct {
	Cidr                      string                   `yaml:"cidr"`
	Instance                  string                   `yaml:"instance"`
	InstanceIsSelf            bool                     `yaml:"-"`
	HealthcheckName           string                   `yaml:"healthcheck"`
	RemoteHealthcheckName     string                   `yaml:"remote_healthcheck"`
	healthcheck               healthcheck.CanBeHealthy `yaml:"-"`
	remotehealthchecktemplate *healthcheck.Healthcheck `yaml:"-"`
	remotehealthcheck         *healthcheck.Healthcheck `yaml:"-"`
	IfUnhealthy               bool                     `yaml:"if_unhealthy"`
	ec2RouteTables            []*ec2.RouteTable        `yaml:"-"`
	Manager                   RouteTableManager        `yaml:"-"`
	NeverDelete               bool                     `yaml:"never_delete"`
	myIPAddress		  string			`yaml:"-"`
}

func (r *ManageRoutesSpec) Validate(meta instancemetadata.InstanceMetadata, manager RouteTableManager, name string, healthchecks map[string]*healthcheck.Healthcheck, remotehealthchecks map[string]*healthcheck.Healthcheck) error {
	r.myIPAddress = meta.IPAddress
	var result *multierror.Error
	r.Manager = manager
	r.ec2RouteTables = make([]*ec2.RouteTable, 0)
	if r.Cidr == "" {
		result = multierror.Append(result, errors.New(fmt.Sprintf("cidr is not defined in %s", name)))
	} else {
		if !strings.Contains(r.Cidr, "/") {
			r.Cidr = fmt.Sprintf("%s/32", r.Cidr)
		}
		if _, _, err := net.ParseCIDR(r.Cidr); err != nil {
			result = multierror.Append(result, errors.New(fmt.Sprintf("Could not parse %s in %s", err.Error(), name)))
		}
	}
	if r.Instance == "" {
		r.Instance = "SELF"
	}
	if r.Instance == "SELF" {
		r.InstanceIsSelf = true
		r.Instance = meta.Instance
	}
	if r.HealthcheckName != "" {
		if hc, ok := healthchecks[r.HealthcheckName]; ok {
			r.healthcheck = hc
		} else {
			result = multierror.Append(result, errors.New(fmt.Sprintf("Route tables %s, route %s cannot find healthcheck '%s'", name, r.Cidr, r.HealthcheckName)))
		}
	}
	if r.RemoteHealthcheckName != "" {
		if hc, ok := remotehealthchecks[r.RemoteHealthcheckName]; ok {
			r.remotehealthchecktemplate = hc
		} else {
			result = multierror.Append(result, errors.New(fmt.Sprintf("Route table %s, route %s cannot find healthcheck '%s'", name, r.Cidr, r.RemoteHealthcheckName)))
		}
	}
	return result.ErrorOrNil()
}

func (r *ManageRoutesSpec) StartHealthcheckListener(noop bool) {
	if r.healthcheck == nil {
		return
	}
	go func() {
		c := r.healthcheck.GetListener()
		for {
			r.handleHealthcheckResult(<-c, false, noop)
		}
	}()
	return
}

func (r *ManageRoutesSpec) handleHealthcheckResult(res bool, remote bool, noop bool) {
	resText := "FAILED"
	if res {
		resText = "PASSED"
	}
	contextLogger := log.WithFields(log.Fields{
		"healtcheck_status": resText,
		"healthcheck_name":  r.HealthcheckName,
		"route_cidr":        r.Cidr,
	})
	contextLogger.Info("Healthcheck status change, reevaluating current routes")
	for _, rtb := range r.ec2RouteTables {
		innerLogger := contextLogger.WithFields(log.Fields{
			//"vpc": *(rtb.VpcId),
			"rtb": *(rtb.RouteTableId),
		})
		innerLogger.Debug("Working for one route table")
		if err := r.Manager.ManageInstanceRoute(*rtb, *r, noop); err != nil {
			innerLogger.WithFields(log.Fields{"err": err.Error()}).Warn("error")
		}
	}
}

func (r *ManageRoutesSpec) UpdateEc2RouteTables(rt []*ec2.RouteTable) {
	log.Debug(fmt.Sprintf("manange routes: %+v", rt))
	r.ec2RouteTables = rt
	r.UpdateRemoteHealthchecks()
}

var eniToIP map[string]string
var srcdstcheckForInstance map[string]bool

func init() {
	eniToIP = make(map[string]string)
	srcdstcheckForInstance = make(map[string]bool)
}

func (r *ManageRoutesSpec) UpdateRemoteHealthchecks() {
	if r.RemoteHealthcheckName == "" {
		return
	}
	eniIds := make([]*string, 0)
	for _, rtb := range r.ec2RouteTables {
		route := findRouteFromRouteTable(*rtb, r.Cidr)
		if route != nil {
			if _, ok := eniToIP[*route.NetworkInterfaceId]; !ok {
				eniIds = append(eniIds, route.NetworkInterfaceId)
			}
		}
	}
	if len(eniIds) > 0 {
		out, err := r.Manager.(RouteTableManagerEC2).conn.DescribeNetworkInterfaces(&ec2.DescribeNetworkInterfacesInput{NetworkInterfaceIds: eniIds})
		if err != nil {
			log.Error("Error " + err.Error())
			return
		}
		for _, iface := range out.NetworkInterfaces {
			eniToIP[*iface.NetworkInterfaceId] = *iface.PrivateIpAddress
		}
	}
	log.Debug(fmt.Sprintf("ENI %+v", eniToIP))
	for _, eniId := range eniIds {
		ip := eniToIP[*eniId]
		contextLogger := log.WithFields(log.Fields{"current_ip": ip})
		if r.remotehealthcheck != nil {
			if r.remotehealthcheck.Destination == ip {
				contextLogger.Debug("Same as current healthcheck, no update needed")
				return
			} else {
				contextLogger.WithFields(log.Fields{"old_ip": r.remotehealthcheck.Destination}).Info("Removing old remote healthcheck")
				r.remotehealthcheck.Stop()
				r.remotehealthcheck = nil
			}
		}
		hc, err := r.remotehealthchecktemplate.NewWithDestination(ip)
		if err != nil {
			contextLogger.Error(err.Error())
		} else {
			r.remotehealthcheck = hc
			r.remotehealthcheck.Run(true)
			contextLogger.Debug(fmt.Sprintf("New healthcheck being run"))
			go func() {
				c := r.remotehealthcheck.GetListener()
				for {
					res := <-c
					contextLogger.WithFields(log.Fields{"result": res}).Debug("Got result from remote healthchecl")
					r.handleHealthcheckResult(res, true, false)
				}
			}()
		}
	}
}
