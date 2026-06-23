---
page_title: "scalegrid_cluster Resource - terraform-provider-scalegrid"
description: |-
  Manages a ScaleGrid database deployment (cluster).
---

# scalegrid_cluster (Resource)

Manages a ScaleGrid database deployment for MongoDB, Redis, MySQL, or
PostgreSQL. Provisioning is asynchronous; Terraform waits for the provisioning
job to finish before completing the apply. Most attributes are immutable —
changing them forces a new cluster. `size` (scale) and `paused` (pause/resume)
are applied in place.

## Example Usage

```terraform
resource "scalegrid_cluster" "mongo" {
  database            = "mongodb"
  name                = "production-mongo"
  size                = "Small"
  version             = "7.0"
  shard_count         = 1
  replica_count       = 3
  cloud_profile_names = ["aws-use1-a", "aws-use1-b", "aws-use1-c"]
  enable_ssl          = true
}
```

## Schema

### Required

- `database` (String) Engine: `mongodb`, `redis`, `mysql`, or `postgresql`. Forces replacement.
- `name` (String) Unique cluster name. Forces replacement.
- `size` (String) Size tier: `Micro`, `Small`, `Medium`, `Large`, `XLarge`, `X2XLarge`, `X4XLarge`. Updatable (scales in place).
- `version` (String) Engine version. See the `scalegrid_database_versions` data source. Forces replacement.
- `cloud_profile_names` (List of String) Cloud profile names to deploy nodes into. Forces replacement.

### Optional

- `shard_count` (Number) Shards. Default `1`. Forces replacement.
- `replica_count` (Number) Nodes per shard for MongoDB/MySQL/PostgreSQL. Forces replacement.
- `server_count` (Number) Nodes per shard for Redis. Forces replacement.
- `sentinel_count` (Number) Redis sentinel nodes. Forces replacement.
- `sentinel_cloud_profile_names` (List of String) Cloud profiles for Redis sentinels. Forces replacement.
- `encrypt_disk` (Boolean) Encrypt the data disk. Default `false`. Forces replacement.
- `enable_ssl` (Boolean) Enable SSL/TLS. Default `false`. Forces replacement.
- `paused` (Boolean) Pause/resume the cluster. Default `false`. Updatable.
- `mongo_engine` (String) MongoDB storage engine. Forces replacement.
- `compression_algo` (String) MongoDB compression (`snappy`, `zlib`, `zstd`). Forces replacement.
- `cluster_mode` (Boolean) Redis cluster mode. Forces replacement.
- `backup_interval_hours` (Number) Redis scheduled backup interval. Forces replacement.
- `maxmemory_policy` (String) Redis eviction policy. Forces replacement.
- `enable_rdb` (Boolean) Redis RDB snapshots. Forces replacement.
- `enable_aof` (Boolean) Redis AOF persistence. Forces replacement.
- `replica_config` (Number) MySQL replication mode (0/1/2). Forces replacement.
- `replication_type` (String) PostgreSQL `ASYNC`/`SYNC`. Forces replacement.
- `sync_commit_type` (String) PostgreSQL synchronous commit type. Forces replacement.
- `enable_pgbouncer` (Boolean) PostgreSQL PgBouncer pooling. Forces replacement.

### Read-Only

- `id` (String) Cluster ID.
- `status` (String) Lifecycle status.
- `cluster_type` (String) Topology.
- `disk_size_gb` (Number) Provisioned disk size.
- `encryption_enabled` (Boolean) Whether encryption at rest is active.
- `ssl_active` (Boolean) Whether SSL is active.

~> **Import is not supported.** Cluster creation parameters such as
`cloud_profile_names` and the per-engine replica/server counts cannot be read
back from the API, so an imported cluster would always show a destructive
replacement diff. Define clusters in configuration instead.
