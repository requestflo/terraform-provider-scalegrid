---
page_title: "scalegrid_alert_rule Resource - terraform-provider-scalegrid"
description: |-
  Manages an alert rule on a ScaleGrid cluster.
---

# scalegrid_alert_rule (Resource)

Manages an alert rule on a ScaleGrid cluster. Alert rules are immutable;
changing any attribute replaces the rule.

## Example Usage

```terraform
resource "scalegrid_alert_rule" "high_cpu" {
  database      = "mongodb"
  cluster_id    = scalegrid_cluster.mongo.id
  type          = "METRIC"
  metric        = "CPU_USAGE"
  operator      = "GT"
  threshold     = "85.0"
  duration      = "SIX"
  notifications = ["EMAIL", "PAGERDUTY"]
}
```

## Schema

### Required

- `database` (String) Engine of the cluster. Forces replacement.
- `cluster_id` (String) Cluster ID. Forces replacement.
- `type` (String) `METRIC`, `DISK_FREE`, or `ROLE_CHANGE`. Forces replacement.
- `operator` (String) `EQ`, `LT`, `GT`, `LTE`, or `GTE`. Forces replacement.
- `threshold` (String) Threshold value. Forces replacement.
- `notifications` (List of String) `EMAIL`, `SMS`, `PAGERDUTY`. Forces replacement.

### Optional

- `metric` (String) Metric name (required when `type` is `METRIC`). Forces replacement.
- `duration` (String) `TWO`, `SIX`, `HOURLY`, or `DAILY`. Forces replacement.

### Read-Only

- `id` (String) Alert rule ID.

## Import

```shell
terraform import scalegrid_alert_rule.high_cpu mongodb:<cluster_id>:<rule_id>
```
