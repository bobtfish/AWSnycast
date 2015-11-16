package aws

import (
	"errors"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/bobtfish/AWSnycast/healthcheck"
	"net"
	"strings"
)

type MyEC2Conn interface {
	CreateRoute(*ec2.CreateRouteInput) (*ec2.CreateRouteOutput, error)
	ReplaceRoute(*ec2.ReplaceRouteInput) (*ec2.ReplaceRouteOutput, error)
	DescribeRouteTables(*ec2.DescribeRouteTablesInput) (*ec2.DescribeRouteTablesOutput, error)
	DeleteRoute(*ec2.DeleteRouteInput) (*ec2.DeleteRouteOutput, error)
}

type ManageRoutesSpec struct {
	Cidr            string                   `yaml:"cidr"`
	Instance        string                   `yaml:"instance"`
	InstanceIsSelf  bool                     `yaml:"-"`
	HealthcheckName string                   `yaml:"healthcheck"`
	healthcheck     healthcheck.CanBeHealthy `yaml:"-"`
	IfUnhealthy     bool                     `yaml:"if_unhealthy"`
	ec2RouteTables  []*ec2.RouteTable        `yaml:"-"`
	Manager         RouteTableManager        `yaml:"-"`
	NeverDelete     bool                     `yaml:"never_delete"`
}

func (r *ManageRoutesSpec) Default(instance string, manager RouteTableManager) {
	if !strings.Contains(r.Cidr, "/") {
		r.Cidr = fmt.Sprintf("%s/32", r.Cidr)
	}
	if r.Instance == "" {
		r.Instance = "SELF"
	}
	if r.Instance == "SELF" {
		r.InstanceIsSelf = true
		r.Instance = instance
	}
	r.ec2RouteTables = make([]*ec2.RouteTable, 0)
	r.Manager = manager
}

func (r *ManageRoutesSpec) Validate(name string, healthchecks map[string]*healthcheck.Healthcheck) error {
	if r.Cidr == "" {
		return errors.New(fmt.Sprintf("cidr is not defined in %s", name))
	}
	if _, _, err := net.ParseCIDR(r.Cidr); err != nil {
		return errors.New(fmt.Sprintf("Could not parse %s in %s", err.Error(), name))
	}
	if r.HealthcheckName != "" {
		hc, ok := healthchecks[r.HealthcheckName]
		if !ok {
			return errors.New(fmt.Sprintf("Route table %s, upsert %s cannot find healthcheck '%s'", name, r.Cidr, r.HealthcheckName))
		}
		r.healthcheck = hc
	}
	return nil
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
}

type RouteTableManager interface {
	GetRouteTables() ([]*ec2.RouteTable, error)
	ManageInstanceRoute(ec2.RouteTable, ManageRoutesSpec, bool) error
}

type RouteTableManagerEC2 struct {
	Region string
	conn   MyEC2Conn
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
		"cidr":        cidr,
		"rtb":         *routeTableId,
		"instance_id": instance,
	})
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

type RouteTableFilter interface {
	Keep(*ec2.RouteTable) bool
}

type RouteTableFilterAlways struct{}

func (fs RouteTableFilterAlways) Keep(rt *ec2.RouteTable) bool {
	return false
}

type RouteTableFilterNot struct {
	Filter RouteTableFilter
}

func (fs RouteTableFilterNot) Keep(rt *ec2.RouteTable) bool {
	return !fs.Filter.Keep(rt)
}

type RouteTableFilterNever struct{}

func (fs RouteTableFilterNever) Keep(rt *ec2.RouteTable) bool {
	return true
}

type RouteTableFilterAnd struct {
	RouteTableFilters []RouteTableFilter
}

func (fs RouteTableFilterAnd) Keep(rt *ec2.RouteTable) bool {
	for _, f := range fs.RouteTableFilters {
		if !f.Keep(rt) {
			return false
		}
	}
	return true
}

type RouteTableFilterOr struct {
	RouteTableFilters []RouteTableFilter
}

func (fs RouteTableFilterOr) Keep(rt *ec2.RouteTable) bool {
	for _, f := range fs.RouteTableFilters {
		if f.Keep(rt) {
			return true
		}
	}
	return false
}

type RouteTableFilterMain struct{}

func (fs RouteTableFilterMain) Keep(rt *ec2.RouteTable) bool {
	for _, a := range rt.Associations {
		if *(a.Main) {
			return true
		}
	}
	return false
}

func FilterRouteTables(f RouteTableFilter, tables []*ec2.RouteTable) []*ec2.RouteTable {
	out := make([]*ec2.RouteTable, 0, len(tables))
	for _, rtb := range tables {
		if f.Keep(rtb) {
			out = append(out, rtb)
		}
	}
	return out
}

func RouteTableForSubnet(subnet string, tables []*ec2.RouteTable) *ec2.RouteTable {
	subnet_rtb := FilterRouteTables(RouteTableFilterSubnet{SubnetId: subnet}, tables)
	if len(subnet_rtb) == 0 {
		main_rtbs := FilterRouteTables(RouteTableFilterMain{}, tables)
		if len(main_rtbs) == 0 {
			return nil
		}
		return main_rtbs[0]
	}
	return subnet_rtb[0]
}

type RouteTableFilterSubnet struct {
	SubnetId string
}

func (fs RouteTableFilterSubnet) Keep(rt *ec2.RouteTable) bool {
	for _, a := range rt.Associations {
		if a.SubnetId != nil && *(a.SubnetId) == fs.SubnetId {
			return true
		}
	}
	return false
}

type RouteTableFilterDestinationCidrBlock struct {
	DestinationCidrBlock string
	ViaIGW               bool
	ViaInstance          bool
	InstanceNotActive    bool
}

func (fs RouteTableFilterDestinationCidrBlock) Keep(rt *ec2.RouteTable) bool {
	for _, r := range rt.Routes {
		if r.DestinationCidrBlock != nil && *(r.DestinationCidrBlock) == fs.DestinationCidrBlock {
			if fs.ViaIGW {
				if r.GatewayId != nil && strings.HasPrefix(*(r.GatewayId), "igw-") {
					return true
				}
			} else {
				if fs.ViaInstance {
					if r.InstanceId != nil {
						if fs.InstanceNotActive {
							if *(r.State) != "active" {
								return true
							}
						} else {
							return true
						}
					}
				} else {
					return true
				}
			}
		}
	}
	return false
}

type RouteTableFilterTagMatch struct {
	Key   string
	Value string
}

func (fs RouteTableFilterTagMatch) Keep(rt *ec2.RouteTable) bool {
	for _, t := range rt.Tags {
		if *(t.Key) == fs.Key && *(t.Value) == fs.Value {
			return true
		}
	}
	return false
}
