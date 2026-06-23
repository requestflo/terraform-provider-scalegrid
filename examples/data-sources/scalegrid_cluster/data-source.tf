# Look up a cluster by name.
data "scalegrid_cluster" "mongo" {
  database = "mongodb"
  name     = "production-mongo"
}

# List all PostgreSQL clusters.
data "scalegrid_clusters" "postgres" {
  database = "postgresql"
}

# Resolve a cloud profile by name.
data "scalegrid_cloud_profile" "aws" {
  name = "aws-use1-a"
}

# Discover available engine versions for AWS.
data "scalegrid_database_versions" "mongo" {
  database       = "mongodb"
  cloud_provider = "AWS"
}

# Fetch connection credentials for a cluster.
data "scalegrid_cluster_credentials" "mongo" {
  database   = "mongodb"
  cluster_id = data.scalegrid_cluster.mongo.id
}
