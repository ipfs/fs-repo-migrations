> [!IMPORTANT]
> **This repository is archived. No new migrations will be added here, ever.**
>
> Kubo stopped downloading external migration binaries in
> [v0.37.0](https://github.com/ipfs/kubo/blob/master/docs/changelogs/v0.37.md#-repository-migration-from-v16-to-v17-with-embedded-tooling).
> Repo migrations from version 17 onward are built into the `ipfs` binary and
> run in milliseconds with no network access. What is here still works for
> migrating a repo older than version 17 up to version 16, and that is the only
> thing it is for. See [Why this is archived](#why-this-is-archived).

# fs-repo-migrations

[![](https://img.shields.io/badge/made%20by-Protocol%20Labs-blue.svg?style=flat-square)](https://protocol.ai)
[![](https://img.shields.io/badge/project-IPFS-blue.svg?style=flat-square)](https://ipfs.tech/)
[![standard-readme compliant](https://img.shields.io/badge/standard--readme-OK-green.svg?style=flat-square)](https://github.com/RichardLitt/standard-readme)

> Migrations for the filesystem repository of Kubo IPFS nodes, up to repo version 16

These are migrations for the filesystem repository of [Kubo](https://github.com/ipfs/kubo) IPFS nodes. Each migration builds a separate binary that converts a repository to the next version. The `fs-repo-migrations` tool downloads individual migrations from the ipfs distribution site and applies them in sequence.

This covers repo versions up to 16 (Kubo 0.36 and older). From repo version 17 on, migrations are [embedded in Kubo itself](https://github.com/ipfs/kubo/blob/master/repo/fsrepo/migrations/README.md) and this tool refuses to run.

## Table of Contents

- [Why this is archived](#why-this-is-archived)
- [Install](#install)
- [Usage](#usage)
  - [When should I migrate](#when-should-i-migrate)
  - [How to Run Migrations](#how-to-run-migrations)
  - [Developing Migrations](#developing-migrations)
- [Migration with Plugins](#migration-with-plugins)
- [Contribute](#contribute)
  - [Want to hack on IPFS?](#want-to-hack-on-ipfs)
- [License](#license)

## Why this is archived

The design here was: ship a small binary per repo version, publish them to a
distribution site, and have the daemon fetch and execute the ones it needs at
upgrade time. That worked in 2016. It does not fit how operating systems treat
downloaded executables now.

- **Download-then-execute is a blocked pattern.** macOS quarantines anything
  fetched from the network and Gatekeeper refuses to run it unless it is signed
  and notarized. Windows SmartScreen and Defender flag freshly downloaded
  unsigned binaries, and attack-surface-reduction rules in managed fleets block
  the launch outright. On Linux, hardened hosts mount `/tmp` and home
  directories `noexec`, and endpoint agents treat "daemon downloaded a binary
  and ran it" as an incident. A node upgrade should not look like malware.
- **It needs the network at the worst moment.** The daemon could not start
  until it reached a distribution server, which turned an offline or firewalled
  or air-gapped machine into a failed upgrade.
- **It was a supply chain to defend.** Every migration binary is a separately
  built, separately signed, separately hosted artifact whose provenance the
  daemon had to verify at runtime. Kubo eventually hardened this to
  [trustless CAR fetches with local verification](https://github.com/ipfs/kubo/blob/master/docs/changelogs/v0.27.md),
  which is the right fix for the legacy path but does not make the shape worth
  keeping.

So Kubo moved the code inside the binary. Migrations now compile in, run
offline, and inherit the signature and provenance of the release itself.

- Kubo [v0.37.0](https://github.com/ipfs/kubo/blob/master/docs/changelogs/v0.37.md#-repository-migration-from-v16-to-v17-with-embedded-tooling) introduced embedded migrations with repo v16 to v17.
- Kubo [v0.38.0](https://github.com/ipfs/kubo/blob/master/docs/changelogs/v0.38.md#-repository-migration-simplified-provide-configuration) shipped repo v17 to v18 the same way.
- The legacy download path still exists in Kubo for repos older than v17, and is
  slated for removal. Only embedded migrations for recent releases will be kept,
  so a very old repo should be upgraded in stages rather than in one jump.

**New migrations go in [ipfs/kubo](https://github.com/ipfs/kubo) under
`repo/fsrepo/migrations/`, not here.** Do not add another external migration to
this repo, and do not build a new tool that downloads and executes migration
binaries.

## Install

```sh
make install
```

## Usage

### When should I migrate

When you want to upgrade Kubo to a new version, you may need to migrate.

Here is the table showing which repo version corresponds to which Kubo version:

| ipfs repo version | Kubo versions    |
| ----------------: | :--------------- |
|                 1 | 0.0.0 - 0.2.3.   |
|                 2 | 0.3.0 - 0.3.11   |
|                 3 | 0.4.0 - 0.4.2    |
|                 4 | 0.4.3 - 0.4.5    |
|                 5 | 0.4.6 - 0.4.10   |
|                 6 | 0.4.11 - 0.4.15  |
|                 7 | 0.4.16 - 0.4.23  |
|                 8 | 0.5.0 - 0.6.0    |
|                 9 | 0.5.0 - 0.6.0    |
|                10 | 0.6.0 - 0.7.0    |
|                11 | 0.8.0 - 0.11.0   |
|                12 | 0.12.0 - 0.17.0  |
|                13 | 0.18.0 - 0.20.0  |
|                14 | 0.21.0 - 0.22.0  |
|                15 | 0.23.0 - 0.29.0  |
|                16 | 0.30.0 - 0.36.0  |

Repo versions 17 and up are handled by Kubo itself, not by this tool:

| ipfs repo version | Kubo versions    | migration lives in |
| ----------------: | :--------------- | :----------------- |
|                17 | 0.37.0           | [kubo `repo/fsrepo/migrations`](https://github.com/ipfs/kubo/tree/master/repo/fsrepo/migrations) |
|                18 | 0.38.0 - current | [kubo `repo/fsrepo/migrations`](https://github.com/ipfs/kubo/tree/master/repo/fsrepo/migrations) |

### How to Run Migrations

Please see the [migration run guide here](run.md).

### Developing Migrations

**Do not develop new migrations here.** Repo version 16 is the last one this
repo handles. Anything newer belongs in
[`repo/fsrepo/migrations`](https://github.com/ipfs/kubo/tree/master/repo/fsrepo/migrations)
in the Kubo tree, compiled into the `ipfs` binary. See
[Why this is archived](#why-this-is-archived).

The principles below still hold wherever a migration is written:

- SAFE. No data lost. Ever.
- Revertible. Tools must implement forward and backward migrations.
- Frozen. After the tool is written, all code must be frozen and vendored.
- To Spec. The tools must conform to the spec.

The rest of this section describes how the existing migrations in this repo were
built, and is kept for anyone maintaining a fork or rebuilding one with a plugin.

#### Build and Test

Each migration is a go module in a directory named `fs-repo-X-to-Y`, where `X` is the repo "from" version and `Y` the repo "to" version, with its dependencies vendored. The build tooling finds these modules and builds the migration binaries.

If the migration directory contains a subdirectory named `sharness`, tests contained in it are run using the sharness test tool. Tests must be named `tNNNN-*.sh`, where NNNN is a 4-digit sequence number.

After the migration is merged into the main repo branch, create a version tag for it.  This is necessary for versioning individual migrations within the repo.
```sh
git tag <migration>/v<version>
git push origin <migration>/v<version>
```

Example:
```sh
git tag fs-repo-99-to-100/v1.0.1
git push origin fs-repo-99-to-100/v1.0.1
```

#### Ignoring legacy migrations

We "archive" legacy migrations by adding them to `ignored-migrations` file.
This keeps CI fast, and removes issues caused by old, with unsupported go code.

### Dependencies

Dependencies must be vendored independently for each migration. Each migration is a separate go module with its own `vendor` directory (created with `go mod vendor` for that migration).  All migrations are built using `go build -mod=vendor` to ensure dependencies come from the module's `vendor` directory.

## Migration with Plugins
If IPFS plugins were used to operate your IPFS datastore, such as the [ipfs-ds-s3](https://github.com/ipfs/go-ds-s3) plugin, then migration may require building a custom migration with the plugin built into it.  There is a script to assist with the process: `build-plugin.sh`.

### Build a Migration with a Plugin
This requires that you have Go installed.

First clone the `fs-repo-migrations` github repo:
```sh
git clone https://github.com/ipfs/fs-repo-migrations.git
```

Then run the `build-plugin.sh` script, supplying the necessary arguments (run with -h for help).

### Run the Custom Migration
After the custom migration with plugin(s) has built successfully, change to the migration directory and run the migration binary.  You can also copy the migration binary into your `PATH` if you want it to be run by `ipfs-update` or by the `fs-repo-migrations` tool.

## Contribute

This repository is archived and is not accepting contributions.

Work on repo migrations happens in [ipfs/kubo](https://github.com/ipfs/kubo).
Open issues and pull requests there.

The IPFS [Code of Conduct](https://github.com/ipfs/community/blob/master/code-of-conduct.md) applies there as it did here.

### Want to hack on IPFS?

[![](https://cdn.rawgit.com/jbenet/contribute-ipfs-gif/master/img/contribute.gif)](https://github.com/ipfs/community/blob/master/CONTRIBUTING.md)

## License

MIT

---

If you are an AI coding assistant working in this repo, read [AGENTS.md](AGENTS.md) first.
