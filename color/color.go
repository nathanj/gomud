package color

import (
	"strings"
)

const (
	NORMAL    = "\x1b[0m"
	BLACK     = "\x1b[30m"
	RED       = "\x1b[31m"
	GREEN     = "\x1b[32m"
	YELLOW    = "\x1b[33m"
	BLUE      = "\x1b[34m"
	MAGENTA   = "\x1b[35m"
	CYAN      = "\x1b[36m"
	WHITE     = "\x1b[37m"
	B_BLACK   = "\x1b[30;1m"
	B_RED     = "\x1b[31;1m"
	B_GREEN   = "\x1b[32;1m"
	B_YELLOW  = "\x1b[33;1m"
	B_BLUE    = "\x1b[34;1m"
	B_MAGENTA = "\x1b[35;1m"
	B_CYAN    = "\x1b[36;1m"
	B_WHITE   = "\x1b[37;1m"
)

var colorTable = map[string]string{
	"@n@": NORMAL,
	"@b@": BLACK,
	"@r@": RED,
	"@g@": GREEN,
	"@y@": YELLOW,
	"@l@": BLUE,
	"@m@": MAGENTA,
	"@c@": CYAN,
	"@w@": WHITE,
	"@B@": B_BLACK,
	"@R@": B_RED,
	"@G@": B_GREEN,
	"@Y@": B_YELLOW,
	"@L@": B_BLUE,
	"@M@": B_MAGENTA,
	"@C@": B_CYAN,
	"@W@": B_WHITE,
}

func Colorize(text string) string {
	for code, color := range colorTable {
		text = strings.Replace(text, code, color, -1)
	}
	return text
}
