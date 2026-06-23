# Terraform Provider for ScaleGrid

A Terraform provider for [ScaleGrid](https://scalegrid.io) — manage MongoDB,
Redis, MySQL, and PostgreSQL database deployments (and their cloud profiles,
firewall whitelists, backups, alert rules, and follower relationships) as code.

Built with the [Terraform Plugin Framework](https://developer.hashicorp.com/terraform/plugin/framework)
(protocol 6).

## How this maps to the ScaleGrid API

ScaleGrid does not publish an open REST API behind an API key. The automation
surface is the ScaleGrid **console API** (`console.scalegrid.io`), the same API
driven by the official `sg-cli` tool: you authenticate with your account email
and password (cookie session), operations return an `actionID` you poll to
completion, and endpoints are per-engine (`/MongoClusters/create`,
`/RedisClusters/list`, …). This provider is implemented directly against that
API, so its capabilities mirror what `sg-cli` (and the console) can do.

## Coverage

The provider exposes the declarative (desired-state) portion of the ScaleGrid
API. Every CLI capability that represents managed state is covered:

| Area | CLI commands | Provider |
|------|--------------|----------|
| Clusters (create/read/delete) | `create-cluster`, `list-clusters`, `delete-cluster` | `scalegrid_cluster` |
| Scale in place | `scale-up` | `scalegrid_cluster.size` |
| Pause / resume | `pause-cluster`, `resume-cluster` | `scalegrid_cluster.paused` |
| Cloud profiles (AWS) | `create-cloud-profile`, `list-cloud-profiles`, `delete-cloud-profile`, `update-cloud-profile-keys` | `scalegrid_cloud_profile` |
| Firewall whitelist | `set-firewall-rules`, `get-firewall-rules` | `scalegrid_firewall` |
| On-demand backups | `start-backup`, `delete-backup`, `list-backups` | `scalegrid_backup` |
| Alert rules | `create-alert-rule`, `list-alert-rules`, `delete-alert-rule` | `scalegrid_alert_rule` |
| Followers | `setup-follower`, `get-follower-status`, `stop-following` | `scalegrid_follower` |
| Credentials | `get-cluster-credentials` | `scalegrid_cluster_credentials` (data source) |
| Available versions | `get-available-db-versions` | `scalegrid_database_versions` (data source) |

**Intentionally out of scope** are imperative, one-shot maintenance actions that
do not represent persistent desired state and therefore do not fit Terraform's
model: `patch-os`, `upgrade-agent`, `compact`, `refresh-cluster`,
`reset-credentials`, `restore-backup`, `sync-follower`, `resolve-alerts`,
`build-index`, `add-column`/`add-index`, and live config edits
(`update-config`, `set-pgbouncer`). These are better run via `sg-cli` or a
`null_resource`/provisioner when needed.

Azure cloud profiles are not managed because creating one requires interactively
running a generated permission-granting script; AWS profiles are fully supported.

## Usage

```terraform
terraform {
  required_providers {
    scalegrid = {
      source  = "requestflo/scalegrid"
      version = "~> 0.1"
    }
  }
}

provider "scalegrid" {
  email    = "you@example.com"
  password = var.scalegrid_password
}

resource "scalegrid_cloud_profile" "aws" {
  database            = "mongodb"
  name                = "aws-use1-a"
  region              = "us-east-1"
  access_key          = var.aws_access_key
  secret_key          = var.aws_secret_key
  vpc_id              = "vpc-0123456789abcdef0"
  subnet_id           = "subnet-0123456789abcdef0"
  vpc_cidr            = "10.0.0.0/16"
  subnet_cidr         = "10.0.1.0/24"
  security_group_id   = "sg-0123456789abcdef0"
  security_group_name = "scalegrid-mongo"
}

resource "scalegrid_cluster" "mongo" {
  database            = "mongodb"
  name                = "production-mongo"
  size                = "Small"
  version             = "7.0"
  shard_count         = 1
  replica_count       = 3
  cloud_profile_names = [scalegrid_cloud_profile.aws.name]
  enable_ssl          = true
}

resource "scalegrid_firewall" "mongo" {
  database   = "mongodb"
  cluster_id = scalegrid_cluster.mongo.id
  cidr_list  = ["203.0.113.0/24"]
}
```

More examples are under [`examples/`](./examples); reference docs under
[`docs/`](./docs).

## Authentication

The provider logs in to the ScaleGrid console with email + password and reuses
the session cookie. Configure via the provider block or environment:

| Argument          | Environment variable        | Default |
|-------------------|-----------------------------|---------|
| `email`           | `SCALEGRID_EMAIL`           | — |
| `password`        | `SCALEGRID_PASSWORD`        | — |
| `two_factor_code` | `SCALEGRID_TWO_FACTOR_CODE` | — |
| `base_url`        | `SCALEGRID_BASE_URL`        | `https://console.scalegrid.io` |

**Two-factor auth:** TOTP codes expire within seconds, so `two_factor_code` is
only practical for one-shot runs. For unattended automation use an account with
2FA disabled. Set `base_url` for a dedicated/on-prem controller.

## Development

Requires Go 1.23+.

```sh
make build      # compile the provider
make test       # unit tests
make vet        # go vet
make fmt        # gofmt
make install    # build + install into ~/.terraform.d/plugins for local testing
```

To use a local build, add a dev override to `~/.terraformrc`:

```hcl
provider_installation {
  dev_overrides {
    "registry.terraform.io/requestflo/scalegrid" = "/path/to/go/bin"
  }
  direct {}
}
```

### Testing

Unit tests use an `httptest` server that emulates the ScaleGrid envelope and
cover login (incl. 2FA), the error/not-found contract, cluster creation, action
polling, and firewall round-trips, plus schema validation for every resource and
data source:

```sh
go test ./...
```

`make testacc` runs acceptance tests against a live account (creates real,
billable resources; gated behind `TF_ACC=1`).

## Repository layout

```
.
├── main.go                 # provider entrypoint
├── internal/
│   ├── client/             # ScaleGrid console API client (no Terraform deps)
│   └── provider/           # provider, resources, and data sources
├── docs/                   # registry documentation
├── examples/               # example configurations
└── .github/workflows/      # CI and release pipelines
```
