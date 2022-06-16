package aws

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2type "github.com/aws/aws-sdk-go-v2/service/ec2/types"

	"github.com/stretchr/testify/assert"

	"github.com/justenwalker/awsnycast/healthcheck"
	"github.com/justenwalker/awsnycast/instancemetadata"
	"github.com/justenwalker/awsnycast/testhelpers"
)

var (
	rtb1 = ec2type.RouteTable{
		RouteTableId: aws.String("rtb-f0ea3b95"),
		Routes: []ec2type.Route{
			{
				DestinationCidrBlock: aws.String("172.17.16.0/22"),
				GatewayId:            aws.String("local"),
				Origin:               ec2type.RouteOriginCreateRouteTable,
				State:                ec2type.RouteStateActive,
			},
		},
		Tags: []ec2type.Tag{
			{
				Key:   aws.String("Name"),
				Value: aws.String("uswest1 devb private insecure"),
			}},
		VpcId: aws.String("vpc-9496cffc"),
	}

	rtb2 = ec2type.RouteTable{
		Associations: []ec2type.RouteTableAssociation{
			{
				Main:                    aws.Bool(true),
				RouteTableAssociationId: aws.String("rtbassoc-b1f025d4"),
				RouteTableId:            aws.String("rtb-9696cffe"),
			},
			{
				Main:                    aws.Bool(false),
				RouteTableAssociationId: aws.String("rtbassoc-85c1cbe7"),
				RouteTableId:            aws.String("rtb-9696cffe"),
				SubnetId:                aws.String("subnet-16b0e97e"),
			},
			{
				Main:                    aws.Bool(false),
				RouteTableAssociationId: aws.String("rtbassoc-ba8573df"),
				RouteTableId:            aws.String("rtb-9696cffe"),
				SubnetId:                aws.String("subnet-3fb0e957"),
			},
			{
				Main:                    aws.Bool(false),
				RouteTableAssociationId: aws.String("rtbassoc-84c1cbe6"),
				RouteTableId:            aws.String("rtb-9696cffe"),
				SubnetId:                aws.String("subnet-28b0e940"),
			},
			{
				Main:                    aws.Bool(false),
				RouteTableAssociationId: aws.String("rtbassoc-858573e0"),
				RouteTableId:            aws.String("rtb-9696cffe"),
				SubnetId:                aws.String("subnet-f3b0e99b"),
			},
		},
		PropagatingVgws: []ec2type.PropagatingVgw{
			{
				GatewayId: aws.String("vgw-d2396a97"),
			},
		},
		RouteTableId: aws.String("rtb-9696cffe"),
		Routes: []ec2type.Route{
			{
				DestinationCidrBlock: aws.String("10.55.35.43/32"),
				GatewayId:            aws.String("vgw-d2396a97"),
				Origin:               ec2type.RouteOriginCreateRoute,
				State:                ec2type.RouteStateActive,
			},
			{
				DestinationCidrBlock: aws.String("172.17.16.0/22"),
				GatewayId:            aws.String("local"),
				Origin:               ec2type.RouteOriginCreateRouteTable,
				State:                ec2type.RouteStateActive,
			},
			{
				DestinationCidrBlock: aws.String("10.0.0.0/8"),
				InstanceId:           aws.String("i-446b201b"),
				InstanceOwnerId:      aws.String("613514870339"),
				NetworkInterfaceId:   aws.String("eni-ea8a9cac"),
				Origin:               ec2type.RouteOriginCreateRoute,
				State:                ec2type.RouteStateActive,
			},
			{
				DestinationCidrBlock: aws.String("0.0.0.0/0"),
				InstanceId:           aws.String("i-605bd2aa"),
				InstanceOwnerId:      aws.String("613514870339"),
				NetworkInterfaceId:   aws.String("eni-09472250"),
				Origin:               ec2type.RouteOriginCreateRoute,
				State:                ec2type.RouteStateActive,
			},
			{
				DestinationCidrBlock: aws.String("10.0.0.0/8"),
				GatewayId:            aws.String("vgw-d2396a97"),
				Origin:               ec2type.RouteOriginEnableVgwRoutePropagation,
				State:                ec2type.RouteStateActive,
			},
		},
		Tags: []ec2type.Tag{
			{
				Key:   aws.String("Name"),
				Value: aws.String("uswest1 devb private"),
			}},
		VpcId: aws.String("vpc-9496cffc"),
	}

	rtb3 = ec2type.RouteTable{
		Associations: []ec2type.RouteTableAssociation{
			{
				Main:                    aws.Bool(false),
				RouteTableAssociationId: aws.String("rtbassoc-818573e4"),
				RouteTableId:            aws.String("rtb-019cab69"),
				SubnetId:                aws.String("subnet-37b0e95f"),
			},
			{
				Main:                    aws.Bool(false),
				RouteTableAssociationId: aws.String("rtbassoc-fd9cab95"),
				RouteTableId:            aws.String("rtb-019cab69"),
				SubnetId:                aws.String("subnet-44b0e92c"),
			},
		},
		PropagatingVgws: []ec2type.PropagatingVgw{
			{
				GatewayId: aws.String("vgw-d2396a97"),
			},
		},
		RouteTableId: aws.String("rtb-019cab69"),
		Routes: []ec2type.Route{
			{
				DestinationCidrBlock: aws.String("10.55.35.43/32"),
				GatewayId:            aws.String("vgw-d2396a97"),
				Origin:               ec2type.RouteOriginCreateRoute,
				State:                ec2type.RouteStateActive,
			},
			{
				DestinationCidrBlock: aws.String("172.17.16.0/22"),
				GatewayId:            aws.String("local"),
				Origin:               ec2type.RouteOriginCreateRouteTable,
				State:                ec2type.RouteStateActive,
			},
			{
				DestinationCidrBlock: aws.String("10.0.0.0/8"),
				InstanceId:           aws.String("i-446b201b"),
				InstanceOwnerId:      aws.String("613514870339"),
				NetworkInterfaceId:   aws.String("eni-ea8a9cac"),
				Origin:               ec2type.RouteOriginCreateRoute,
				State:                ec2type.RouteStateActive,
			},
			{
				DestinationCidrBlock: aws.String("0.0.0.0/0"),
				GatewayId:            aws.String("igw-9ab1e8f2"),
				Origin:               ec2type.RouteOriginCreateRoute,
				State:                ec2type.RouteStateActive,
			},
			{
				DestinationCidrBlock: aws.String("10.0.0.0/8"),
				GatewayId:            aws.String("vgw-d2396a97"),
				Origin:               ec2type.RouteOriginEnableVgwRoutePropagation,
				State:                ec2type.RouteStateActive,
			},
		},
		Tags: []ec2type.Tag{
			{
				Key:   aws.String("Name"),
				Value: aws.String("uswest1 devb public"),
			},
		},
		VpcId: aws.String("vpc-9496cffc"),
	}

	rtb4 = ec2type.RouteTable{
		RouteTableId: aws.String("rtb-f1ea3b94"),
		Routes: []ec2type.Route{
			{
				DestinationCidrBlock: aws.String("172.17.16.0/22"),
				GatewayId:            aws.String("local"),
				Origin:               ec2type.RouteOriginCreateRouteTable,
				State:                ec2type.RouteStateActive,
			},
			{
				DestinationCidrBlock: aws.String("0.0.0.0/0"),
				GatewayId:            aws.String("igw-9ab1e8f2"),
				Origin:               ec2type.RouteOriginCreateRoute,
				State:                ec2type.RouteStateActive,
			},
		},
		Tags: []ec2type.Tag{
			{
				Key:   aws.String("Name"),
				Value: aws.String("uswest1 devb public insecure"),
			},
		},
		VpcId: aws.String("vpc-9496cffc"),
	}

	rtb5 = ec2type.RouteTable{
		RouteTableId: aws.String("rtb-f0ea3b96"),
		Routes: []ec2type.Route{
			{
				DestinationCidrBlock: aws.String("0.0.0.0/0"),
				InstanceId:           aws.String("i-605bd2ab"),
				InstanceOwnerId:      aws.String("613514870339"),
				NetworkInterfaceId:   aws.String("eni-09472251"),
				Origin:               ec2type.RouteOriginCreateRoute,
				State:                ec2type.RouteStateBlackhole,
			},
		},
		VpcId: aws.String("vpc-9496cffc"),
	}
	emptyHealthchecks map[string]*healthcheck.Healthcheck
	im1               instancemetadata.InstanceMetadata
	im2               instancemetadata.InstanceMetadata
)

