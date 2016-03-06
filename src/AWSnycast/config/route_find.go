package config

import (
	"AWSnycast/aws"
	"errors"
	"fmt"
	"github.com/hashicorp/go-multierror"
	"gopkg.in/yaml.v2"
)

type RouteTableFindSpec struct {
	NoResultsOk bool                   `yaml:"no_results_ok"`
	Type        string                 `yaml:"type"`
	Not         bool                   `yaml:"not"`
	Config      map[string]interface{} `yaml:"config"`
}

var routeFindTypes map[string]func(RouteTableFindSpec) (aws.RouteTableFilter, error)

func init() {
	routeFindTypes = make(map[string]func(RouteTableFindSpec) (aws.RouteTableFilter, error))
	routeFindTypes["by_tag"] = func(spec RouteTableFindSpec) (aws.RouteTableFilter, error) {
		var result *multierror.Error
		var key string
		var value string
		if v, ok := spec.Config["key"]; !ok {
			result = multierror.Append(result, errors.New("No key in config for by_tag route table finder"))
		} else {
			key = v.(string)
		}
		if v, ok := spec.Config["value"]; !ok {
			result = multierror.Append(result, errors.New("No value in config for by_tag route table finder"))
		} else {
			value = v.(string)
		}
		if err := result.ErrorOrNil(); err != nil {
			return nil, err
		}
		return aws.RouteTableFilterTagMatch{
			Key:   key,
			Value: value,
		}, nil
	}
	routeFindTypes["and"] = func(spec RouteTableFindSpec) (aws.RouteTableFilter, error) {
		filters, err := getFiltersListForSpec(spec)
		if err != nil {
			return nil, appendMultiError(err, "for and route table finder")
		}
		return aws.RouteTableFilterAnd{filters}, nil
	}
	routeFindTypes["or"] = func(spec RouteTableFindSpec) (aws.RouteTableFilter, error) {
		filters, err := getFiltersListForSpec(spec)
		if err != nil {
			return nil, appendMultiError(err, "for or route table finder")
		}
		return aws.RouteTableFilterOr{filters}, nil
	}
	routeFindTypes["main"] = func(spec RouteTableFindSpec) (aws.RouteTableFilter, error) {
		return aws.RouteTableFilterMain{}, nil
	}
	routeFindTypes["subnet"] = func(spec RouteTableFindSpec) (aws.RouteTableFilter, error) {
		if _, ok := spec.Config["subnet_id"]; !ok {
			return nil, errors.New("No subnet_id in config for subnet route table finder")
		}
		return aws.RouteTableFilterSubnet{spec.Config["subnet_id"].(string)}, nil
	}
	routeFindTypes["has_route_to"] = func(spec RouteTableFindSpec) (aws.RouteTableFilter, error) {
		if _, ok := spec.Config["cidr"]; !ok {
			return nil, errors.New("No cidr in config for has_route_to route table finder")
		}
		return aws.RouteTableFilterDestinationCidrBlock{DestinationCidrBlock: spec.Config["cidr"].(string)}, nil
	}
}

func appendMultiError(in *multierror.Error, a string) *multierror.Error {
	var result *multierror.Error
	for _, element := range in.Errors {
		result = multierror.Append(result, errors.New(element.Error()+" "+a))
	}
	return result
}

func getFiltersListForSpec(spec RouteTableFindSpec) ([]aws.RouteTableFilter, *multierror.Error) {
	var result *multierror.Error
	v, ok := spec.Config["filters"]
	if !ok {
		result = multierror.Append(errors.New("No filters in config"))
		return nil, result
	}
	var filters []aws.RouteTableFilter
	switch t := v.(type) {
	default:
		result = multierror.Append(result, errors.New(fmt.Sprintf("unexpected type %T for 'filters' key", t)))
	case []interface{}:
		for _, filter := range t { // I REGRET NOTHING
			filterRepacked, err := yaml.Marshal(filter)
			if err != nil {
				result = multierror.Append(result, err)
				continue
			}
			var spec RouteTableFindSpec
			err = yaml.Unmarshal(filterRepacked, &spec)
			if err != nil {
				result = multierror.Append(result, err)
				continue
			}
			filter, err := spec.GetFilter()
			if err != nil {
				result = multierror.Append(result, err)
				continue
			}
			filters = append(filters, filter)
		} // End lack of regret
	}
	return filters, result
}

func (spec RouteTableFindSpec) GetFilter() (aws.RouteTableFilter, error) {
	if genFilter, found := routeFindTypes[spec.Type]; found {
		filter, err := genFilter(spec)
		if err != nil {
			return filter, err
		}
		if spec.Not {
			return aws.RouteTableFilterNot{filter}, nil
		}
		return filter, nil
	}
	return nil, errors.New(fmt.Sprintf("Route table finder type '%s' not found in the registry", spec.Type))
}

func (r *RouteTableFindSpec) Validate(name string) error {
	var result *multierror.Error
	if r.Config == nil {
		result = multierror.Append(result, errors.New(fmt.Sprintf("Route find spec %s needs config", name)))
		r.Config = make(map[string]interface{})
	}
	if r.Type == "" {
		result = multierror.Append(result, errors.New(fmt.Sprintf("Route find spec %s needs a type key", name)))
	} else {
		if _, ok := routeFindTypes[r.Type]; !ok {
			result = multierror.Append(result, errors.New(fmt.Sprintf("Route find spec %s type '%s' not known", name, r.Type)))
		}
	}
	return result.ErrorOrNil()
}
