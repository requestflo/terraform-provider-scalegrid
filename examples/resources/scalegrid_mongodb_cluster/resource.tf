# A MongoDB replica set across three AWS cloud profiles.
resource "scalegrid_mongodb_cluster" "this" {
  name                = "production-mongo"
  size                = "Small"
  version             = "V366"
  shard_count         = 1
  replica_count       = 3
  cloud_profile_names = ["aws-use1-a", "aws-use1-b", "aws-use1-c"]
  mongo_engine        = "wiredtiger"
  compression_algo    = "snappy"
  enable_ssl          = true
  encrypt_disk        = true
}
