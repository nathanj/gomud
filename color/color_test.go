package color

import (
	"testing"
)

var colortests = []struct {
	in  string
	out string
}{
	{"@r@Hello @g@There@n@", "\x1b[31mHello \x1b[32mThere\x1b[0m"},
}

func TestColorize(t *testing.T) {
	for i, tt := range colortests {
		s := Colorize(tt.in)
		if s != tt.out {
			t.Errorf("%d. Colorize(%q) => %q, want %q", i, tt.in, s, tt.out)
		}
	}
}
