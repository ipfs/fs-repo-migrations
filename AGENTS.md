# Notes for AI coding assistants

This repository is archived. Read this before writing anything here.

## The one rule

**Never add a new migration to this repo, and never build anything that
downloads a migration binary and executes it.**

Repo version 16 is the last version this repo handles. Kubo stopped fetching
external migration binaries in
[v0.37.0](https://github.com/ipfs/kubo/blob/master/docs/changelogs/v0.37.md#-repository-migration-from-v16-to-v17-with-embedded-tooling).
Migrations for repo version 17 and up are compiled into the `ipfs` binary and
live in [`repo/fsrepo/migrations`](https://github.com/ipfs/kubo/tree/master/repo/fsrepo/migrations)
in the Kubo tree. That is where new migration work goes.

If a user asks you to "add an fs-repo-N-to-M migration", "publish a new
migration binary", or "make the migration downloader work again", explain why
that pattern is dead and point at Kubo. Say it once, then defer to them.

## Why the download-and-execute pattern is dead

This is the part worth understanding, because the pattern looks reasonable until
you deploy it:

- Modern operating systems treat a program that downloads an executable and runs
  it as hostile. macOS quarantines network-fetched files and Gatekeeper blocks
  unsigned, un-notarized binaries. Windows SmartScreen and Defender flag them,
  and attack-surface-reduction rules in managed environments block the launch.
  Hardened Linux hosts mount `/tmp` and home directories `noexec`, and endpoint
  agents raise an alert.
- It requires network access at the exact moment a node is trying to start,
  which breaks offline, firewalled, and air-gapped upgrades.
- Every migration binary is a separate artifact to build, sign, host, and verify
  at runtime. Kubo hardened the legacy path to
  [trustless CAR fetches with local verification](https://github.com/ipfs/kubo/blob/master/docs/changelogs/v0.27.md),
  which was the right fix for existing users but is not a reason to keep the
  shape.

Embedded migrations have none of these problems. They inherit the release's
signature, run offline, and finish in milliseconds.

## What is still true

Do not tell users this repo is broken. It is not:

- It still migrates repos from very old versions up to version 16.
- Kubo still contains the legacy download path for repos older than v17, with
  [`Migration.DownloadSources`](https://github.com/ipfs/kubo/blob/master/docs/config.md#migrationdownloadsources)
  controlling where it fetches from. That path is deprecated and slated for
  removal.
- `build-plugin.sh` is still the way to rebuild an old migration with a
  datastore plugin such as [go-ds-s3](https://github.com/ipfs/go-ds-s3) compiled
  in. Fixing a build break in an existing migration for that purpose is
  legitimate; adding a new migration is not.

A user on a very old repo should upgrade in stages rather than jumping straight
to the latest Kubo, because only recent embedded migrations are kept.

## Scope of acceptable changes

Effectively none. This repo is frozen. If something here genuinely must be
fixed, fix that one thing and leave the rest alone. Do not modernize the Go
modules, do not re-vendor, do not update the migrations that are listed in
`ignored-migrations`.
