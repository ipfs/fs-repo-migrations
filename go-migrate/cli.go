package migrate

import (
	"flag"
	"fmt"
	"os"
)

type Flags struct {
	Force   bool
	Revert  bool
	Path    string // file path to migrate for fs based migrations
	Verbose bool
}

func (f *Flags) Setup() {
	flag.BoolVar(&f.Force, "f", false, "whether to force a migration (ignores warnings)")
	flag.BoolVar(&f.Revert, "revert", false, "whether to apply the migration backwards")
	flag.BoolVar(&f.Verbose, "verbose", false, "enable verbose logging")
	flag.StringVar(&f.Path, "path", "", "file path to migrate for fs based migrations")
}

func (f *Flags) Parse() {
	flag.Parse()
}

func Run(m Migration) error {
	f := Flags{}
	f.Setup()
	f.Parse()

	if f.Path == "" {
		flag.Usage()
		return nil
	}

	if !m.Reversible() {
		if f.Revert {
			return fmt.Errorf("migration %d is irreversible", m.Versions())
		}
		if !f.Force {
			return fmt.Errorf("migration %d is irreversible (use -f to proceed)", m.Versions())
		}
	}

	if f.Revert {
		return m.Revert(Options{
			Flags:   f,
			Verbose: f.Verbose,
		})
	} else {
		return m.Apply(Options{
			Flags:   f,
			Verbose: f.Verbose,
		})
	}
}

func Main(m Migration) {
	if err := Run(m); err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}
}
