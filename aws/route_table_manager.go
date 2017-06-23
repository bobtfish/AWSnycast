package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/bobtfish/AWSnycast/version"
	log "github.com/bobtfish/logrus"
	"os/exec"
)

type MyEC2Conn interface {
	CreateRoute(*ec2.CreateRouteInput) (*ec2.CreateRouteOutput, error)
	ReplaceRoute(*ec2.ReplaceRouteInput) (*ec2.ReplaceRouteOutput, error)
	DescribeRouteTables(*ec2.DescribeRouteTablesInput) (*ec2.DescribeRouteTablesOutput, error)
	DeleteRoute(*ec2.DeleteRouteInput) (*ec2.DeleteRouteOutput, error)
	DescribeNetworkInterfaces(*ec2.DescribeNetworkInterfacesInput) (*ec2.DescribeNetworkInterfacesOutput, error)
	DescribeInstanceAttribute(*ec2.DescribeInstanceAttributeInput) (*ec2.DescribeInstanceAttributeOutput, error)
	DescribeInstanceStatus(*ec2.DescribeInstanceStatusInput) (*ec2.DescribeInstanceStatusOutput, error)
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
	if rs.HealthcheckName != "" {
		contextLogger = contextLogger.WithFields(log.Fields{
			"healthcheck":         rs.HealthcheckName,
			"healthcheck_healthy": rs.healthcheck.IsHealthy(),
			"healthcheck_ready":   rs.healthcheck.CanPassYet(),
		})
	}
	if rs.RemoteHealthcheckName != "" {
		contextLogger = contextLogger.WithFields(log.Fields{
			"remote_healthcheck": rs.RemoteHealthcheckName,
		})
	}
	if route != nil {
		if route.InstanceId != nil {
			contextLogger = contextLogger.WithFields(log.Fields{
				"instance_id": *(route.InstanceId),
			})
			if *(route.InstanceId) == rs.Instance {
				if rs.HealthcheckName != "" && !rs.healthcheck.IsHealthy() && rs.healthcheck.CanPassYet() {
					if rs.NeverDelete {
						contextLogger.Info("Healthcheck unhealthy, but set to never_delete - ignoring")
						return nil
					}
					contextLogger.Info("Healthcheck unhealthy: deleting route")
					if len(rs.RunBeforeDeleteRoute) > 0 {
						cmd := rs.RunBeforeDeleteRoute[0]
						if err := exec.Command(cmd, rs.RunBeforeDeleteRoute[1:]...).Run(); err != nil {
							contextLogger.WithFields(log.Fields{"err": err.Error()}).Debug("RunBeforeDeleteRoute failed")
						}
					}
					if err := r.DeleteInstanceRoute(rtb.RouteTableId, route, rs.Cidr, rs.Instance, noop); err != nil {
						return err
					}
					if len(rs.RunAfterDeleteRoute) > 0 {
						cmd := rs.RunAfterDeleteRoute[0]
						if err := exec.Command(cmd, rs.RunAfterDeleteRoute[1:]...).Run(); err != nil {
							contextLogger.WithFields(log.Fields{"err": err.Error()}).Debug("RunAfterDeleteRoute failed")
						}
					}
					return nil
				}
				contextLogger.Debug("Currently routed by this instance, doing nothing")
				return nil
			} else {
				contextLogger.Debug("Not routed by my instance - evaluate for replacement")
			}
		}

		if err := r.ReplaceInstanceRoute(rtb.RouteTableId, route, rs, noop); err != nil {
			return err
		}
		return nil
	}

	// These is no pre-existing route
	if rs.HealthcheckName != "" && !rs.healthcheck.IsHealthy() {
		if rs.healthcheck.CanPassYet() {
			contextLogger.Info("Healthcheck unhealthy: not creating route")
		} else {
			contextLogger.Debug("Healthcheck cannot be healthy yet: not creating route")
		}
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
		if route.DestinationCidrBlock != nil && *(route.DestinationCidrBlock) == cidr {
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

func (r RouteTableManagerEC2) checkRemoteHealthCheck(contextLogger *log.Entry, route *ec2.Route, rs ManageRoutesSpec) bool {
	contextLogger = contextLogger.WithFields(log.Fields{
		"remote_healthcheck": rs.RemoteHealthcheckName,
		"current_eni":        *(route.NetworkInterfaceId),
	})
	contextLogger.Info("Has remote healthcheck ")
	if ip, ok := eniToIP[*route.NetworkInterfaceId]; ok {
		contextLogger = contextLogger.WithFields(log.Fields{"current_ip": ip})
		if hc, ok := rs.remotehealthchecks[ip]; ok {
			contextLogger = contextLogger.WithFields(log.Fields{
				"healthcheck_healthy": hc.IsHealthy(),
				"healthcheck_ready":   hc.CanPassYet(),
			})
			contextLogger.Info("Has remote healthcheck instance")
			if hc.CanPassYet() {
				if hc.IsHealthy() {
					contextLogger.Info("Not replacing route, as current route and remote healthcheck is healthy")
					return false
				} else {
					contextLogger.Debug("Replacing route as remote healthcheck is unhealthy")
				}
			} else {
				contextLogger.Debug("Not replacing route as remote healthcheck cannot pass yet")
				return false
			}
		} else {
			contextLogger.Error("Cannot find healthcheck")
			return false
		}
	} else {
		contextLogger.Error("Cannot find ip for ENI")
		return false
	}
	return true
}

func (r RouteTableManagerEC2) ReplaceInstanceRoute(routeTableId *string, route *ec2.Route, rs ManageRoutesSpec, noop bool) error {
	cidr := rs.Cidr
	instance := rs.Instance
	ifUnhealthy := rs.IfUnhealthy
	disi := &ec2.DescribeInstanceStatusInput{
		IncludeAllInstances: aws.Bool(false),
		InstanceIds:         []*string{aws.String(instance)},
	}
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
		contextLogger = contextLogger.WithFields(log.Fields{"current_instance_id": *(route.InstanceId)})
	}
	if ifUnhealthy {
		if *(route.State) == "active" {
			if rs.RemoteHealthcheckName != "" {
				if !r.checkRemoteHealthCheck(contextLogger, route, rs) {
					return nil
				}
			}
			o, err := r.conn.DescribeInstanceStatus(disi)
			if err != nil {
				contextLogger.WithFields(log.Fields{"err": err.Error()}).Error("Error trying to DescribeInstanceStatus, not replacing route")
				return nil
			}
			if len(o.InstanceStatuses) != 1 {
				contextLogger.Error("Did not get 1 instance for DescribeInstanceStatus")
				return nil
			}
			is := o.InstanceStatuses[0]
			instanceHealthOK := true
			if *(is.InstanceStatus.Status) == "impaired" {
				instanceHealthOK = false
			}
			systemHealthOK := true
			if *(is.SystemStatus.Status) == "impaired" {
				systemHealthOK = false
			}
			contextLogger = contextLogger.WithFields(log.Fields{"instanceHealthOK": instanceHealthOK, "systemHealthOK": systemHealthOK})
			if systemHealthOK && instanceHealthOK {
				contextLogger.Info("Not replacing route, as current route is active and instance is healthy")
				return nil
			}
			contextLogger.Info("Current route is active, but instance with the route is unhealthy, evaluating for replacement")
		}
	}
	if rs.HealthcheckName != "" && !rs.healthcheck.IsHealthy() && rs.healthcheck.CanPassYet() {
		contextLogger.Info("Not replacing route, as local healthcheck is failing")
		return nil
	}
	if len(rs.RunBeforeReplaceRoute) > 0 {
		cmd := rs.RunBeforeReplaceRoute[0]
		if err := exec.Command(cmd, rs.RunBeforeReplaceRoute[1:]...).Run(); err != nil {
			contextLogger.WithFields(log.Fields{"err": err.Error()}).Debug("RunBeforeReplaceRoute failed")
		}
	}
	_, err := r.conn.ReplaceRoute(params)
	if err != nil {
		contextLogger.WithFields(log.Fields{
			"err": err.Error(),
		}).Warn("Error replacing route")
		return err
	}
	contextLogger.Info("Replaced route")
	if len(rs.RunAfterReplaceRoute) > 0 {
		cmd := rs.RunAfterReplaceRoute[0]
		if err := exec.Command(cmd, rs.RunAfterReplaceRoute[1:]...).Run(); err != nil {
			contextLogger.WithFields(log.Fields{"err": err.Error()}).Debug("RunAfterReplaceRoute failed")
		}
	}
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
	sess.Handlers.Build.PushFrontNamed(addAWSnycastToUserAgent)
	r.conn = ec2.New(sess)
	return r
}

// addAWSnycastToUserAgent is a named handler that will add AWSnycast
// to requests made by the AWS SDK.
var addAWSnycastToUserAgent = request.NamedHandler{
	Name: "AWSNycast.AWSnycastUserAgentHandler",
	Fn:   request.MakeAddToUserAgentHandler("AWSnycast", version.Version),
}
