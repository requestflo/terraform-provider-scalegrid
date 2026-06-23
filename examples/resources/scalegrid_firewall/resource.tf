# Manage the complete IP whitelist for a cluster.
resource "scalegrid_firewall" "mongo" {
  database   = "mongodb"
  cluster_id = scalegrid_cluster.mongo.id
  cidr_list = [
    "203.0.113.0/24",
    "198.51.100.10/32",
  ]
}
