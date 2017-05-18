package aws

import (
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go/service/ec2"
)

type RouteTableFilter interface {
	Keep(*ec2.RouteTable) bool
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

// FIXME weird function
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

type RouteTableFilterTagRegexMatch struct {
	Key    string
	Regexp *regexp.Regexp
}

func (fs RouteTableFilterTagRegexMatch) Keep(rt *ec2.RouteTable) bool {
	for _, t := range rt.Tags {
		if *(t.Key) == fs.Key && fs.Regexp.MatchString(*(t.Value)) {
			return true
		}
	}
	return false
}
