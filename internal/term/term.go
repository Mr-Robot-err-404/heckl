package term

import "os"

const (
	Reset  = "\033[0m"
	Dim    = "\033[2m"
	Bold   = "\033[1m"
	Red    = "\033[31m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Blue   = "\033[34m"
	Cyan   = "\033[36m"
	Grey   = "\033[90m"
)

func Enabled(out *os.File) bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	info, err := out.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

type Printer struct {
	color bool
}

func NewPrinter(out *os.File) *Printer {
	return &Printer{color: Enabled(out)}
}

func (p *Printer) Paint(color, s string) string {
	if !p.color || color == "" {
		return s
	}
	return color + s + Reset
}
