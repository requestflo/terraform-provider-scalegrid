# A MongoDB replica set across three AWS cloud profiles.
resource "scalegrid_cluster" "mongo" {
  database            = "mongodb"
  name                = "production-mongo"
  size                = "Small"
  version             = "7.0"
  shard_count         = 1
  replica_count       = 3
  cloud_profile_names = ["aws-use1-a", "aws-use1-b", "aws-use1-c"]
  enable_ssl          = true
  encrypt_disk        = true
}

# A Redis standalone deployment.
resource "scalegrid_cluster" "redis" {
  database            = "redis"
  name                = "cache"
  size                = "Small"
  version             = "7.2"
  shard_count         = 1
  server_count        = 1
  cloud_profile_names = ["aws-use1-a"]
  maxmemory_policy    = "allkeys-lru"
}

# A PostgreSQL master/slave deployment with PgBouncer.
resource "scalegrid_cluster" "postgres" {
  database            = "postgresql"
  name                = "app-db"
  size                = "Medium"
  version             = "16"
  shard_count         = 1
  replica_count       = 2
  cloud_profile_names = ["aws-use1-a", "aws-use1-b", "aws-use1-c"]
  replication_type    = "ASYNC"
  sync_commit_type    = "LOCAL"
  enable_pgbouncer    = true
}
