# A MySQL replica set with semi-synchronous replication.
resource "scalegrid_mysql_cluster" "this" {
  name                = "app-mysql"
  size                = "Medium"
  version             = "v8020"
  shard_count         = 1
  replica_count       = 3
  replica_config      = 1
  cloud_profile_names = ["aws-use1-a", "aws-use1-b", "aws-use1-c"]
  enable_ssl          = true
  encrypt_disk        = true
}