func init() {
	emptyHealthchecks = make(map[string]*healthcheck.Healthcheck)
	im1 = instancemetadata.InstanceMetadata{Instance: "i-1234"}
	im2 = instancemetadata.InstanceMetadata{Instance: "i-other"}
}

type FakeHealthCheck struct {
	isHealthy bool
}

func (h *FakeHealthCheck) IsHealthy() bool {
	return h.isHealthy
}

func (h *FakeHealthCheck) GetListener() <-chan bool {
	return make(chan bool)
}

func (h *FakeHealthCheck) CanPassYet() bool {
	return true
}

func NewFakeEC2Conn() *FakeEC2Conn {
	return &FakeEC2Conn{
		DescribeRouteTablesOutput: &ec2.DescribeRouteTablesOutput{
			RouteTables: make([]ec2type.RouteTable, 0),
		},
		DescribeNetworkInterfacesOutput: &ec2.DescribeNetworkInterfacesOutput{
			NetworkInterfaces: []ec2type.NetworkInterface{
				{NetworkInterfaceId: aws.String("foo"), SourceDestCheck: aws.Bool(true)},
				{NetworkInterfaceId: aws.String("bar"), SourceDestCheck: aws.Bool(false)},
			},
		},
	}
}

type FakeEC2Conn struct {
	CreateRouteOutput               *ec2.CreateRouteOutput
	CreateRouteError                error
	CreateRouteInput                *ec2.CreateRouteInput
	ReplaceRouteOutput              *ec2.ReplaceRouteOutput
	ReplaceRouteError               error
	ReplaceRouteInput               *ec2.ReplaceRouteInput
	DeleteRouteInput                *ec2.DeleteRouteInput
	DeleteRouteOutput               *ec2.DeleteRouteOutput
	DeleteRouteError                error
	DescribeRouteTablesInput        *ec2.DescribeRouteTablesInput
	DescribeRouteTablesOutput       *ec2.DescribeRouteTablesOutput
	DescribeRouteTablesError        error
	DescribeInstanceAttributeInput  *ec2.DescribeInstanceAttributeInput
	DescribeInstanceAttributeOutput *ec2.DescribeInstanceAttributeOutput
	DescribeInstanceAttributError   error
	DescribeNetworkInterfacesOutput *ec2.DescribeNetworkInterfacesOutput
}

func (f *FakeEC2Conn) DescribeInstanceAttribute(ctx context.Context, i *ec2.DescribeInstanceAttributeInput, opts ...func(options *ec2.Options)) (*ec2.DescribeInstanceAttributeOutput, error) {
	f.DescribeInstanceAttributeInput = i
	return f.DescribeInstanceAttributeOutput, f.DescribeInstanceAttributError
}

