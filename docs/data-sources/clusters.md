---
page_title: "scalegrid_clusters Data Source - terraform-provider-scalegrid"
description: |-
  Lists all ScaleGrid clusters of a given database engine.
---

# scalegrid_clusters (Data Source)

Lists all ScaleGrid clusters of a given database engine.

## Example Usage

```terraform
data "scalegrid_clusters" "postgres" {
  database = "postgresql"
}
```

## Schema

### Required

- `database` (String) Engine: `mongodb`, `redis`, `mysql`, or `postgresql`.

### Read-Only

- `clusters` (List of Object) Each element exposes `id`, `name`, `status`,
  `size`, `version`, `cluster_type`, `disk_size_gb`, `ssl_enabled`, and
  `encryption_enabled`.
