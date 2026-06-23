---
page_title: "scalegrid_follower Resource - terraform-provider-scalegrid"
description: |-
  Manages a follower (cross-cluster sync) relationship.
---

# scalegrid_follower (Resource)

Manages a follower relationship in which a target cluster periodically syncs
from a source cluster. All attributes are immutable; changing them recreates the
relationship.

## Example Usage

```terraform
resource "scalegrid_follower" "dr" {
  database          = "postgresql"
  target_cluster_id = scalegrid_cluster.dr_replica.id
  source_cluster_id = scalegrid_cluster.postgres.id
  interval_hours    = 6
  start_hour        = 2
}
```

## Schema

### Required

- `database` (String) Engine of the clusters. Forces replacement.
- `target_cluster_id` (String) Follower (destination) cluster ID. Forces replacement.
- `source_cluster_id` (String) Source cluster ID. Forces replacement.
- `interval_hours` (Number) Hours between syncs. Forces replacement.

### Optional

- `start_hour` (Number) Hour of day (0-23) for the first sync. Default `0`. Forces replacement.

### Read-Only

- `id` (String) Equal to the target cluster ID.
