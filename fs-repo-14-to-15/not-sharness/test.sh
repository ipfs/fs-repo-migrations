#!/bin/sh

set -x

echo "Migration 14 to 15" &&
cp -r repotest-init repotest && # init repo
go run .. -verbose -path=repotest && # run forward migration
diff -r repotest-golden repotest && # check forward migration against golden
go run .. -verbose -revert -path=repotest && # run backward migration
diff -r repotest-init repotest # check that after backward migration everything is back to how it used to be

FINISH="$?" # save exit code

rm -r repotest # cleanup

exit "$FINISH" # forward exit code
