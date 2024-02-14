// Code generated by internal/generate/servicepackages/main.go; DO NOT EDIT.

package redshiftserverless

import (
	"context"

	aws_sdkv1 "github.com/aws/aws-sdk-go/aws"
	session_sdkv1 "github.com/aws/aws-sdk-go/aws/session"
	redshiftserverless_sdkv1 "github.com/aws/aws-sdk-go/service/redshiftserverless"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	"github.com/hashicorp/terraform-provider-aws/internal/types"
	"github.com/hashicorp/terraform-provider-aws/names"
)

type servicePackage struct{}

func (p *servicePackage) FrameworkDataSources(ctx context.Context) []*types.ServicePackageFrameworkDataSource {
	return []*types.ServicePackageFrameworkDataSource{}
}

func (p *servicePackage) FrameworkResources(ctx context.Context) []*types.ServicePackageFrameworkResource {
	return []*types.ServicePackageFrameworkResource{}
}

func (p *servicePackage) SDKDataSources(ctx context.Context) []*types.ServicePackageSDKDataSource {
	return []*types.ServicePackageSDKDataSource{
		{
			Factory:  DataSourceCredentials,
			TypeName: "aws_redshiftserverless_credentials",
		},
		{
			Factory:  DataSourceNamespace,
			TypeName: "aws_redshiftserverless_namespace",
		},
		{
			Factory:  DataSourceWorkgroup,
			TypeName: "aws_redshiftserverless_workgroup",
		},
	}
}

func (p *servicePackage) SDKResources(ctx context.Context) []*types.ServicePackageSDKResource {
	return []*types.ServicePackageSDKResource{
		{
			Factory:  ResourceEndpointAccess,
			TypeName: "aws_redshiftserverless_endpoint_access",
			Name:     "Endpoint Access",
		},
		{
			Factory:  ResourceNamespace,
			TypeName: "aws_redshiftserverless_namespace",
			Name:     "Namespace",
			Tags: &types.ServicePackageResourceTags{
				IdentifierAttribute: "arn",
			},
		},
		{
			Factory:  ResourceResourcePolicy,
			TypeName: "aws_redshiftserverless_resource_policy",
		},
		{
			Factory:  ResourceSnapshot,
			TypeName: "aws_redshiftserverless_snapshot",
		},
		{
			Factory:  ResourceUsageLimit,
			TypeName: "aws_redshiftserverless_usage_limit",
		},
		{
			Factory:  ResourceWorkgroup,
			TypeName: "aws_redshiftserverless_workgroup",
			Name:     "Workgroup",
			Tags: &types.ServicePackageResourceTags{
				IdentifierAttribute: "arn",
			},
		},
	}
}

func (p *servicePackage) ServicePackageName() string {
	return names.RedshiftServerless
}

// NewConn returns a new AWS SDK for Go v1 client for this service package's AWS API.
func (p *servicePackage) NewConn(ctx context.Context, config map[string]any) (*redshiftserverless_sdkv1.RedshiftServerless, error) {
	sess := config["session"].(*session_sdkv1.Session)

	return redshiftserverless_sdkv1.New(sess.Copy(&aws_sdkv1.Config{Endpoint: aws_sdkv1.String(config["endpoint"].(string))})), nil
}

func ServicePackage(ctx context.Context) conns.ServicePackage {
	return &servicePackage{}
}
