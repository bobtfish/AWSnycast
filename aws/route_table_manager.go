package aws

import (
	log "github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

type MyEC2Conn interface {
	CreateRoute(*ec2.CreateRouteInput) (*ec2.CreateRouteOutput, error)
	ReplaceRoute(*ec2.ReplaceRouteInput) (*ec2.ReplaceRouteOutput, error)
	DescribeRouteTables(*ec2.DescribeRouteTablesInput) (*ec2.DescribeRouteTablesOutput, error)
	DeleteRoute(*ec2.DeleteRouteInput) (*ec2.DeleteRouteOutput, error)
	DescribeNetworkInterfaces(*ec2.DescribeNetworkInterfacesInput) (*ec2.DescribeNetworkInterfacesOutput, error)
	DescribeInstanceAttribute(*ec2.DescribeInstanceAttributeInput) (*ec2.DescribeInstanceAttributeOutput, error)
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
