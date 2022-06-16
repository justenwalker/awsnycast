package config

import (
	"context"
	"errors"
	"fmt"

	ec2type "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/hashicorp/go-multierror"
	log "github.com/sirupsen/logrus"

	"github.com/justenwalker/awsnycast/aws"
	"github.com/justenwalker/awsnycast/healthcheck"
	"github.com/justenwalker/awsnycast/instancemetadata"
)

type RouteTable struct {
	Name           string                  `yaml:"-"`
	Find           RouteTableFindSpec      `yaml:"find"`
	ManageRoutes   []*aws.ManageRoutesSpec `yaml:"manage_routes"`
	ec2RouteTables []ec2type.RouteTable
}

func (r *RouteTable) UpdateEc2RouteTables(ctx context.Context, rt []ec2type.RouteTable) error {
	filter, err := r.Find.GetFilter()
	if err != nil {
		return err
	}
	r.ec2RouteTables = aws.FilterRouteTables(filter, rt)
	if len(r.ec2RouteTables) == 0 {
		if r.Find.NoResultsOk {
			return nil
		}
		return errors.New(fmt.Sprintf("No route table in AWS matched filter spec in route table '%s'", r.Name))
	}
	for _, manage := range r.ManageRoutes {
		manage.UpdateEc2RouteTables(ctx, r.ec2RouteTables)
	}
	return nil
}

func (r *RouteTable) RunEc2Updates(ctx context.Context, manager aws.RouteTableManager, noop bool) error {
	for _, rtb := range r.ec2RouteTables {
		contextLogger := log.WithFields(log.Fields{
			"rtb": *(rtb.RouteTableId),
		})
		contextLogger.Debug("Finder found route table")
		for _, manageRoute := range r.ManageRoutes {
			contextLogger.WithFields(log.Fields{"cidr": manageRoute.Cidr}).Debug("Trying to manage route")
			if err := manager.ManageInstanceRoute(ctx, rtb, *manageRoute, noop); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *RouteTable) Validate(meta instancemetadata.InstanceMetadata, manager aws.RouteTableManager, name string, healthchecks map[string]*healthcheck.Healthcheck, remotehealthchecks map[string]*healthcheck.Healthcheck) error {
	r.Name = name
	if r.ManageRoutes == nil {
		r.ManageRoutes = make([]*aws.ManageRoutesSpec, 0)
	}
	var result *multierror.Error
	if len(r.ManageRoutes) == 0 {
		result = multierror.Append(result, errors.New(fmt.Sprintf("No manage_routes key in route table '%s'", r.Name)))
	}
	if err := r.Find.Validate(name); err != nil {
		result = multierror.Append(result, err)
	}
	if r.ec2RouteTables == nil {
		r.ec2RouteTables = make([]ec2type.RouteTable, 0)
	}
	for _, v := range r.ManageRoutes {
		if err := v.Validate(meta, manager, name, healthchecks, remotehealthchecks); err != nil {
			result = multierror.Append(result, err)
		}
	}

	return result.ErrorOrNil()
}
