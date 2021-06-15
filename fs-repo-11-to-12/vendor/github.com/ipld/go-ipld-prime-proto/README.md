# go-ipld-prime-proto

[![](https://img.shields.io/badge/made%20by-Protocol%20Labs-blue.svg?style=flat-square)](https://protocol.ai/)
[![Coverage Status](https://codecov.io/gh/ipld/go-ipld-prime-proto/branch/master/graph/badge.svg)](https://codecov.io/gh/ipld/go-ipld-prime-proto)
[![Travis CI](https://travis-ci.com/ipld/go-ipld-prime-proto.svg?branch=master)](https://travis-ci.com/ipld/go-ipld-prime-proto)

> An implementation of the [DAG Protobuf](https://github.com/ipld/specs/blob/master/block-layer/codecs/dag-pb.md) for the [go-ipld-prime library](https://github.com/ipld/go-ipld-prime)

## Table of Contents

- [Background](#background)
- [Install](#install)
- [Usage](#usage)
- [Contribute](#contribute)
- [License](#license)

## Background

The package adds Dag Protobuf support to go-ipld-prime. It is designed primarily as an adjunct to `go-ipld-prime` to enable using selectors and Graphsync, features which are unique to `go-ipld-prime`, with UnixFS v1 files encoded in protobuf.

`go-ipld-prime-proto` is primarily used to run selectors against Dag Protobuf encoded IPLD Graphs. It does not include robust facilities to support actual creation of DAG protobuf nodes, and none of the actual features for working with files contained in UnixFS. However, it can read, traverse and copy UnixFS data, which is what you need to run selectors and graphsync against existing UnixFS data.

## Install

`go-ipld-prime-proto` requires Go >= 1.13 and can be installed using Go modules

## Usage

To run a selector traversal against a graph encoded in Dag-Protobuf, you need to configure it with a custom `NodeBuilderChooser`. The simplest way to create such a node builder chooser is to take an existing function and pass it to `AddDagPBSupportToChooser`. This will add support for Dag Protobuf and Raw node encodings, the two types required by UnixFS.

Example:

```golang

var existing traversal.NodeBuilderChooser = func(ipld.Link, ipld.LinkContext) ipld.NodeBuilder {
	return ipldfree.NodeBuilder()
}
  
var pbChooser traversal.NodeBuilderChooser = 
dagpb.AddDagPBSupportToChooser(existing)

var selector selector.Selector
var loader ipld.Loader
var unixfsCid cid.Cid

// Load the first node
unixFSRootNode, err := cidlink.Link{Cid: unixfsCid}.Load(ipld.LinkContext{}, dagpb.PBNode__NodeBuilder(), loader)

// execute the traversal
err = traversal.Progress{
  Cfg: &traversal.Config{
    LinkLoader:             loader,
    LinkNodeBuilderChooser: pbChooser,
  },
}.WalkAdv(unixFSRootNode, allSelector, func(pg traversal.Progress, nd ipld.Node, r traversal.VisitReason) error {
  // do something with your traversed nodes here
}
```

All so checkout [unixfs_test.go](./unixfs_test.go) for good example of a complete setup with UnixFS files.

## Contribute

PRs are welcome!

Small note: If editing the Readme, please conform to the [standard-readme](https://github.com/RichardLitt/standard-readme) specification.

## License

This library is dual-licensed under Apache 2.0 and MIT terms.

Copyright 2019. Protocol Labs, Inc.
