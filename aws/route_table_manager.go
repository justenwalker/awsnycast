package aws

import (
	"context"
	"errors"
	"os/exec"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"

	ec2type "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	log "github.com/sirupsen/logrus"
)

var errNICNotFound = errors.New("nic with source dest check disabled was not found")

type EC2API interface {
	CreateRoute(context.Context, *ec2.CreateRouteInput, ...func(*ec2.Options)) (*ec2.CreateRouteOutput, error)
	ReplaceRoute(context.Context, *ec2.ReplaceRouteInput, ...func(*ec2.Options)) (*ec2.ReplaceRouteOutput, error)
	DescribeRouteTables(context.Context, *ec2.DescribeRouteTablesInput, ...func(*ec2.Options)) (*ec2.DescribeRouteTablesOutput, error)
	DeleteRoute(context.Context, *ec2.DeleteRouteInput, ...func(*ec2.Options)) (*ec2.DeleteRouteOutput, error)
	DescribeNetworkInterfaces(context.Context, *ec2.DescribeNetworkInterfacesInput, ...func(*ec2.Options)) (*ec2.DescribeNetworkInterfacesOutput, error)
	DescribeInstanceAttribute(context.Context, *ec2.DescribeInstanceAttributeInput, ...func(*ec2.Options)) (*ec2.DescribeInstanceAttributeOutput, error)
	DescribeInstanceStatus(context.Context, *ec2.DescribeInstanceStatusInput, ...func(*ec2.Options)) (*ec2.DescribeInstanceStatusOutput, error)
}

type RouteTableManager interface {
	GetRouteTables(ctx context.Context) ([]ec2type.RouteTable, error)
	ManageInstanceRoute(context.Context, ec2type.RouteTable, ManageRoutesSpec, bool) error
	InstanceIsRouter(context.Context, string) bool
}

type RouteTableManagerEC2 struct {
	Region                 string
	conn                   EC2API
	srcdstcheckForInstance map[string]bool
}

func NewRouteTableManagerEC2(cfg aws.Config) *RouteTableManagerEC2 {
	r := RouteTableManagerEC2{
		srcdstcheckForInstance: map[string]bool{},
	}
	r.conn = ec2.NewFromConfig(cfg)
	return &r
}

// InstanceIsRouter when source destination check is disabled on any interface.
func (r RouteTableManagerEC2) InstanceIsRouter(ctx context.Context, instanceID string) bool {
	if v, ok := r.srcdstcheckForInstance[instanceID]; ok {
		return v
	}

	if _, err := r.routerInterface(ctx, instanceID); err != nil {
		switch err {
		case errNICNotFound:
			return false
		default:
			panic(err)
		}
	}

	r.srcdstcheckForInstance[instanceID] = true
	return true
}

func (r RouteTableManagerEC2) routerInterface(ctx context.Context, instanceID string) (nicID string, err error) {
	out, err := r.conn.DescribeNetworkInterfaces(ctx, &ec2.DescribeNetworkInterfacesInput{
		Filters: []ec2type.Filter{
			{Name: aws.String("attachment.instance-id"), Values: []string{instanceID}},
		},
	})
	if err != nil {
		return "", err
	}

	// Search all interfaces for a disabled source check.
	for _, nic := range out.NetworkInterfaces {
		if !*nic.SourceDestCheck {
			return *nic.NetworkInterfaceId, nil
		}
	}

	return "", errNICNotFound
}

func (r RouteTableManagerEC2) ManageInstanceRoute(ctx context.Context, rtb ec2type.RouteTable, rs ManageRoutesSpec, noop bool) error {
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
					if err := r.DeleteInstanceRoute(ctx, rtb.RouteTableId, *route, rs.Cidr, rs.Instance, noop); err != nil {
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
			}
			contextLogger.Debug("Not routed by my instance - evaluate for replacement")
		}

		if err := r.ReplaceInstanceRoute(ctx, rtb.RouteTableId, *route, rs, noop); err != nil {
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
	if _, err := r.conn.CreateRoute(ctx, &opts); err != nil {
		return err
	}
	return nil
}

func findRouteFromRouteTable(rtb ec2type.RouteTable, cidr string) *ec2type.Route {
	for _, route := range rtb.Routes {
		if route.DestinationCidrBlock != nil && *(route.DestinationCidrBlock) == cidr {
			return &route
		}
	}
	return nil
}

