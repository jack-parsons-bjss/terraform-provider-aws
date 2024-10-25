---
subcategory: "Lightsail"
layout: "aws"
page_title: "AWS: aws_lightsail_bucket_resource_access"
description: |-
  Provides a lightsail resource access to a bucket.
---


<!-- Please do not edit this file, it is generated. -->
# Resource: aws_lightsail_bucket_resource_access

Provides a lightsail resource access to a bucket.

## Example Usage

```python
# DO NOT EDIT. Code generated by 'cdktf convert' - Please report bugs at https://cdk.tf/bug
from constructs import Construct
from cdktf import Token, TerraformStack
#
# Provider bindings are generated by running `cdktf get`.
# See https://cdk.tf/provider-generation for more details.
#
from imports.aws.lightsail_bucket import LightsailBucket
from imports.aws.lightsail_bucket_resource_access import LightsailBucketResourceAccess
from imports.aws.lightsail_instance import LightsailInstance
class MyConvertedCode(TerraformStack):
    def __init__(self, scope, name):
        super().__init__(scope, name)
        LightsailBucket(self, "test",
            bundle_id="small_1_0",
            name="mytestbucket"
        )
        aws_lightsail_bucket_resource_access_test =
        LightsailBucketResourceAccess(self, "test_1",
            bucket_name=Token.as_string(aws_lightsail_bucket_resource_access_test.id),
            resource_name=id
        )
        # This allows the Terraform resource name to match the original name. You can remove the call if you don't need them to match.
        aws_lightsail_bucket_resource_access_test.override_logical_id("test")
        aws_lightsail_instance_test = LightsailInstance(self, "test_2",
            availability_zone="us-east-1b",
            blueprint_id="amazon_linux_2",
            bundle_id="nano_3_0",
            name="mytestinstance"
        )
        # This allows the Terraform resource name to match the original name. You can remove the call if you don't need them to match.
        aws_lightsail_instance_test.override_logical_id("test")
```

## Argument Reference

This resource supports the following arguments:

* `bucket_name` - (Required) The name of the bucket to grant access to.
* `resource_name` - (Required) The name of the resource to be granted bucket access.

## Attribute Reference

This resource exports the following attributes in addition to the arguments above:

* `id` - A combination of attributes separated by a `,` to create a unique id: `bucket_name`,`resource_name`

## Import

In Terraform v1.5.0 and later, use an [`import` block](https://developer.hashicorp.com/terraform/language/import) to import `aws_lightsail_bucket_resource_access` using the `id` attribute. For example:

```python
# DO NOT EDIT. Code generated by 'cdktf convert' - Please report bugs at https://cdk.tf/bug
from constructs import Construct
from cdktf import TerraformStack
#
# Provider bindings are generated by running `cdktf get`.
# See https://cdk.tf/provider-generation for more details.
#
from imports.aws.lightsail_bucket_resource_access import LightsailBucketResourceAccess
class MyConvertedCode(TerraformStack):
    def __init__(self, scope, name):
        super().__init__(scope, name)
        LightsailBucketResourceAccess.generate_config_for_import(self, "test", "example-bucket,example-instance")
```

Using `terraform import`, import `aws_lightsail_bucket_resource_access` using the `id` attribute. For example:

```console
% terraform import aws_lightsail_bucket_resource_access.test example-bucket,example-instance
```

<!-- cache-key: cdktf-0.20.8 input-11d161bb3c5191460b3ab1be6d53678d62712049e854a8f1218a3be8610b0749 -->