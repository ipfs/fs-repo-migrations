package mg0

import (
	"fmt"
	"os"
	"strings"

	migrate "github.com/ipfs/fs-repo-migrations/go-migrate"
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

	return nil
}

// Revert un-applies the migration in question. This should be best-effort.
// Some migrations are definitively one-way. If so, return an error.
func (m Migration) Revert(opts migrate.Options) error {
	repo := mfsr.RepoPath(opts.Path)

	if err := repo.CheckVersion("1"); err != nil {
		return err
	}

	// remove the version file
	if err := os.Remove(repo.VersionFile()); err != nil {
		return err
	}

	return nil
}