func (f *FakeEC2Conn) CreateRoute(ctx context.Context, i *ec2.CreateRouteInput, opts ...func(options *ec2.Options)) (*ec2.CreateRouteOutput, error) {
	f.CreateRouteInput = i
	return f.CreateRouteOutput, f.CreateRouteError
}
func (f *FakeEC2Conn) ReplaceRoute(ctx context.Context, i *ec2.ReplaceRouteInput, opts ...func(options *ec2.Options)) (*ec2.ReplaceRouteOutput, error) {
	f.ReplaceRouteInput = i
	return f.ReplaceRouteOutput, f.ReplaceRouteError
}
func (f *FakeEC2Conn) DeleteRoute(ctx context.Context, i *ec2.DeleteRouteInput, opts ...func(options *ec2.Options)) (*ec2.DeleteRouteOutput, error) {
	f.DeleteRouteInput = i
	return f.DeleteRouteOutput, f.DeleteRouteError
}
func (f *FakeEC2Conn) DescribeRouteTables(ctx context.Context, i *ec2.DescribeRouteTablesInput, opts ...func(options *ec2.Options)) (*ec2.DescribeRouteTablesOutput, error) {
	f.DescribeRouteTablesInput = i
	return f.DescribeRouteTablesOutput, f.DescribeRouteTablesError
}
func (f *FakeEC2Conn) DescribeNetworkInterfaces(context.Context, *ec2.DescribeNetworkInterfacesInput, ...func(options *ec2.Options)) (*ec2.DescribeNetworkInterfacesOutput, error) {
	return f.DescribeNetworkInterfacesOutput, nil
}
func (f *FakeEC2Conn) DescribeInstanceStatus(context.Context, *ec2.DescribeInstanceStatusInput, ...func(options *ec2.Options)) (*ec2.DescribeInstanceStatusOutput, error) {
	return &ec2.DescribeInstanceStatusOutput{
		InstanceStatuses: []ec2type.InstanceStatus{
			{
				InstanceStatus: &ec2type.InstanceStatusSummary{Status: ec2type.SummaryStatusOk},
				SystemStatus:   &ec2type.InstanceStatusSummary{Status: ec2type.SummaryStatusOk},
			}},
	}, nil
}

func TestMetaDataFetcher(t *testing.T) {
	_ = NewMetadataFetcher(false)
	_ = NewMetadataFetcher(true)
}

type FakeRouteTableManager struct {
	RouteTable       *ec2type.RouteTable
	ManageRoutesSpec *ManageRoutesSpec
	Noop             bool
	Error            error
	Routes           []ec2type.RouteTable
}

func (r *FakeRouteTableManager) InstanceIsRouter(ctx context.Context, id string) bool {
	return true
}

func (r *FakeRouteTableManager) GetRouteTables(ctx context.Context) ([]ec2type.RouteTable, error) {
	return r.Routes, r.Error
}

func (r *FakeRouteTableManager) ManageInstanceRoute(ctx context.Context, rtb ec2type.RouteTable, rs ManageRoutesSpec, noop bool) error {
	r.RouteTable = &rtb
	r.ManageRoutesSpec = &rs
	r.Noop = noop
	return r.Error
}

func TestInstanceIsRouter(t *testing.T) {
	ctx := context.Background()
	conn := NewFakeEC2Conn()
	conn.DescribeNetworkInterfacesOutput = &ec2.DescribeNetworkInterfacesOutput{
		NetworkInterfaces: []ec2type.NetworkInterface{
			{NetworkInterfaceId: aws.String("foo"), SourceDestCheck: aws.Bool(true)},
			{NetworkInterfaceId: aws.String("bar"), SourceDestCheck: aws.Bool(false)},
		},
	}
	rtf := RouteTableManagerEC2{conn: conn, srcdstcheckForInstance: map[string]bool{}}
	ans := rtf.InstanceIsRouter(ctx, "i-1234")
	assert.Equal(t, true, ans)

	// Check cached path
	ans = rtf.InstanceIsRouter(ctx, "i-1234")
	assert.Equal(t, true, ans)
}

func TestInstanceIsRouter2(t *testing.T) {
	ctx := context.Background()
	conn := NewFakeEC2Conn()
	conn.DescribeNetworkInterfacesOutput = &ec2.DescribeNetworkInterfacesOutput{
		NetworkInterfaces: []ec2type.NetworkInterface{
			{NetworkInterfaceId: aws.String("foo"), SourceDestCheck: aws.Bool(true)},
		},
	}
	rtf := RouteTableManagerEC2{conn: conn, srcdstcheckForInstance: map[string]bool{}}
	ans := rtf.InstanceIsRouter(ctx, "i-4567")
	assert.Equal(t, false, ans)

	// Check cached path
	ans = rtf.InstanceIsRouter(ctx, "i-4567")
	assert.Equal(t, false, ans)
}

func TestFakeFetcher(t *testing.T) {
	ctx := context.Background()
	var f RouteTableManager
	f = &FakeRouteTableManager{
		Routes: []ec2type.RouteTable{rtb1},
	}
	rtb, err := f.GetRouteTables(ctx)
	assert.Nil(t, err)
	assert.Equal(t, len(rtb), 1)
	assert.Equal(t, rtb[0], rtb1)
}

func TestRouteTableFilterAlways(t *testing.T) {
	f := RouteTableFilterAlways{}
	assert.Equal(t, f.Keep(rtb1), false)
}

func TestRouteTableFilterNever(t *testing.T) {
	f := RouteTableFilterNever{}
	assert.Equal(t, f.Keep(rtb1), true)
}

func TestRouteTableFilterNot(t *testing.T) {
	f := RouteTableFilterNot{Filter: RouteTableFilterAlways{}}
	assert.Equal(t, f.Keep(rtb1), true)
	f = RouteTableFilterNot{Filter: RouteTableFilterNever{}}
	assert.Equal(t, f.Keep(rtb1), false)
}

func TestRouteTableFilterAndTwoNever(t *testing.T) {
	f := RouteTableFilterAnd{
		RouteTableFilters: []RouteTableFilter{
			RouteTableFilterNever{},
			RouteTableFilterNever{},
		},
	}
	assert.Equal(t, f.Keep(rtb1), true)
}

func TestRouteTableFilterAndOneNever(t *testing.T) {
	f := RouteTableFilterAnd{
		RouteTableFilters: []RouteTableFilter{
			RouteTableFilterNever{},
			RouteTableFilterAlways{},
		},
	}
	assert.Equal(t, f.Keep(rtb1), false)
}

func TestRouteTableFilterOrOneNever(t *testing.T) {
	f := RouteTableFilterOr{
		RouteTableFilters: []RouteTableFilter{
			RouteTableFilterNever{},
			RouteTableFilterAlways{},
		},
	}
	assert.Equal(t, f.Keep(rtb1), true)
}

