package aws

import (
	"errors"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/bobtfish/AWSnycast/healthcheck"
	"github.com/hashicorp/go-multierror"
	"net"
	"strings"
)

type MyEC2Conn interface {
	CreateRoute(*ec2.CreateRouteInput) (*ec2.CreateRouteOutput, error)
	ReplaceRoute(*ec2.ReplaceRouteInput) (*ec2.ReplaceRouteOutput, error)
	DescribeRouteTables(*ec2.DescribeRouteTablesInput) (*ec2.DescribeRouteTablesOutput, error)
	DeleteRoute(*ec2.DeleteRouteInput) (*ec2.DeleteRouteOutput, error)
	DescribeNetworkInterfaces(*ec2.DescribeNetworkInterfacesInput) (*ec2.DescribeNetworkInterfacesOutput, error)
	DescribeInstanceAttribute(*ec2.DescribeInstanceAttributeInput) (*ec2.DescribeInstanceAttributeOutput, error)
}

type ManageRoutesSpec struct {
	Cidr                  string                              `yaml:"cidr"`
	Instance              string                              `yaml:"instance"`
	InstanceIsSelf        bool                                `yaml:"-"`
	HealthcheckName       string                              `yaml:"healthcheck"`
	RemoteHealthcheckName string                              `yaml:"remote_healthcheck"`
	healthcheck           healthcheck.CanBeHealthy            `yaml:"-"`
	remotehealthcheck     *healthcheck.Healthcheck            `yaml:"-"`
	remotehealthchecks    map[string]*healthcheck.Healthcheck `yaml:"-"`
	IfUnhealthy           bool                                `yaml:"if_unhealthy"`
	ec2RouteTables        []*ec2.RouteTable                   `yaml:"-"`
	Manager               RouteTableManager                   `yaml:"-"`
	NeverDelete           bool                                `yaml:"never_delete"`
}

func (r *ManageRoutesSpec) Validate(instance string, manager RouteTableManager, name string, healthchecks map[string]*healthcheck.Healthcheck, remotehealthchecks map[string]*healthcheck.Healthcheck) error {
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
		r.Instance = instance
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
			r.remotehealthcheck = hc
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
			r.handleHealthcheckResult(<-c, noop)
		}
	}()
	return
}

func (r *ManageRoutesSpec) handleHealthcheckResult(res bool, noop bool) {
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
		if _, ok := r.remotehealthchecks[ip]; !ok {
			hc, err := r.remotehealthcheck.NewWithDestination(ip)
			if err != nil {
				log.Error(err.Error())
			} else {
				r.remotehealthchecks[ip] = hc
				r.remotehealthchecks[ip].Run(true)
				log.Debug(fmt.Sprintf("New healthcheck being run"))
				go func() {
					c := r.remotehealthchecks[ip].GetListener()
					for {
						res := <-c
						log.Debug("Got result from remote healthchecl")
						r.handleHealthcheckResult(res, false)
					}
				}()
			}
		}
	}
}

type RouteTableManager interface {
	GetRouteTables() ([]*ec2.RouteTable, error)
	ManageInstanceRoute(ec2.RouteTable, ManageRoutesSpec, bool) error
	InstanceIsRouter(string) bool
}

type RouteTableManagerEC2 struct {
	Region string
	conn   MyEC2Conn
}

func (m RouteTableManagerEC2) InstanceIsRouter(id string) bool {
	if v, ok := srcdstcheckForInstance[id]; ok {
		return v
	}
	out, err := m.conn.DescribeInstanceAttribute(&ec2.DescribeInstanceAttributeInput{
		Attribute:  aws.String("sourceDestCheck"),
		InstanceId: &id,
	})
	if err != nil {
		panic(err)
	}
	srcdstcheckForInstance[id] = !*(out.SourceDestCheck.Value)
	return srcdstcheckForInstance[id]
}

func getCreateRouteInput(rtb ec2.RouteTable, cidr string, instance string, noop bool) ec2.CreateRouteInput {
	return ec2.CreateRouteInput{
		RouteTableId:         rtb.RouteTableId,
		DestinationCidrBlock: aws.String(cidr),
		InstanceId:           aws.String(instance),
		DryRun:               aws.Bool(noop),
	}
}

