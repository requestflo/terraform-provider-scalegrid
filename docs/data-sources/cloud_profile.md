---
page_title: "scalegrid_cloud_profile Data Source - terraform-provider-scalegrid"
description: |-
  Fetches a ScaleGrid cloud profile by ID or name.
---

# scalegrid_cloud_profile (Data Source)

Fetches a ScaleGrid cloud profile by ID or name.

## Example Usage

```terraform
data "scalegrid_cloud_profile" "aws" {
  name = "aws-use1-a"
}
```

## Schema

### Optional

- `id` (String) Machine pool ID. Either `id` or `name` must be set.
- `name` (String) Profile name. Either `id` or `name` must be set.

### Read-Only

- `cloud_type` (String) Cloud provider (e.g. AWS).
- `database` (String) Engine the profile is for.
- `status` (String) Profile status.
- `shared` (Boolean) Whether it is a shared (Dedicated plan) profile.