func TestRouteTableFilterOrOneNever2(t *testing.T) {
	f := RouteTableFilterOr{
		RouteTableFilters: []RouteTableFilter{
			RouteTableFilterAlways{},
			RouteTableFilterNever{},
		},
	}
	assert.Equal(t, f.Keep(rtb1), true)
}

func TestRouteTableFilterOrAlways(t *testing.T) {
	f := RouteTableFilterOr{
		RouteTableFilters: []RouteTableFilter{
			RouteTableFilterAlways{},
			RouteTableFilterAlways{},
		},
	}
	assert.Equal(t, f.Keep(rtb1), false)
}

func TestFilterRouteTables(t *testing.T) {
	rtb := FilterRouteTables(RouteTableFilterNever{}, []ec2type.RouteTable{rtb1})
	assert.Equal(t, len(rtb), 1)
	assert.Equal(t, rtb[0], rtb1)
}

func TestRouteTableFilterMain(t *testing.T) {
	f := RouteTableFilterMain{}
	assert.Equal(t, f.Keep(rtb1), false)
	assert.Equal(t, f.Keep(rtb2), true)
}

func TestRoutTableFilterSubnet(t *testing.T) {
	f := RouteTableFilterSubnet{
		SubnetId: "subnet-28b0e940",
	}
	assert.Equal(t, f.Keep(rtb1), false)
	assert.Equal(t, f.Keep(rtb2), true)
}

func TestRouteTableForSubnetExplicitAssociation(t *testing.T) {
	rt := RouteTableForSubnet("subnet-37b0e95f", []ec2type.RouteTable{rtb1, rtb2, rtb3, rtb4})
	if assert.NotNil(t, rt) {
		assert.Equal(t, rt, &rtb3)
	}
}

func TestRouteTableForSubnetDefaultMain(t *testing.T) {
	rt := RouteTableForSubnet("subnet-38b0e95f", []ec2type.RouteTable{rtb1, rtb2, rtb3, rtb4})
	if assert.NotNil(t, rt) {
		assert.Equal(t, rt, &rtb2)
	}
}

func TestRouteTableForSubnetNone(t *testing.T) {
	rt := RouteTableForSubnet("subnet-38b0e95f", []ec2type.RouteTable{rtb1, rtb3, rtb4})
	assert.Nil(t, rt)
}

func TestRouteTableFilterDestinationCidrBlock(t *testing.T) {
	f := RouteTableFilterDestinationCidrBlock{
		DestinationCidrBlock: "0.0.0.0/0",
	}
	assert.Equal(t, f.Keep(rtb1), false)
	assert.Equal(t, f.Keep(rtb2), true)
}

func TestRouteTableFilterDestinationCidrBlockViaIGW(t *testing.T) {
	f := RouteTableFilterDestinationCidrBlock{
		DestinationCidrBlock: "0.0.0.0/0",
		ViaIGW:               true,
	}
	assert.Equal(t, f.Keep(rtb2), false)
	assert.Equal(t, f.Keep(rtb4), true)
}

func TestRouteTableFilterDestinationCidrBlockViaInstance(t *testing.T) {
	f := RouteTableFilterDestinationCidrBlock{
		DestinationCidrBlock: "0.0.0.0/0",
		ViaInstance:          true,
	}
	/* Via IGW */
	assert.Equal(t, f.Keep(rtb4), false)
	/* Via instance */
	assert.Equal(t, f.Keep(rtb2), true)
}

func TestRouteTableFilterDestinationCidrBlockViaInstanceInactive(t *testing.T) {
	f := RouteTableFilterDestinationCidrBlock{
		DestinationCidrBlock: "0.0.0.0/0",
		ViaInstance:          true,
		InstanceNotActive:    true,
	}
	assert.Equal(t, f.Keep(rtb2), false)
	assert.Equal(t, f.Keep(rtb5), true)
}

func TestRouteTableFilterTagMatch(t *testing.T) {
	f := RouteTableFilterTagMatch{
		Key:   "Name",
		Value: "uswest1 devb private insecure",
	}
	assert.Equal(t, f.Keep(rtb2), false)
	assert.Equal(t, f.Keep(rtb1), true)
}

func TestRouteTableFilterTagRegexpMatch(t *testing.T) {
	f := RouteTableFilterTagRegexMatch{
		Key:    "Name",
		Regexp: regexp.MustCompile("private"),
	}
	assert.Equal(t, f.Keep(rtb1), true)
	assert.Equal(t, f.Keep(rtb2), true)

	f = RouteTableFilterTagRegexMatch{
		Key:    "Name",
		Regexp: regexp.MustCompile("insecure$"),
	}
	assert.Equal(t, f.Keep(rtb1), true)
	assert.Equal(t, f.Keep(rtb2), false)
}

func TestGetCreateRouteInput(t *testing.T) {
	rtb := ec2type.RouteTable{RouteTableId: aws.String("rtb-1234")}
	in := getCreateRouteInput(rtb, "0.0.0.0/0", "i-12345", false)
	assert.Equal(t, *(in.RouteTableId), "rtb-1234")
	assert.Equal(t, *(in.DestinationCidrBlock), "0.0.0.0/0")
	assert.Equal(t, *(in.InstanceId), "i-12345")
	assert.Equal(t, *(in.DryRun), false)
}

func TestGetCreateRouteInputDryRun(t *testing.T) {
	rtb := ec2type.RouteTable{RouteTableId: aws.String("rtb-1234")}
	in := getCreateRouteInput(rtb, "0.0.0.0/0", "i-12345", true)
	assert.Equal(t, *(in.DryRun), true)
}

