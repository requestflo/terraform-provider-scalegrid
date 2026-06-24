# A MongoDB replica set across three AWS cloud profiles (Bring Your Own Cloud).
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

# A Dedicated (shared, ScaleGrid-hosted) replica set. On a Dedicated plan you do
# not need to reference a cloud profile: omit cloud_profile_names and the
# provider selects the shared profile for the engine automatically. Set region
# when more than one shared profile is available.
resource "scalegrid_mongodb_cluster" "dedicated" {
  name          = "dedicated-mongo"
  size          = "Small"
  version       = "V366"
  shard_count   = 1
  replica_count = 3
  region        = "useast1"
  enable_ssl    = true
}
