# A PostgreSQL master/standby deployment with PgBouncer.
resource "scalegrid_postgresql_cluster" "this" {
  name                = "app-db"
  size                = "Medium"
  version             = "V122"
  shard_count         = 1
  replica_count       = 2
  cloud_profile_names = ["aws-use1-a", "aws-use1-b", "aws-use1-c"]
  replication_type    = "ASYNC"
  sync_commit_type    = "LOCAL"
  enable_pgbouncer    = true
}
