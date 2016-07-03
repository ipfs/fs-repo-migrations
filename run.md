# Running Repo Migrations

How to run [fs-repo](https://github.com/ipfs/specs/tree/master/repo/fs-repo) migrations for [ipfs](https://github.com/ipfs/ipfs).

We have changed the internal, on-disk format we use to store data. In order to avoid losing your data, we're taking extra care to provide a stable migration tool that upgrades old versions of the repo to the new ones. You'll need to run the migration if you find an error like this:

```
> ipfs daemon
Error: ipfs repo found in old '.go-ipfs' location, please run migration tool.
Please see https://github.com/ipfs/fs-repo-migrations/blob/master/run.md
```

Soon, we hope to run these entirely automatically. But for now, we ask you to run these manually in case something goes wrong. It's very easy. See the quick steps below. If you run into any trouble, please feel free to open an issue in this repository: [issues](https://github.com/ipfs/fs-repo-migrations/issues).

## Step 0. Back up your repo (optional)

The migration tool is safe -- it should not delete any data. If you have important data stored _only_ in your ipfs node, and want to be extra safe, you can back up the whole repo with:

```sh
# version 0
cp -r ~/.go-ipfs ~/.go-ipfs.bak

# version 1+
cp -r ~/.ipfs ~/.ipfs.bak
```

## Step 1. Downloading the Migration

- If you have Go installed: `go get -u github.com/ipfs/fs-repo-migrations`
- Otherwise, download a prebuilt binary from [the distributions page](https://dist.ipfs.io/#fs-repo-migrations)

## Step 2. Run the Migration

Now, run the migration tool.

```sh
# if you installed from Go, tool is in your global $PATH
fs-repo-migrations

# otherwise, unzip the package, cd into it and run the binary:
./fs-repo-migrations
```


## Step 3. Done! Run IPFS.

If the migration completed without error, then you're done! Try running the new ipfs:

```
ipfs
```
