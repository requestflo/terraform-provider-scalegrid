# Make one cluster follow another, syncing every 6 hours starting at 02:00.
resource "scalegrid_follower" "dr" {
  database          = "postgresql"
  target_cluster_id = scalegrid_cluster.dr_replica.id
  source_cluster_id = scalegrid_cluster.postgres.id
  interval_hours    = 6
  start_hour        = 2
}
