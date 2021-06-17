# Running Repo Migrations

This document explains how to run [fs-repo](https://github.com/ipfs/specs/tree/master/repo/fs-repo) migrations for [ipfs](https://github.com/ipfs/ipfs).

Note that running migrations is a task automatically performed by the `ipfs` when starting the `ipfs` daemon after an upgrade or running the `ipfs-update` tool, so you would normally not need to run the `fs-repo-migrations` tool.

The `fs-repo-migrations` tool comes into play when there is a change in the internal on-disk format `ipfs` uses to store data. In order to avoid losing data, this tool upgrades old versions of the repo to the new ones.

If you run into any trouble, please feel free to [open an issue in this repository](https://github.com/ipfs/fs-repo-migrations/issues).

## Step 0. Back up your repo (optional)

The migration tool is safe -- it should not delete any data. If you have important data stored _only_ in your ipfs node, and want to be extra safe, you can back up the whole repo with:

```sh
cp -r ~/.ipfs ~/.ipfs.bak
```

## Step 1. Downloading the Migration

- If you have Go installed: `go get -u github.com/ipfs/fs-repo-migrations`
- Otherwise, download a prebuilt binary from [the distributions page](https://dist.ipfs.io/#fs-repo-migrations)

## Step 2. Run the Migration

Now, run the migration tool:

```sh
# if you installed from Go, the tool is in your global $PATH
fs-repo-migrations

# otherwise, unzip the package, cd into it and run the binary:
./fs-repo-migrations
```

## Step 3. Done! Run IPFS.

If the migration completed without error, then you're done! Try running the new ipfs:

```
ipfs daemon
```