func TestFindRouteFromRouteTableNoCidr(t *testing.T) {
	findRouteFromRouteTable(ec2type.RouteTable{
		RouteTableId: aws.String("rtb-f0ea3b95"),
		Routes: []ec2type.Route{
			{
				// Note no DestinationCidrBlock
				GatewayId: aws.String("local"),
				Origin:    ec2type.RouteOriginCreateRouteTable,
				State:     ec2type.RouteStateActive,
			},
		},
		Tags:  []ec2type.Tag{},
		VpcId: aws.String("vpc-9496cffc"),
	}, "0.0.0.0/0")
}

func TestRouteTableManagerEC2ReplaceInstanceRouteNoop(t *testing.T) {
	ctx := context.Background()
	rtf := RouteTableManagerEC2{conn: NewFakeEC2Conn()}
	route := findRouteFromRouteTable(rtb2, "0.0.0.0/0")
	if assert.NotNil(t, route) {
		rs := ManageRoutesSpec{Cidr: "0.0.0.0/0", Instance: "i-1234", IfUnhealthy: false}
		assert.Nil(t, rtf.ReplaceInstanceRoute(ctx, rtb2.RouteTableId, *route, rs, true))
		if assert.NotNil(t, rtf.conn.(*FakeEC2Conn).ReplaceRouteInput) {
			// Should *not* have actually tried to replace the route - dry run mode
			r := rtf.conn.(*FakeEC2Conn).ReplaceRouteInput
			assert.Equal(t, *(r.DryRun), true)
		}
	}
}

func TestRouteTableManagerEC2ReplaceInstanceRoute(t *testing.T) {
	ctx := context.Background()
	rtf := RouteTableManagerEC2{conn: NewFakeEC2Conn()}
	route := findRouteFromRouteTable(rtb2, "0.0.0.0/0")
	if assert.NotNil(t, route) {
		rs := ManageRoutesSpec{Cidr: "0.0.0.0/0", Instance: "i-1234", IfUnhealthy: false}
		if assert.Nil(t, rtf.ReplaceInstanceRoute(ctx, rtb2.RouteTableId, *route, rs, false)) {
			if assert.NotNil(t, rtf.conn.(*FakeEC2Conn).ReplaceRouteInput) {
				r := rtf.conn.(*FakeEC2Conn).ReplaceRouteInput
				assert.Equal(t, *(r.DestinationCidrBlock), "0.0.0.0/0")
				assert.Equal(t, *(r.RouteTableId), *(rtb2.RouteTableId))
				assert.Equal(t, *(r.NetworkInterfaceId), "bar")
			}
		}
	}
}

func TestRouteTableManagerEC2ReplaceInstanceRouteFails(t *testing.T) {
	ctx := context.Background()
	rtf := RouteTableManagerEC2{conn: NewFakeEC2Conn()}
	rtf.conn.(*FakeEC2Conn).ReplaceRouteError = errors.New("Whoops, AWS blew up")
	route := findRouteFromRouteTable(rtb2, "0.0.0.0/0")
	if assert.NotNil(t, route) {
		rs := ManageRoutesSpec{Cidr: "0.0.0.0/0", Instance: "i-1234", IfUnhealthy: false}
		err := rtf.ReplaceInstanceRoute(ctx, rtb2.RouteTableId, *route, rs, false)
		if assert.NotNil(t, err) {
			assert.Equal(t, err.Error(), "Whoops, AWS blew up")
			assert.NotNil(t, rtf.conn.(*FakeEC2Conn).ReplaceRouteInput)
		}
	}
}

func TestRouteTableManagerEC2ReplaceInstanceRouteNotIfHealthy(t *testing.T) {
	ctx := context.Background()
	rtf := RouteTableManagerEC2{conn: NewFakeEC2Conn()}
	route := findRouteFromRouteTable(rtb2, "0.0.0.0/0")
	if assert.NotNil(t, route) {
		rs := ManageRoutesSpec{Cidr: "0.0.0.0/0", Instance: "i-1234", IfUnhealthy: true}
		err := rtf.ReplaceInstanceRoute(ctx, rtb2.RouteTableId, *route, rs, false)
		assert.Nil(t, err)
		assert.Nil(t, rtf.conn.(*FakeEC2Conn).ReplaceRouteInput)
	}
}

func TestRouteTableManagerEC2ManageInstanceRouteAlreadyThisInstance(t *testing.T) {
	ctx := context.Background()
	rtf := RouteTableManagerEC2{conn: NewFakeEC2Conn()}
	s := ManageRoutesSpec{
		Cidr:        "0.0.0.0/0",
		Instance:    "i-605bd2aa",
		IfUnhealthy: false,
	}
	err := rtf.ManageInstanceRoute(ctx, rtb2, s, false)
	assert.Nil(t, err)
	assert.Nil(t, rtf.conn.(*FakeEC2Conn).ReplaceRouteInput)
}

func TestManageInstanceRoute(t *testing.T) {
	ctx := context.Background()

	rtf := RouteTableManagerEC2{conn: NewFakeEC2Conn()}
	s := ManageRoutesSpec{
		Cidr:        "0.0.0.0/0",
		Instance:    "i-1234",
		IfUnhealthy: false,
	}
	err := rtf.ManageInstanceRoute(ctx, rtb2, s, false)
	assert.Nil(t, err)
	if assert.NotNil(t, rtf.conn.(*FakeEC2Conn).ReplaceRouteInput) {
		r := rtf.conn.(*FakeEC2Conn).ReplaceRouteInput
		assert.Equal(t, *(r.DestinationCidrBlock), "0.0.0.0/0")
		assert.Equal(t, *(r.RouteTableId), *(rtb2.RouteTableId))
		assert.Equal(t, *(r.NetworkInterfaceId), "bar")
	}
}

func TestManageInstanceRouteAWSFailOnReplace(t *testing.T) {
	ctx := context.Background()
	rtf := RouteTableManagerEC2{conn: NewFakeEC2Conn()}
	rtf.conn.(*FakeEC2Conn).ReplaceRouteError = errors.New("Whoops, AWS blew up")
	s := ManageRoutesSpec{
		Cidr:        "0.0.0.0/0",
		Instance:    "i-1234",
		IfUnhealthy: false,
	}
	err := rtf.ManageInstanceRoute(ctx, rtb2, s, false)
	if assert.NotNil(t, err) {
		assert.Equal(t, err.Error(), "Whoops, AWS blew up")
	}
}

