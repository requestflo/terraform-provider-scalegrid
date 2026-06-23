---
page_title: "scalegrid_firewall Resource - terraform-provider-scalegrid"
description: |-
  Manages the IP whitelist (firewall) of a ScaleGrid cluster.
---

# scalegrid_firewall (Resource)

Manages the cluster-level IP whitelist for a ScaleGrid cluster. This resource
owns the **complete** CIDR list for the cluster; applying it replaces any
existing rules, and destroying it clears the list.

## Example Usage

```terraform
resource "scalegrid_firewall" "mongo" {
  database   = "mongodb"
  cluster_id = scalegrid_cluster.mongo.id
  cidr_list  = ["203.0.113.0/24", "198.51.100.10/32"]
}
```

## Schema

### Required

- `database` (String) Engine of the cluster. Forces replacement.
- `cluster_id` (String) Cluster ID. Forces replacement.
- `cidr_list` (List of String) Complete list of allowed CIDR ranges.

### Read-Only

- `id` (String) Equal to the cluster ID.

## Import

```shell
terraform import scalegrid_firewall.mongo mongodb:<cluster_id>
```
