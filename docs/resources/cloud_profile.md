---
page_title: "scalegrid_cloud_profile Resource - terraform-provider-scalegrid"
description: |-
  Manages an AWS ScaleGrid cloud profile.
---

# scalegrid_cloud_profile (Resource)

Manages an AWS (EC2/VPC) ScaleGrid cloud profile used to provision clusters in
your own AWS account. Azure profiles require an interactive permission-granting
script and are not supported by this resource.

## Example Usage

```terraform
resource "scalegrid_cloud_profile" "aws" {
  database            = "mongodb"
  name                = "aws-use1-a"
  region              = "us-east-1"
  access_key          = var.aws_access_key
  secret_key          = var.aws_secret_key
  vpc_id              = "vpc-0123456789abcdef0"
  subnet_id           = "subnet-0123456789abcdef0"
  vpc_cidr            = "10.0.0.0/16"
  subnet_cidr         = "10.0.1.0/24"
  security_group_id   = "sg-0123456789abcdef0"
  security_group_name = "scalegrid-mongo"
}
```

## Schema

### Required

- `database` (String) Engine the profile is for. Forces replacement.
- `name` (String) Unique profile name. Forces replacement.
- `region` (String) AWS region. Forces replacement.
- `access_key` (String) AWS access key. Updatable (rotates keys).
- `secret_key` (String, Sensitive) AWS secret key. Updatable (rotates keys).
- `vpc_id`, `subnet_id`, `vpc_cidr`, `subnet_cidr`, `security_group_id`, `security_group_name` (String) AWS networking details. Force replacement.

### Optional

- `connectivity_config` (String) `INTERNET` (default), `INTRANET`, `SECURITYGROUP`, or `CUSTOMIPRANGE`. Forces replacement.
- `enable_ssh` (Boolean) Enable SSH access. Default `false`. Forces replacement.

### Read-Only

- `id` (String) Machine pool ID.
- `cloud_type` (String) Cloud provider (e.g. AWS).

~> **Import is not supported.** The `secret_key` is sensitive and never returned
by the API, so an imported cloud profile cannot be reconciled without a
replacement diff. Define cloud profiles in configuration instead.
