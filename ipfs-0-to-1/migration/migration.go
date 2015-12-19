package mg0

import (
	"fmt"
	"os"
	"strings"

	migrate "github.com/ipfs/fs-repo-migrations/go-migrate"
	lock "github.com/ipfs/fs-repo-migrations/ipfs-0-to-1/repolock"
	mfsr "github.com/ipfs/fs-repo-migrations/mfsr"
)

type Migration struct {
}

// Version is the int version number. This could be a string
// in future versions
func (m Migration) Versions() string {
	return "0-to-1"
}

// Reversible returns whether this migration can be reverted.
// Endeavor to make them all reversible. This is here only to warn users
// in case this is not the case.
func (m Migration) Reversible() bool {
	return true
}

// Apply applies the migration in question.
// This migration merely adds a version file.
func (m Migration) Apply(opts migrate.Options) error {
	repolk, err := lock.Lock(opts.Path)
	if err != nil {
		return err
	}
	defer repolk.Close()

	repo := mfsr.RepoPath(opts.Path)

	// first, check if there is a version file.
	// if there is, bail out.
	if v, err := repo.Version(); err == nil {
		return fmt.Errorf("repo at %s is version %s (not 0)", opts.Path, v)
	} else if !strings.Contains(err.Error(), "no version file in repo") {
		return err
	}

	// add the version file
	if err := repo.WriteVersion("1"); err != nil {
		return err
	}

	if opts.Verbose {
		fmt.Println("wrote version file")
	}

	return nil
}

// Revert un-applies the migration in question. This should be best-effort.
// Some migrations are definitively one-way. If so, return an error.
func (m Migration) Revert(opts migrate.Options) error {
	lk, err := lock.Lock(opts.Path)
	if err != nil {
		return err
	}
	defer lk.Close()

	repo := mfsr.RepoPath(opts.Path)

	repolk, err := lock.Lock(opts.Path)
	if err != nil {
		return err
	}
	defer repolk.Close()

	if err := repo.CheckVersion("1"); err != nil {
		return err
	}

	// remove the version file
	if err := os.Remove(repo.VersionFile()); err != nil {
		return err
	}
	if opts.Verbose {
		fmt.Println("deleted version file")
	}

	return nil
}
