# A Redis standalone deployment.
resource "scalegrid_redis_cluster" "this" {
  name                = "cache"
  size                = "Small"
  version             = "V505"
  shard_count         = 1
  server_count        = 1
  cloud_profile_names = ["aws-use1-a"]
}

# A three-node Redis master/slave deployment with sentinels.
resource "scalegrid_redis_cluster" "ha" {
  name                  = "cache-ha"
  size                  = "Small"
  version               = "V505"
  shard_count           = 1
  server_count          = 3
  sentinel_count        = 3
  cloud_profile_names   = ["aws-use1-a", "aws-use1-b", "aws-use1-c"]
  backup_interval_hours = 24
  encrypt_disk          = true
}
