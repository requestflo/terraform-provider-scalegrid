---
page_title: "scalegrid_cluster_credentials Data Source - terraform-provider-scalegrid"
description: |-
  Fetches root credentials and connection strings for a ScaleGrid cluster.
---

# scalegrid_cluster_credentials (Data Source)

Fetches the root database credentials and connection strings for a cluster.
The password and connection strings are marked sensitive.

## Example Usage

```terraform
data "scalegrid_cluster_credentials" "mongo" {
  database   = "mongodb"
  cluster_id = scalegrid_cluster.mongo.id
}
```

## Schema

### Required

- `database` (String) Engine: `mongodb`, `redis`, `mysql`, or `postgresql`.
- `cluster_id` (String) Cluster ID.

### Read-Only

- `username` (String) Root database username.
- `password` (String, Sensitive) Root database password.
- `command_line` (String, Sensitive) Command-line connection syntax.
- `connection_strings` (List of Object) Driver-specific connection strings, each
  with `driver` and `connection_string` (sensitive).
