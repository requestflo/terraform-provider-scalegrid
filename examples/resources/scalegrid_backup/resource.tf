# Trigger an on-demand backup of a cluster.
resource "scalegrid_backup" "snapshot" {
  database   = "mongodb"
  cluster_id = scalegrid_cluster.mongo.id
  name       = "pre-migration-snapshot"
  comment    = "Taken before the v2 migration"
}
