---
page_title: "ScaleGrid Provider"
description: |-
  Manage ScaleGrid database deployments (MongoDB, Redis, MySQL, PostgreSQL) through the ScaleGrid console API.
---

# ScaleGrid Provider

The ScaleGrid provider manages [ScaleGrid](https://scalegrid.io) database
deployments and related resources through the ScaleGrid console API (the same
API used by the official `sg-cli` tool).

## Example Usage

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
```

## Authentication

The provider authenticates against the ScaleGrid console
(`https://console.scalegrid.io`) with your account **email and password**,
establishing a session cookie — the same flow as `sg-cli login`. There is no
API-key scheme.

| Argument          | Environment variable        |
|-------------------|-----------------------------|
| `email`           | `SCALEGRID_EMAIL`           |
| `password`        | `SCALEGRID_PASSWORD`        |
| `two_factor_code` | `SCALEGRID_TWO_FACTOR_CODE` |
| `base_url`        | `SCALEGRID_BASE_URL`        |

### Two-factor authentication

If the account has 2FA enabled, login requires a current TOTP code via
`two_factor_code`. Because TOTP codes expire within seconds, this is only
practical for one-shot runs. For unattended automation, use a dedicated account
with 2FA disabled.

### Dedicated / on-prem controllers

Set `base_url` to your controller's URL if you do not use the public console.

## Schema

### Optional

- `email` (String) ScaleGrid account email.
- `password` (String, Sensitive) ScaleGrid account password.
- `two_factor_code` (String, Sensitive) One-time TOTP code (see above).
- `base_url` (String) Console base URL. Defaults to `https://console.scalegrid.io`.
