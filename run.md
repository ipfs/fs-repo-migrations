# Running Repo Migrations

How to run [fs-repo](https://github.com/ipfs/specs/tree/master/repo/fs-repo) migrations for [ipfs](https://github.com/ipfs/ipfs).

We have changed the internal, on-disk format we use to store data. In order to avoid losing your data, we're taking extra care to provide a stable migration tool that upgrades old versions of the repo to the new ones. You'll know you need to run the migration if you find an error like this:

```
> ipfs daemon
Error: ipfs repo found in old '.go-ipfs' location, please run migration tool.
Please see https://github.com/ipfs/fs-repo-migrations/blob/master/run.md
```

Soon, we hope to run these entirely automatically. But for now, we ask you to run these manually in case something goes wrong. It's very easy. See the quick steps below. If you run into any trouble, please feel free to open an issue in this repository: [issues](https://github.com/ipfs/fs-repo-migrations/issues).

## Step 0. Back up your repo (optional)

The migration tool is safe-- it should not delete any data. Just to be safe though -- if you _do_ have important data stored only in your ipfs node -- back up the repo just in case:

```
# version 0
cp -r ~/.go-ipfs ~/.go-ipfs.bak

# version 1+
cp -r ~/.ipfs ~/.ipfs.bak
```

## Step 1. Downloading the Migration

- If you have Go installed: `go get github.com/ipfs/fs-repo-migrations`
- Otherwise, download a prebuilt binary:
  - [Mac OSX](https://gobuilder.me/get/github.com/ipfs/fs-repo-migrations/fs-repo-migrations_master_darwin-amd64.zip)
  - [Linux 32bit](https://gobuilder.me/get/github.com/ipfs/fs-repo-migrations/fs-repo-migrations_master_linux-386.zip)
  - [Linux 64bit](https://gobuilder.me/get/github.com/ipfs/fs-repo-migrations/fs-repo-migrations_master_linux-amd64.zip)
  - [Linux ARM](https://gobuilder.me/get/github.com/ipfs/fs-repo-migrations/fs-repo-migrations_master_linux-arm.zip)
  - [Windows 32bit](https://gobuilder.me/get/github.com/ipfs/fs-repo-migrations/fs-repo-migrations_master_windows-386.zip)
  - [Windows 64bit](https://gobuilder.me/get/github.com/ipfs/fs-repo-migrations/fs-repo-migrations_master_windows-amd64.zip)
  - [See more available builds](https://gobuilder.me/github.com/ipfs/fs-repo-migrations)

## Step 2. Run the Migration

Now, run the migration tool. (Note: if you installed from Go, the tool is in your global `$PATH`, so use `fs-repo-migrations` instead of `./fs-repo-migrations`)

```
./fs-repo-migrations
```


## Step 3. Done! Run IPFS.

If the migration completed without error, then you're done! Try running the new ipfs:

```
ipfs
```