func TestManageInstanceRouteAWSFailOnCreate(t *testing.T) {
	ctx := context.Background()
	rtf := RouteTableManagerEC2{conn: NewFakeEC2Conn()}
	rtf.conn.(*FakeEC2Conn).CreateRouteError = errors.New("Whoops, AWS blew up")
	s := ManageRoutesSpec{
		Cidr:        "0.0.0.0/0",
		Instance:    "i-1234",
		IfUnhealthy: false,
	}
	err := rtf.ManageInstanceRoute(ctx, rtb1, s, false)
	if assert.NotNil(t, err) {
		assert.Equal(t, err.Error(), "Whoops, AWS blew up")
	}
}

func TestManageInstanceRouteCreateRoute(t *testing.T) {
	ctx := context.Background()
	rtf := RouteTableManagerEC2{conn: NewFakeEC2Conn()}
	s := ManageRoutesSpec{
		Cidr:        "0.0.0.0/0",
		Instance:    "i-1234",
		IfUnhealthy: false,
	}
	err := rtf.ManageInstanceRoute(ctx, rtb1, s, false)
	assert.Nil(t, err)
	if assert.NotNil(t, rtf.conn.(*FakeEC2Conn).CreateRouteInput) {
		in := rtf.conn.(*FakeEC2Conn).CreateRouteInput
		assert.Equal(t, *(in.RouteTableId), *(rtb1.RouteTableId))
		assert.Equal(t, *(in.DestinationCidrBlock), "0.0.0.0/0")
		assert.Equal(t, *(in.InstanceId), "i-1234")
	}
}

func TestGetRouteTables(t *testing.T) {
	ctx := context.Background()
	rtf := RouteTableManagerEC2{conn: NewFakeEC2Conn()}
	_, err := rtf.GetRouteTables(ctx)
	assert.Nil(t, err)
	assert.NotNil(t, rtf.conn.(*FakeEC2Conn).DescribeRouteTablesInput)
}

func TestGetRouteTablesAWSFail(t *testing.T) {
	ctx := context.Background()
	rtf := RouteTableManagerEC2{conn: NewFakeEC2Conn()}
	rtf.conn.(*FakeEC2Conn).DescribeRouteTablesError = errors.New("Whoops, AWS blew up")
	_, err := rtf.GetRouteTables(ctx)
	if assert.NotNil(t, err) {
		assert.Equal(t, err.Error(), "Whoops, AWS blew up")
	}
	assert.NotNil(t, rtf.conn.(*FakeEC2Conn).DescribeRouteTablesInput, "rtf.conn.(*FakeEC2Conn).DescribeRouteTablesInput was never called")
}

func TestNewRouteTableManager(t *testing.T) {
	assert.Nil(t, os.Setenv("AWS_ACCESS_KEY_ID", "AKIAIOSFODNN7EXAMPLE"))
	assert.Nil(t, os.Setenv("AWS_SECRET_ACCESS_KEY", "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"))
	cfg, err := config.LoadDefaultConfig(context.TODO())
	assert.NoError(t, err)
	rtf := NewRouteTableManagerEC2(cfg)
	if assert.NotNil(t, rtf) {
		assert.NotNil(t, rtf.conn)
	}
}

func TestManageRoutesSpecDefault(t *testing.T) {
	u := &ManageRoutesSpec{
		Cidr: "127.0.0.1",
	}
	err := u.Validate(im1, &FakeRouteTableManager{}, "foo", emptyHealthchecks, emptyHealthchecks)
	assert.Nil(t, err)
	assert.Equal(t, u.Cidr, "127.0.0.1/32", "Not canonicalized in ManageRoutesSpecDefault")
	assert.Equal(t, u.Instance, "i-1234", fmt.Sprintf("Instance not defaulted to SELF (i-1234), is '%s'", u.Instance))
	assert.NotNil(t, u.Manager)
}

func TestManageRoutesSpecValidateMissingCidr(t *testing.T) {
	r := ManageRoutesSpec{
		Instance: "SELF",
	}
	err := r.Validate(im1, &FakeRouteTableManager{}, "foo", emptyHealthchecks, emptyHealthchecks)
	testhelpers.CheckOneMultiError(t, err, "cidr is not defined in foo")
}

func TestManageRoutesSpecValidateBadCidr1(t *testing.T) {
	r := ManageRoutesSpec{
		Cidr:     "300.0.0.0/16",
		Instance: "SELF",
	}
	err := r.Validate(im1, &FakeRouteTableManager{}, "foo", emptyHealthchecks, emptyHealthchecks)
	testhelpers.CheckOneMultiError(t, err, "Could not parse invalid CIDR address: 300.0.0.0/16 in foo")
}

func TestManageRoutesSpecValidateBadCidr2(t *testing.T) {
	r := ManageRoutesSpec{
		Cidr:     "3.0.0.0/160",
		Instance: "SELF",
	}
	err := r.Validate(im1, &FakeRouteTableManager{}, "foo", emptyHealthchecks, emptyHealthchecks)
	testhelpers.CheckOneMultiError(t, err, "Could not parse invalid CIDR address: 3.0.0.0/160 in foo")
}

func TestManageRoutesSpecValidateBadCidr3(t *testing.T) {
	r := ManageRoutesSpec{
		Cidr:     "foo",
		Instance: "SELF",
	}
	err := r.Validate(im1, &FakeRouteTableManager{}, "bar", emptyHealthchecks, emptyHealthchecks)
	testhelpers.CheckOneMultiError(t, err, "Could not parse invalid CIDR address: foo/32 in bar")
}

func TestManageRoutesSpecValidate(t *testing.T) {
	r := ManageRoutesSpec{
		Cidr:     "0.0.0.0/0",
		Instance: "SELF",
	}
	assert.Nil(t, r.Validate(im1, &FakeRouteTableManager{}, "foo", emptyHealthchecks, emptyHealthchecks))
}

