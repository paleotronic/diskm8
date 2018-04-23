package disk

import (
	"strings"
	"testing"
)

func TestHGR2Tokenise(t *testing.T) {

	lines := []string{
		"10 HGR2 : REM SOMETHING",
		"20 REM SOMETHING ELSE",
	}

	a := ApplesoftTokenize(lines)

	s := string(ApplesoftDetoks(a))

	t.Logf("code: %s", s)

	if !strings.Contains(s, "HGR2 ") {
		t.Fatalf("Expected HGR2")
	}

}

func TestHGRTokenise(t *testing.T) {

	lines := []string{
		"10 HGR : REM SOMETHING",
		"20 REM SOMETHING ELSE",
	}

	a := ApplesoftTokenize(lines)

	s := string(ApplesoftDetoks(a))

	t.Logf("code: %s", s)

	if !strings.Contains(s, "HGR ") {
		t.Fatalf("Expected HGR")
	}

}
