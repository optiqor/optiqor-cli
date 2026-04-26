package style

import (
	"os"

	"github.com/mattn/go-isatty"
)

func isTTY(f *os.File) bool {
	if f == nil {
		return false
	}
	return isatty.IsTerminal(f.Fd()) || isatty.IsCygwinTerminal(f.Fd())
}
