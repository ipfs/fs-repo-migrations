package isatty_test

import (
	"fmt"
	"os"

	"gx/ipfs/QmRRr1zSEeFmjPWJeDAdhhQBRM2kYuPFC4T1QVwXKg7UrG/go-isatty"
)

func Example() {
	if isatty.IsTerminal(os.Stdout.Fd()) {
		fmt.Println("Is Terminal")
	} else if isatty.IsCygwinTerminal(os.Stdout.Fd()) {
		fmt.Println("Is Cygwin/MSYS2 Terminal")
	} else {
		fmt.Println("Is Not Terminal")
	}
}
