---
page_title: "scalegrid_cluster Data Source - terraform-provider-scalegrid"
description: |-
  Fetches a single ScaleGrid cluster by ID or name.
---

# scalegrid_cluster (Data Source)

Fetches a single ScaleGrid cluster by ID or name.

## Example Usage

```terraform
data "scalegrid_cluster" "mongo" {
  database = "mongodb"
  name     = "production-mongo"
}
```

## Schema

### Required

- `database` (String) Engine: `mongodb`, `redis`, `mysql`, or `postgresql`.

### Optional

- `id` (String) Cluster ID. Either `id` or `name` must be set.
- `name` (String) Cluster name. Either `id` or `name` must be set.

### Read-Only

- `status` (String)
- `size` (String)
- `version` (String)
- `cluster_type` (String)
- `disk_size_gb` (Number)
- `ssl_enabled` (Boolean)
- `encryption_enabled` (Boolean)
