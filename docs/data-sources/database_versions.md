---
page_title: "scalegrid_database_versions Data Source - terraform-provider-scalegrid"
description: |-
  Returns the database versions available for an engine and cloud provider.
---

# scalegrid_database_versions (Data Source)

Returns the database engine versions available for a given engine and cloud
provider. Useful for populating the `version` argument of `scalegrid_cluster`.

## Example Usage

```terraform
data "scalegrid_database_versions" "mongo" {
  database       = "mongodb"
  cloud_provider = "AWS"
}
```

## Schema

### Required

- `database` (String) Engine: `mongodb`, `redis`, `mysql`, or `postgresql`.
- `cloud_provider` (String) `AWS`, `AZURE`, or `DO`.

### Read-Only

- `versions` (Map of String) Map of version identifier to display name.
