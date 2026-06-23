---
page_title: "scalegrid_backup Resource - terraform-provider-scalegrid"
description: |-
  Triggers and manages an on-demand backup of a ScaleGrid cluster.
---

# scalegrid_backup (Resource)

Triggers and manages an on-demand backup of a ScaleGrid cluster. All
configurable attributes are immutable; changing them creates a new backup.

## Example Usage

```terraform
resource "scalegrid_backup" "snapshot" {
  database   = "mongodb"
  cluster_id = scalegrid_cluster.mongo.id
  name       = "pre-migration-snapshot"
  comment    = "Taken before the v2 migration"
}
```

## Schema

### Required

- `database` (String) Engine of the cluster. Forces replacement.
- `cluster_id` (String) Cluster ID. Forces replacement.
- `name` (String) Unique backup name. Forces replacement.

### Optional

- `comment` (String) Backup description. Forces replacement.
- `target` (String) For replica sets, which node to back up (`PRIMARY`/`SECONDARY` or `MASTER`/`SLAVE`). Forces replacement.

### Read-Only

- `id` (String) Backup ID.
- `type` (String) Backup type.
- `created` (Number) Creation timestamp (epoch milliseconds).