func TestManageRoutesSpecValidateMissingHealthcheck(t *testing.T) {
	r := ManageRoutesSpec{
		Cidr:            "0.0.0.0/0",
		Instance:        "SELF",
		HealthcheckName: "test",
	}
	err := r.Validate(im1, &FakeRouteTableManager{}, "foo", emptyHealthchecks, emptyHealthchecks)
	testhelpers.CheckOneMultiError(t, err, "Route tables foo, route 0.0.0.0/0 cannot find healthcheck 'test'")
}

func TestManageRoutesSpecValidateWithHealthcheck(t *testing.T) {
	r := ManageRoutesSpec{
		Cidr:            "0.0.0.0/0",
		Instance:        "SELF",
		HealthcheckName: "test",
	}
	h := make(map[string]*healthcheck.Healthcheck)
	h["test"] = &healthcheck.Healthcheck{}
	err := r.Validate(im1, &FakeRouteTableManager{}, "foo", h, emptyHealthchecks)
	assert.Nil(t, err)
	assert.Equal(t, h["test"], r.healthcheck, "r.healthcheck not set")
}

func TestManageRoutesSpecValidateMissingRemoteHealthcheck(t *testing.T) {
	r := ManageRoutesSpec{
		Cidr:                  "0.0.0.0/0",
		Instance:              "SELF",
		RemoteHealthcheckName: "test",
	}
	err := r.Validate(im1, &FakeRouteTableManager{}, "foo", emptyHealthchecks, emptyHealthchecks)
	testhelpers.CheckOneMultiError(t, err, "Route tables foo, route 0.0.0.0/0 cannot find remote healthcheck 'test'")
}

func TestManageRoutesSpecValidateWithRemoteHealthcheck(t *testing.T) {
	r := ManageRoutesSpec{
		Cidr:                  "0.0.0.0/0",
		Instance:              "SELF",
		RemoteHealthcheckName: "test",
	}
	h := make(map[string]*healthcheck.Healthcheck)
	h["test"] = &healthcheck.Healthcheck{}
	err := r.Validate(im1, &FakeRouteTableManager{}, "foo", emptyHealthchecks, h)
	assert.Nil(t, err)
	assert.Equal(t, h["test"], r.remotehealthchecktemplate, "r.temptehealthchecktemplate not set")
}

func TestManageRouteSpecStartHealthcheckListenerNoHealthcheck(t *testing.T) {
	urs := ManageRoutesSpec{
		Cidr:     "127.0.0.1",
		Instance: "SELF",
	}
	urs.StartHealthcheckListener(false)
}

func TestHandleHealthcheckResult(t *testing.T) {
	ctx := context.Background()

	urs := ManageRoutesSpec{
		Cidr:           "127.0.0.1",
		Instance:       "SELF",
		ec2RouteTables: []ec2type.RouteTable{rtb1},
		Manager:        &FakeRouteTableManager{},
	}
	urs.handleHealthcheckResult(ctx, true, false, true)
	assert.NotNil(t, urs.Manager.(*FakeRouteTableManager).RouteTable)
	assert.NotNil(t, urs.Manager.(*FakeRouteTableManager).ManageRoutesSpec)
	assert.Equal(t, urs.Manager.(*FakeRouteTableManager).Noop, true)
}

func TestHandleHealthcheckResultError(t *testing.T) {
	ctx := context.Background()
	urs := ManageRoutesSpec{
		Cidr:           "127.0.0.1",
		Instance:       "SELF",
		ec2RouteTables: []ec2type.RouteTable{rtb1},
		Manager:        &FakeRouteTableManager{Error: errors.New("Test")},
	}
	urs.handleHealthcheckResult(ctx, true, false, false)
}

func TestManageRouteSpecDefaultInstanceSELF(t *testing.T) {
	urs := ManageRoutesSpec{
		Cidr:     "127.0.0.1",
		Instance: "SELF",
	}
	urs.Validate(im1, &FakeRouteTableManager{}, "foo", emptyHealthchecks, emptyHealthchecks)
	assert.Equal(t, urs.Instance, "i-1234")
}

func TestManageRouteSpecDefaultInstanceOther(t *testing.T) {
	urs := ManageRoutesSpec{
		Cidr:     "127.0.0.1",
		Instance: "i-foo",
	}
	urs.Validate(im2, &FakeRouteTableManager{}, "foo", emptyHealthchecks, emptyHealthchecks)
	assert.Equal(t, urs.Instance, "i-foo")
}

func TestManageInstanceRouteNoCreateRouteBadHealthcheck(t *testing.T) {
	ctx := context.Background()
	rtf := RouteTableManagerEC2{conn: NewFakeEC2Conn()}
	s := ManageRoutesSpec{
		Cidr:            "0.0.0.0/0",
		Instance:        "i-1234",
		IfUnhealthy:     false,
		HealthcheckName: "foo",
		healthcheck:     &FakeHealthCheck{isHealthy: false},
	}
	err := rtf.ManageInstanceRoute(ctx, rtb1, s, false)
	assert.Nil(t, err)
	assert.Nil(t, rtf.conn.(*FakeEC2Conn).CreateRouteInput, "rtf.conn.(*FakeEC2Conn).CreateRoute was called")
}

func TestManageInstanceRouteCreateRouteGoodHealthcheck(t *testing.T) {
	ctx := context.Background()
	rtf := RouteTableManagerEC2{conn: NewFakeEC2Conn()}
	s := ManageRoutesSpec{
		Cidr:            "0.0.0.0/0",
		Instance:        "i-1234",
		IfUnhealthy:     false,
		HealthcheckName: "foo",
		healthcheck:     &FakeHealthCheck{isHealthy: true},
	}
	err := rtf.ManageInstanceRoute(ctx, rtb1, s, false)
	assert.Nil(t, err)
	assert.NotNil(t, rtf.conn.(*FakeEC2Conn).CreateRouteInput, "rtf.conn.(*FakeEC2Conn).CreateRoute was not called")
}