func (r RouteTableManagerEC2) DeleteInstanceRoute(ctx context.Context, routeTableId *string, route ec2type.Route, cidr string, instance string, noop bool) error {
	params := &ec2.DeleteRouteInput{
		DestinationCidrBlock: aws.String(cidr),
		RouteTableId:         routeTableId,
		DryRun:               aws.Bool(noop),
	}
	_, err := r.conn.DeleteRoute(ctx, params)
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

func (r RouteTableManagerEC2) checkRemoteHealthCheck(contextLogger *log.Entry, route ec2type.Route, rs ManageRoutesSpec) bool {
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

func (r RouteTableManagerEC2) ReplaceInstanceRoute(ctx context.Context, routeTableId *string, route ec2type.Route, rs ManageRoutesSpec, noop bool) error {
	cidr := rs.Cidr
	instance := rs.Instance
	ifUnhealthy := rs.IfUnhealthy
	contextLogger := log.WithFields(log.Fields{
		"cidr":                cidr,
		"rtb":                 *routeTableId,
		"instance_id":         instance,
		"current_route_state": route.State,
	})
	if route.InstanceId != nil {
		contextLogger = contextLogger.WithFields(log.Fields{"current_instance_id": *(route.InstanceId)})
	}
	if ifUnhealthy {
		if route.State == ec2type.RouteStateActive {
			if rs.RemoteHealthcheckName != "" {
				if !r.checkRemoteHealthCheck(contextLogger, route, rs) {
					return nil
				}
			}
			o, err := r.conn.DescribeInstanceStatus(ctx, &ec2.DescribeInstanceStatusInput{
				IncludeAllInstances: aws.Bool(false),
				InstanceIds:         []string{*(route.InstanceId)},
			})
			if err != nil {
				contextLogger.WithFields(log.Fields{"err": err.Error()}).Error("Error trying to DescribeInstanceStatus, not replacing route")
				return nil
			}
			if len(o.InstanceStatuses) == 1 {
				is := o.InstanceStatuses[0]
				instanceHealthOK := true
				if is.InstanceStatus.Status == ec2type.SummaryStatusImpaired {
					instanceHealthOK = false
				}
				systemHealthOK := true
				if is.SystemStatus.Status == ec2type.SummaryStatusImpaired {
					systemHealthOK = false
				}
				contextLogger = contextLogger.WithFields(log.Fields{"instanceHealthOK": instanceHealthOK, "systemHealthOK": systemHealthOK})
				if instanceHealthOK && systemHealthOK {
					contextLogger.Info("Not replacing route, as current route is active and instance is healthy")
					return nil
				}
			} else {
				contextLogger.Error("Did not get 1 instance for DescribeInstanceStatus - assuming instance has been terminated")
			}
		} else {
			contextLogger.Info("Current route is not active - replacing")
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

	nicID, err := r.routerInterface(ctx, instance)
	if err != nil {
		if err != nil {
			contextLogger.WithFields(log.Fields{
				"err": err.Error(),
			}).Warn("Error replacing route")
			return err
		}
	}
	if _, err = r.conn.ReplaceRoute(ctx, &ec2.ReplaceRouteInput{
		DestinationCidrBlock: aws.String(cidr),
		RouteTableId:         routeTableId,
		NetworkInterfaceId:   aws.String(nicID),
		DryRun:               aws.Bool(noop),
	}); err != nil {
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

func (r RouteTableManagerEC2) GetRouteTables(ctx context.Context) ([]ec2type.RouteTable, error) {
	resp, err := r.conn.DescribeRouteTables(ctx, &ec2.DescribeRouteTablesInput{})
	if err != nil {
		log.WithFields(log.Fields{
			"err": err.Error(),
		}).Warn("Error on DescribeRouteTables")
		return []ec2type.RouteTable{}, err
	}
	return resp.RouteTables, nil
}

func getCreateRouteInput(rtb ec2type.RouteTable, cidr string, instance string, noop bool) ec2.CreateRouteInput {
	return ec2.CreateRouteInput{
		RouteTableId:         rtb.RouteTableId,
		DestinationCidrBlock: aws.String(cidr),
		InstanceId:           aws.String(instance),
		DryRun:               aws.Bool(noop),
	}
}
