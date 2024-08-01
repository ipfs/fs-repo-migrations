#!/bin/sh

set -x

echo "Migration 15 to 16" &&
cp -r repo-v15 repo-test && # init repo
go run .. -verbose -path=repo-test && # run forward migration
diff -r repo-v16 repo-test && # check forward migration against expected state
echo "Revert 16 to 15" &&
go run .. -verbose -revert -path=repo-test && # run backward migration
diff -r repo-v15 repo-test # check that after backward migration everything is back to how it used to be

FINISH="$?" # save exit code

rm -r repo-test # cleanup

exit "$FINISH" # forward exit code
