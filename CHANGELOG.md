# [2.4.0](https://github.com/requestflo/terraform-provider-scalegrid/compare/v2.3.0...v2.4.0) (2026-06-26)


### Features

* add scalegrid_backup_schedule resource ([#16](https://github.com/requestflo/terraform-provider-scalegrid/issues/16)) ([df66f8f](https://github.com/requestflo/terraform-provider-scalegrid/commit/df66f8f42b37f7806c08494afd3581f2d84b4c3f))

# [2.3.0](https://github.com/requestflo/terraform-provider-scalegrid/compare/v2.2.2...v2.3.0) (2026-06-26)


### Features

* allow the Nano instance size ([31bbe13](https://github.com/requestflo/terraform-provider-scalegrid/commit/31bbe136de88aef1cbe3e1139b11d3a1d07665d0))
* allow the Nano instance size ([#15](https://github.com/requestflo/terraform-provider-scalegrid/issues/15)) ([f8ba75b](https://github.com/requestflo/terraform-provider-scalegrid/commit/f8ba75bd79290ead342cf1e9f48eb7fc1404091c))

## [2.2.2](https://github.com/requestflo/terraform-provider-scalegrid/compare/v2.2.1...v2.2.2) (2026-06-26)


### Bug Fixes

* decode action id returned as a JSON number ([c1cb994](https://github.com/requestflo/terraform-provider-scalegrid/commit/c1cb994cff198105d802efa43b36267b83ea7fb9))
* decode action id returned as a JSON number ([#14](https://github.com/requestflo/terraform-provider-scalegrid/issues/14)) ([0700712](https://github.com/requestflo/terraform-provider-scalegrid/commit/07007120051247e5804c29d4ca41a1c317833a33))

## [2.2.1](https://github.com/requestflo/terraform-provider-scalegrid/compare/v2.2.0...v2.2.1) (2026-06-26)


### Bug Fixes

* decode database versions returned as a JSON object ([8d38a04](https://github.com/requestflo/terraform-provider-scalegrid/commit/8d38a0415b81249c8687dd43a4f801d7d0975bbb))
* decode database versions returned as a JSON object ([#13](https://github.com/requestflo/terraform-provider-scalegrid/issues/13)) ([b00d585](https://github.com/requestflo/terraform-provider-scalegrid/commit/b00d585a8bf22395cd4c757d3946ee257bc47b79))

# [2.2.0](https://github.com/requestflo/terraform-provider-scalegrid/compare/v2.1.0...v2.2.0) (2026-06-25)


### Features

* add cloud_provider selector for shared (Dedicated) hosting ([baa3474](https://github.com/requestflo/terraform-provider-scalegrid/commit/baa34740b59b3cbad8b7ceb3b3e72ec420da534c))
* add cloud_provider selector for shared (Dedicated) hosting ([#12](https://github.com/requestflo/terraform-provider-scalegrid/issues/12)) ([a3b0438](https://github.com/requestflo/terraform-provider-scalegrid/commit/a3b0438cd3779cebde9923c82c4343062e66dfde))

# [2.1.0](https://github.com/requestflo/terraform-provider-scalegrid/compare/v2.0.0...v2.1.0) (2026-06-24)


### Features

* make cloud_profile_names optional for shared (Dedicated) hosting ([5edddf4](https://github.com/requestflo/terraform-provider-scalegrid/commit/5edddf4bfae9649b895b196c6e8d481e928b00c2))
* make cloud_profile_names optional for shared (Dedicated) hosting ([#11](https://github.com/requestflo/terraform-provider-scalegrid/issues/11)) ([7763a66](https://github.com/requestflo/terraform-provider-scalegrid/commit/7763a6652346e496071fa38f93bcdd5d43e02a78))

# [2.0.0](https://github.com/requestflo/terraform-provider-scalegrid/compare/v1.0.1...v2.0.0) (2026-06-24)


* feat!: split per-engine cluster resources and reconcile client with the OpenAPI spec ([bfb95b6](https://github.com/requestflo/terraform-provider-scalegrid/commit/bfb95b6df642215ff7864d2aeed7b9eff0704683))


### Bug Fixes

* align async actions, error envelope, backups and alert rules with the spec ([07aff31](https://github.com/requestflo/terraform-provider-scalegrid/commit/07aff31d4fb37a250fe370fccf1e392233ad06dc))


### BREAKING CHANGES

* `scalegrid_cluster` is removed in favour of per-engine
resources. There are no `moved` blocks; configurations must be migrated to
the new resource types.

Client reconciliation against console.scalegrid.io:
- Decode integer IDs (clusterID/machinePoolID/actionID and the id fields on
  clusters, cloud profiles, backups and alert rules) which the API returns
  as JSON numbers, tolerating both actionID/actionId casings.
- Handle the singular `cluster` list key (Mongo/Redis/MySQL) vs `clusters`
  (PostgreSQL).
- Use PostgreSQL's all-lowercase endpoint paths for list/fetch/scale/delete/
  credentials/backup/restore, and its mixed-case create/deletebackup paths.
- Treat the database-versions response as a string array, not a map; add GCP
  to the supported cloud providers.
- Compare async action status case-insensitively (Running/Completed/Failed),
  fixing a poll loop that never detected completion.
- Drop the undocumented `enableAuth` field from create bodies; align the
  Redis create body and remove unsupported knobs (redisConfigParams,
  sentinelMachinePool, maxmemory_policy, enable_rdb, enable_aof,
  sentinel_cloud_profile_names).
- PostgreSQL on-demand backup sends type=ONDEMAND and a target; PG backup
  deletion uses /PostgreSQLClusters/deletebackup; alert-rule deletion sends
  no body.
- Restrict compression_algo to snappy/zlib; expand alert rule type and
  notification channel enums; document that pause/resume is BYOC-only.

Regenerate docs and examples for the new resource layout.

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_014QBX2XjGQq6hAN1mnHtSHf

## [1.0.1](https://github.com/requestflo/terraform-provider-scalegrid/compare/v1.0.0...v1.0.1) (2026-06-24)


### Bug Fixes

* include manifest checksum in SHA256SUMS for the registry ([#9](https://github.com/requestflo/terraform-provider-scalegrid/issues/9)) ([cbb3b7d](https://github.com/requestflo/terraform-provider-scalegrid/commit/cbb3b7dd5a55ffc472f1fb07c7681c30aca97608))

# 1.0.0 (2026-06-24)


### Bug Fixes

* gate releases on a dry run and verify build/signing before tagging ([#7](https://github.com/requestflo/terraform-provider-scalegrid/issues/7)) ([32c7e18](https://github.com/requestflo/terraform-provider-scalegrid/commit/32c7e187337aece96a93ca64bcd094cb67192de2))
* match first-release wording in the release-pending gate ([#8](https://github.com/requestflo/terraform-provider-scalegrid/issues/8)) ([43e9a01](https://github.com/requestflo/terraform-provider-scalegrid/commit/43e9a01a162572e349fd984e35eb1730141c8298))
* sign release artifacts non-interactively in CI ([#5](https://github.com/requestflo/terraform-provider-scalegrid/issues/5)) ([3ec5e97](https://github.com/requestflo/terraform-provider-scalegrid/commit/3ec5e97c49bdc9fe5d80048e29b988a543f36134))
* verify GPG signing before tagging the release ([#6](https://github.com/requestflo/terraform-provider-scalegrid/issues/6)) ([468748f](https://github.com/requestflo/terraform-provider-scalegrid/commit/468748fe70411800848c80cea0656f226a7358d8))


### Features

* initial public release of the ScaleGrid provider ([#4](https://github.com/requestflo/terraform-provider-scalegrid/issues/4)) ([d290c3e](https://github.com/requestflo/terraform-provider-scalegrid/commit/d290c3eba161a594f13104973dcf6522545acb16))

# Changelog

All notable changes to this project are documented in this file. It is
maintained automatically by [semantic-release](https://semantic-release.gitbook.io/)
based on [Conventional Commits](https://www.conventionalcommits.org/); do not
edit it by hand.
