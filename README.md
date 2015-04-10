# fs-repo-migrations

These are migrations for the filesystem repository of [ipfs](https://github.com/ipfs/ipfs) clients. This tool is written in Go, and developed alongside [go-ipfs](https://github.com/ipfs/go-ipfs), but it should work with any repo conforming to the [fs-repo specs](https://github.com/ipfs/specs/tree/master/repo/fs-repo).

## How to Run Migrations

Please see the [migration run guide here](run.md).

## Developing Migrations

Migrations are one of those things that can be extremely painful on users. At the end of the day, we want users never to have to think about it. The process should be:

- SAFE. No data lost. Ever.
- Revertible. Tools must implement forward and backward migrations.
- Frozen. After the tool is written, all code must be frozen and vendored.
- To Spec. The tools must conform to the spec.
