# Daily scheduled backups for a PostgreSQL cluster, kept to the last 7, taken at
# 02:00 UTC. The ScaleGrid API has no endpoint to read the schedule back, so this
# resource applies the policy but cannot detect out-of-band drift.
resource "scalegrid_backup_schedule" "this" {
  database        = "postgresql"
  cluster_id      = scalegrid_postgresql_cluster.this.id
  enabled         = true
  interval_hours  = 24
  hour            = 2
  retention_limit = 7
}
