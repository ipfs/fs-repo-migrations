# fs-repo-migrations

[![](https://img.shields.io/badge/made%20by-Protocol%20Labs-blue.svg?style=flat-square)](http://ipn.io)
[![](https://img.shields.io/badge/project-IPFS-blue.svg?style=flat-square)](http://ipfs.io/)
[![](https://img.shields.io/badge/freenode-%23ipfs-blue.svg?style=flat-square)](http://webchat.freenode.net/?channels=%23ipfs)
[![standard-readme compliant](https://img.shields.io/badge/standard--readme-OK-green.svg?style=flat-square)](https://github.com/RichardLitt/standard-readme)

> Migrations for the filesystem repository of ipfs nodes

These are migrations for the filesystem repository of [ipfs](https://github.com/ipfs/ipfs) nodes. Each migration builds a separate binary that converts a repository to the next version.  The `fs-repo-migrations` is a tool that downloads individual migrations from the ipfs distribution site and applies them in sequence to migrate the ipfs repository to the target version.  This tool is written in Go, and developed alongside [go-ipfs](https://github.com/ipfs/go-ipfs).

## Table of Contents

- [Install](#install)
- [Usage](#usage)
  - [When should I migrate](#when-should-i-migrate)
  - [How to Run Migrations](#how-to-run-migrations)
  - [Developing Migrations](#developing-migrations)
- [Migration with Plugins](#migration-with-plugins)
- [Contribute](#contribute)
  - [Want to hack on IPFS?](#want-to-hack-on-ipfs)
- [License](#license)

## Install

```sh
make install
```

## Usage

### When should I migrate

When you want to upgrade go-ipfs to a new version, you may need to migrate.

Here is the table showing which repo version corresponds to which go-ipfs version:

| ipfs repo version | go-ipfs versions |
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
|                12 | 0.12.0 - current |

### How to Run Migrations

Please see the [migration run guide here](run.md).

### Developing Migrations

Migrations are one of those things that can be extremely painful on users. At the end of the day, we want users never to have to think about it. The process should be:

- SAFE. No data lost. Ever.
- Revertible. Tools must implement forward and backward migrations.
- Frozen. After the tool is written, all code must be frozen and vendored.
- To Spec. The tools must conform to the spec.

#### Build and Test

To create a new migration, create a go module in a directory named `fs-repo-X-to-Y`, where `X` is the repo "from" version and `Y` the repo "to" version.  Vendor the module's dependencies. The build tooling will find this module and build the migration binary.

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

Feel free to join in. All welcome. Open an [issue](https://github.com/ipfs/fs-repo-migrations/issues)!

This repository falls under the IPFS [Code of Conduct](https://github.com/ipfs/community/blob/master/code-of-conduct.md).

### Want to hack on IPFS?

[![](https://cdn.rawgit.com/jbenet/contribute-ipfs-gif/master/img/contribute.gif)](https://github.com/ipfs/community/blob/master/CONTRIBUTING.md)

## License

MIT
