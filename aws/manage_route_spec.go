package aws

import (
	"errors"
	"fmt"
	log "github.com/bobtfish/logrus"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/bobtfish/AWSnycast/healthcheck"
	"github.com/bobtfish/AWSnycast/instancemetadata"
	"github.com/hashicorp/go-multierror"
	"net"
	"strings"
)

type ManageRoutesSpec struct {
	Cidr                      string                              `yaml:"cidr"`
	Instance                  string                              `yaml:"instance"`
	InstanceIsSelf            bool                                `yaml:"-"`
	HealthcheckName           string                              `yaml:"healthcheck"`
	RemoteHealthcheckName     string                              `yaml:"remote_healthcheck"`
	healthcheck               healthcheck.CanBeHealthy            `yaml:"-"`
	remotehealthchecktemplate *healthcheck.Healthcheck            `yaml:"-"`
	remotehealthchecks        map[string]*healthcheck.Healthcheck `yaml:"-"`
	IfUnhealthy               bool                                `yaml:"if_unhealthy"`
	ec2RouteTables            []*ec2.RouteTable                   `yaml:"-"`
	Manager                   RouteTableManager                   `yaml:"-"`
	NeverDelete               bool                                `yaml:"never_delete"`
	myIPAddress               string                              `yaml:"-"`
	RunBeforeReplaceRoute     []string                            `yaml:"run_before_replace_route"`
	RunAfterReplaceRoute      []string                            `yaml:"run_after_replace_route"`
	RunBeforeDeleteRoute      []string                            `yaml:"run_before_delete_route"`
	RunAfterDeleteRoute       []string                            `yaml:"run_after_delete_route"`
}

func (r *ManageRoutesSpec) Validate(meta instancemetadata.InstanceMetadata, manager RouteTableManager, name string, healthchecks map[string]*healthcheck.Healthcheck, remotehealthchecks map[string]*healthcheck.Healthcheck) error {
	r.myIPAddress = meta.IPAddress
	var result *multierror.Error
	r.Manager = manager
	r.ec2RouteTables = make([]*ec2.RouteTable, 0)
	r.remotehealthchecks = make(map[string]*healthcheck.Healthcheck)
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
			result = multierror.Append(result, errors.New(fmt.Sprintf("Route tables %s, route %s cannot find remote healthcheck '%s'", name, r.Cidr, r.RemoteHealthcheckName)))
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
	typeText := "local"
	if remote {
		typeText = "remote"
	}
	contextLogger := log.WithFields(log.Fields{
		"healtcheck_status": resText,
		"healthcheck_name":  r.HealthcheckName,
		"healthcheck_type":  typeText,
		"route_cidr":        r.Cidr,
	})
	contextLogger.Info("Healthcheck status change, reevaluating current routes")
	for _, rtb := range r.ec2RouteTables {
		innerLogger := contextLogger.WithFields(log.Fields{
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
	eniIdsToFetch := make([]*string, 0)
	routeEnis := make([]string, 0)
	for _, rtb := range r.ec2RouteTables {
		route := findRouteFromRouteTable(*rtb, r.Cidr)
		if route != nil {
			routeEnis = append(routeEnis, *route.NetworkInterfaceId)
			if _, ok := eniToIP[*route.NetworkInterfaceId]; !ok {
				eniIdsToFetch = append(eniIdsToFetch, route.NetworkInterfaceId)
			}
		}
	}
	if len(eniIdsToFetch) > 0 {
		out, err := r.Manager.(RouteTableManagerEC2).conn.DescribeNetworkInterfaces(&ec2.DescribeNetworkInterfacesInput{NetworkInterfaceIds: eniIdsToFetch})
		if err != nil {
			log.Error("Error " + err.Error())
			return
		}
		for _, iface := range out.NetworkInterfaces {
			eniToIP[*iface.NetworkInterfaceId] = *iface.PrivateIpAddress
		}
	}
	log.Debug(fmt.Sprintf("ENI %+v", eniToIP))
	healthchecks := make(map[string]bool)
	for ip, _ := range r.remotehealthchecks {
		healthchecks[ip] = false
	}
	for _, eniId := range routeEnis {
		ip := eniToIP[eniId]
		contextLogger := log.WithFields(log.Fields{"ip": ip})
		healthchecks[ip] = true
		if ip == r.myIPAddress {
			contextLogger.Debug("Skipping starting a remote healthcheck on myself")
			continue
		}
		if _, ok := r.remotehealthchecks[ip]; !ok {
			hc, err := r.remotehealthchecktemplate.NewWithDestination(ip)
			if err != nil {
				contextLogger.Error(err.Error())
			} else {
				r.remotehealthchecks[ip] = hc
				r.remotehealthchecks[ip].Run(true)
				contextLogger.Debug(fmt.Sprintf("New healthcheck being run"))
				go func() {
					c := hc.GetListener()
					for {
						res := <-c
						contextLogger.WithFields(log.Fields{"result": res}).Debug("Got result from remote healthchecl")
						r.handleHealthcheckResult(res, true, false)
					}
				}()
			}
		}
	}
	for ip, v := range healthchecks {
		if v {
			continue
		}
		log.WithFields(log.Fields{"ip": ip}).Debug("Stopping healthcheck")
		r.remotehealthchecks[ip].Stop()
		delete(r.remotehealthchecks, ip)
	}
}
