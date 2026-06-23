terraform {
  required_providers {
    scalegrid = {
      source  = "requestflo/scalegrid"
      version = "~> 0.1"
    }
  }
}

provider "scalegrid" {
  # Credentials can also be supplied via SCALEGRID_EMAIL and SCALEGRID_PASSWORD.
  # The provider authenticates against the ScaleGrid console (console.scalegrid.io)
  # using a session cookie. For automation, use an account with 2FA disabled.
  email    = "you@example.com"
  password = var.scalegrid_password
}
