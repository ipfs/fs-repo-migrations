# fs-repo-migrations

[![](https://img.shields.io/badge/made%20by-Protocol%20Labs-blue.svg?style=flat-square)](http://ipn.io)
[![](https://img.shields.io/badge/project-IPFS-blue.svg?style=flat-square)](http://ipfs.io/)
[![](https://img.shields.io/badge/freenode-%23ipfs-blue.svg?style=flat-square)](http://webchat.freenode.net/?channels=%23ipfs)
[![standard-readme compliant](https://img.shields.io/badge/standard--readme-OK-green.svg?style=flat-square)](https://github.com/RichardLitt/standard-readme)

> Migrations for the filesystem repository of ipfs clients

These are migrations for the filesystem repository of [ipfs](https://github.com/ipfs/ipfs) clients. This tool is written in Go, and developed alongside [go-ipfs](https://github.com/ipfs/go-ipfs), but it should work with any repo conforming to the [fs-repo specs](https://github.com/ipfs/specs/tree/master/repo/fs-repo).

## Table of Contents

- [Install](#install)
- [Usage](#usage)
  - [When should I migrate](#when-should-i-migrate)
  - [How to Run Migrations](#how-to-run-migrations)
  - [Developing Migrations](#developing-migrations)
- [Contribute](#contribute)
  - [Want to hack on IPFS?](#want-to-hack-on-ipfs)
- [License](#license)

## Install

```sh
make install
```

## Usage

### When should I migrate

When you want to upgrade go-ipfs to a new version, you may need to
migrate.

Here is the table showing which repo version corresponds to which
go-ipfs version:

ipfs repo version | go-ipfs versions
----------------- | ----------------
                1 |  0.0.0 - 0.2.3
                2 |  0.3.0 - 0.3.11
                3 |  0.4.0 - 0.4.2
                4 |  0.4.3 - 0.4.5
                5 |  0.4.6 - 0.4.10
                6 |  0.4.11 - 0.4.15
                7 |  0.4.16 - current

### How to Run Migrations

Please see the [migration run guide here](run.md).

### Developing Migrations

Migrations are one of those things that can be extremely painful on users. At the end of the day, we want users never to have to think about it. The process should be:

- SAFE. No data lost. Ever.
- Revertible. Tools must implement forward and backward migrations.
- Frozen. After the tool is written, all code must be frozen and vendored.
- To Spec. The tools must conform to the spec.

## Contribute

Feel free to join in. All welcome. Open an [issue](https://github.com/ipfs/fs-repo-migrations/issues)!

This repository falls under the IPFS [Code of Conduct](https://github.com/ipfs/community/blob/master/code-of-conduct.md).

### Want to hack on IPFS?

[![](https://cdn.rawgit.com/jbenet/contribute-ipfs-gif/master/img/contribute.gif)](https://github.com/ipfs/community/blob/master/contributing.md)

## License

MIT
