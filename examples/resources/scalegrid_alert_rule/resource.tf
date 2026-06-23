# Alert when free disk drops below 10%.
resource "scalegrid_alert_rule" "low_disk" {
  database      = "mongodb"
  cluster_id    = scalegrid_cluster.mongo.id
  type          = "DISK_FREE"
  operator      = "LT"
  threshold     = "10.0"
  notifications = ["EMAIL"]
}

# Metric-based alert with a duration window.
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
