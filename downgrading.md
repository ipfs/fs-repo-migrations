## Automatically using `ipfs-update`

If you want to do a downgrade (and not a clean install of an older version) you can revert your repo using the fs-migration tools:

```sh
$ ipfs-2-to-3 -revert -path=$IPFS_PATH
```

That should be it. You're all set.

## Manually downgrading IPFS versions

**You should not have to do this manually. Automatically downgrading is the preferred method.**

Manually downgrading your IPFS version (for instance, from `0.4.0` to `0.3.8`) - which would involve keeping your data - is currently **not possible**. You can't use the same repository simultaneously for both versions. Here is how you switch. However, you can reinstall from scratch.

First, go to your repo for IPFS, check out the right branch, and re-install.

```sh
cd $GOPATH/src/github.com/ipfs/go-ipfs
git checkout master # (or the version you want)
make install
```

Then to backup your repo:

```sh
mv ~/.ipfs ~/.ipfs.bak
```

Finally, run:

```sh
ipfs init
ipfs daemon
```

And you should be all set.