func (r RouteTableManagerEC2) ManageInstanceRoute(rtb ec2.RouteTable, rs ManageRoutesSpec, noop bool) error {
	route := findRouteFromRouteTable(rtb, rs.Cidr)
	contextLogger := log.WithFields(log.Fields{
		"vpc":         *(rtb.VpcId),
		"rtb":         *(rtb.RouteTableId),
		"noop":        noop,
		"cidr":        rs.Cidr,
		"my_instance": rs.Instance,
	})
	if route != nil {
		if route.InstanceId != nil && *(route.InstanceId) == rs.Instance {
			if rs.HealthcheckName != "" && !rs.healthcheck.IsHealthy() && rs.healthcheck.CanPassYet() {
				if rs.NeverDelete {
					contextLogger.Info("Healthcheck unhealthy, but set to never_delete - ignoring")
					return nil
				}
				contextLogger.Info("Healthcheck unhealthy: deleting route")
				if err := r.DeleteInstanceRoute(rtb.RouteTableId, route, rs.Cidr, rs.Instance, noop); err != nil {
					return err
				}
				return nil
			}
			contextLogger.Debug("Currently routed by my instance")
			return nil
		}

		if err := r.ReplaceInstanceRoute(rtb.RouteTableId, route, rs.Cidr, rs.Instance, rs.IfUnhealthy, noop); err != nil {
			return err
		}
		return nil
	}
	if rs.HealthcheckName != "" && !rs.healthcheck.IsHealthy() {
		return nil
	}

	opts := getCreateRouteInput(rtb, rs.Cidr, rs.Instance, noop)

	contextLogger.Info("Creating route to my instance")
	if _, err := r.conn.CreateRoute(&opts); err != nil {
		return err
	}
	return nil
}

func findRouteFromRouteTable(rtb ec2.RouteTable, cidr string) *ec2.Route {
	for _, route := range rtb.Routes {
		if *(route.DestinationCidrBlock) == cidr {
			return route
		}
	}
	return nil
}

func (r RouteTableManagerEC2) DeleteInstanceRoute(routeTableId *string, route *ec2.Route, cidr string, instance string, noop bool) error {
	params := &ec2.DeleteRouteInput{
		DestinationCidrBlock: aws.String(cidr),
		RouteTableId:         routeTableId,
		DryRun:               aws.Bool(noop),
	}
	_, err := r.conn.DeleteRoute(params)
	contextLogger := log.WithFields(log.Fields{
		"cidr": cidr,
		"rtb":  *routeTableId,
	})
	if err != nil {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		contextLogger.WithFields(log.Fields{
			"err": err.Error(),
		}).Warn("Error deleting route")
		return err
	}
	contextLogger.Debug("Successfully deleted route")
	return nil
}

func (r RouteTableManagerEC2) ReplaceInstanceRoute(routeTableId *string, route *ec2.Route, cidr string, instance string, ifUnhealthy bool, noop bool) error {
	params := &ec2.ReplaceRouteInput{
		DestinationCidrBlock: aws.String(cidr),
		RouteTableId:         routeTableId,
		InstanceId:           aws.String(instance),
		DryRun:               aws.Bool(noop),
	}
	contextLogger := log.WithFields(log.Fields{
		"cidr":                cidr,
		"rtb":                 *routeTableId,
		"instance_id":         instance,
		"current_route_state": *(route.State),
	})
	if route.InstanceId != nil {
		contextLogger = log.WithFields(log.Fields{"current_instance_id": *(route.InstanceId)})
	}
	if ifUnhealthy && *(route.State) == "active" {
		contextLogger.Info("Not replacing route, as current route is active/healthy")
		return nil
	}
	_, err := r.conn.ReplaceRoute(params)
	if err != nil {
		contextLogger.WithFields(log.Fields{
			"err": err.Error(),
		}).Warn("Error replacing route")
		return err
	}
	contextLogger.Info("Replaced route")
	return nil
}

func (r RouteTableManagerEC2) GetRouteTables() ([]*ec2.RouteTable, error) {
	resp, err := r.conn.DescribeRouteTables(&ec2.DescribeRouteTablesInput{})
	if err != nil {
		log.WithFields(log.Fields{
			"err": err.Error(),
		}).Warn("Error on DescribeRouteTables")
		return []*ec2.RouteTable{}, err
	}
	return resp.RouteTables, nil
}

func NewRouteTableManager(region string, debug bool) RouteTableManager {
	r := RouteTableManagerEC2{}
	sess := session.New(&aws.Config{
		Region:     aws.String(region),
		MaxRetries: aws.Int(3),
	})
	r.conn = ec2.New(sess)
	return r
}