func TestManageInstanceRouteDeleteInstanceRouteThisInstanceUnhealthy(t *testing.T) {
	ctx := context.Background()
	rtf := RouteTableManagerEC2{conn: NewFakeEC2Conn()}
	s := ManageRoutesSpec{
		Cidr:            "0.0.0.0/0",
		Instance:        "i-605bd2aa",
		IfUnhealthy:     false,
		HealthcheckName: "localhealthcheck",
		healthcheck:     &FakeHealthCheck{isHealthy: false},
	}
	err := rtf.ManageInstanceRoute(ctx, rtb2, s, false)
	assert.Nil(t, err)
	assert.Nil(t, rtf.conn.(*FakeEC2Conn).ReplaceRouteInput, "ReplaceRouteInput was called")
	if assert.NotNil(t, rtf.conn.(*FakeEC2Conn).DeleteRouteInput, "DeleteRouteInput was never called") {
		r := rtf.conn.(*FakeEC2Conn).DeleteRouteInput
		assert.Equal(t, *(r.DestinationCidrBlock), "0.0.0.0/0")
		assert.Equal(t, *(r.RouteTableId), *(rtb2.RouteTableId))
	}
}

func TestManageInstanceRouteDeleteInstanceRouteThisInstanceUnhealthyNeverDelete(t *testing.T) {
	ctx := context.Background()
	rtf := RouteTableManagerEC2{conn: NewFakeEC2Conn()}
	s := ManageRoutesSpec{
		Cidr:            "0.0.0.0/0",
		Instance:        "i-605bd2aa",
		IfUnhealthy:     false,
		HealthcheckName: "localhealthcheck",
		healthcheck:     &FakeHealthCheck{isHealthy: false},
		NeverDelete:     true,
	}
	err := rtf.ManageInstanceRoute(ctx, rtb2, s, false)
	assert.Nil(t, err)
	assert.Nil(t, rtf.conn.(*FakeEC2Conn).ReplaceRouteInput, "ReplaceRouteInput was called")
	assert.Nil(t, rtf.conn.(*FakeEC2Conn).DeleteRouteInput, "DeleteRouteInput was called")
}

func TestManageInstanceRouteDeleteInstanceRouteThisInstanceUnhealthyAWSFail(t *testing.T) {
	ctx := context.Background()
	rtf := RouteTableManagerEC2{conn: NewFakeEC2Conn()}
	rtf.conn.(*FakeEC2Conn).DeleteRouteError = errors.New("Whoops, AWS blew up")
	s := ManageRoutesSpec{
		Cidr:            "0.0.0.0/0",
		Instance:        "i-605bd2aa",
		IfUnhealthy:     false,
		HealthcheckName: "localhealthcheck",
		healthcheck:     &FakeHealthCheck{isHealthy: false},
	}
	err := rtf.ManageInstanceRoute(ctx, rtb2, s, false)
	if assert.NotNil(t, err) {
		assert.Equal(t, err.Error(), "Whoops, AWS blew up")
	}
}

func TestEc2RouteTablesDefault(t *testing.T) {
	rs := &ManageRoutesSpec{
		Cidr: "127.0.0.1",
	}
	rs.Validate(im1, &FakeRouteTableManager{}, "foo", emptyHealthchecks, emptyHealthchecks)
	assert.NotNil(t, rs.ec2RouteTables)
}

func TestUpdateEc2RouteTables(t *testing.T) {
	ctx := context.Background()
	rs := &ManageRoutesSpec{}
	rs.UpdateEc2RouteTables(ctx, []ec2type.RouteTable{})
	assert.NotNil(t, rs.ec2RouteTables)
}

func TestStartHealthcheckListenerNoHealthcheck(t *testing.T) {
	rs := &ManageRoutesSpec{}
	rs.StartHealthcheckListener(false)
}

func TestUpdateRemoteHealthchecksEmpty(t *testing.T) {
	ctx := context.Background()
	rs := &ManageRoutesSpec{
		Cidr:                  "127.0.0.1",
		RemoteHealthcheckName: "test",
	}
	err := rs.Validate(im1, &FakeRouteTableManager{}, "foo", emptyHealthchecks, emptyHealthchecks)
	testhelpers.CheckOneMultiError(t, err, "Route tables foo, route 127.0.0.1/32 cannot find remote healthcheck 'test'")
	rs.UpdateRemoteHealthchecks(ctx)
}

func TestUpdateRemoteHealthchecksNoHealthcheck(t *testing.T) {
	ctx := context.Background()
	rt := make([]ec2type.RouteTable, 0)
	hc := make(map[string]*healthcheck.Healthcheck)
	hc["192.168.1.1"] = &healthcheck.Healthcheck{}
	rs := &ManageRoutesSpec{
		Cidr:                  "127.0.0.1",
		RemoteHealthcheckName: "test",
		ec2RouteTables:        rt,
	}
	rs.remotehealthchecks = hc
	templates := make(map[string]*healthcheck.Healthcheck)
	templates["test"] = &healthcheck.Healthcheck{}
	err := rs.Validate(im1, &FakeRouteTableManager{}, "foo", emptyHealthchecks, templates)
	assert.Nil(t, err)
	rs.UpdateRemoteHealthchecks(ctx)
	_, _ = hc["192.168.1.1"]
	//assert.Equal(t, ok, false, "Has been deleted")
}

func TestUpdateRemoteHealthchecks(t *testing.T) {
	ctx := context.Background()
	hc := make(map[string]*healthcheck.Healthcheck)
	hc["test"] = &healthcheck.Healthcheck{}
	rs := &ManageRoutesSpec{
		Cidr:                  "127.0.0.1",
		RemoteHealthcheckName: "test",
	}
	err := rs.Validate(im1, &FakeRouteTableManager{}, "foo", emptyHealthchecks, hc)
	assert.Nil(t, err)
	rs.ec2RouteTables = []ec2type.RouteTable{{}}
	rs.UpdateRemoteHealthchecks(ctx)
}
