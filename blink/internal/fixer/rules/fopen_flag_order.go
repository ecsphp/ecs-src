package rules

import (
	"slices"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/FunctionNotation/FopenFlagOrderFixer.php
//
// FopenFlagOrder sorts the mode flags of a global fopen() call whose mode (the
// second of two to four arguments) is a single valid string literal, putting
// "t" and then "b" last: fopen($f, 'br+') becomes fopen($f, 'r+b'). Risky:
// fopen may be overridden.
type FopenFlagOrder struct{}

func (FopenFlagOrder) Name() string {
	return `PhpCsFixer\Fixer\FunctionNotation\FopenFlagOrderFixer`
}

func (FopenFlagOrder) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/FunctionNotation/FopenFlagOrderFixer.php"
}

func (FopenFlagOrder) Fix(s *tokens.Stream) bool {
	changed := false
	index := 0
	for {
		_, open, closeIdx, ok := aliasFind(s, "fopen", index, s.Len()-1)
		if !ok || closeIdx < 0 {
			return changed
		}
		index = open
		args := aliasArguments(s, open, closeIdx)
		if len(args) < 2 || len(args) > 4 {
			continue
		}
		if fixFopenFlag(s, args[1][0], args[1][1]) {
			changed = true
		}
	}
}

// fixFopenFlag sorts the flags of the mode argument spanning [start, end]
// (AbstractFopenFlagFixer + FopenFlagOrderFixer::fixFopenFlagToken).
func fixFopenFlag(s *tokens.Stream, start, end int) bool {
	flagIdx := -1
	for i := start; i <= end; i++ {
		k := s.At(i).Kind
		if k == token.Whitespace || k == token.Comment || k == token.DocComment {
			continue
		}
		if flagIdx >= 0 {
			return false // several meaningful tokens: not a candidate
		}
		if !aliasIsConstantString(s.At(i)) {
			return false
		}
		flagIdx = i
	}
	if flagIdx < 0 {
		return false
	}
	content := s.At(flagIdx).Value
	prefix := ""
	if content[0] == 'b' || content[0] == 'B' {
		prefix = content[:1] // a binary string
	}
	quote := content[len(prefix) : len(prefix)+1]
	mode := content[len(prefix)+1 : len(content)-1]
	if len(mode) < 2 || !isValidFopenMode(mode) {
		return false
	}
	flags := splitFopenFlags(mode)
	slices.SortStableFunc(flags, compareFopenFlags)
	if fixed := prefix + quote + strings.Join(flags, "") + quote; fixed != content {
		s.SetValue(flagIdx, fixed)
		return true
	}
	return false
}

// isValidFopenMode ports AbstractFopenFlagFixer::isValidModeString: each of
// the flags a, b, c, e, r, t, w, x at most once, "+" only after a, c, r, w, x.
func isValidFopenMode(mode string) bool {
	if len(mode) < 1 || len(mode) > 13 { // 13 === length 'r+w+a+x+c+etb'
		return false
	}
	valid := map[byte]bool{'a': true, 'b': true, 'c': true, 'e': true, 'r': true, 't': true, 'w': true, 'x': true}
	if !valid[mode[0]] {
		return false
	}
	delete(valid, mode[0])
	for i := 1; i < len(mode); i++ {
		if valid[mode[i]] {
			delete(valid, mode[i])
			continue
		}
		if mode[i] != '+' || !strings.ContainsRune("acrwx", rune(mode[i-1])) {
			return false
		}
	}
	return true
}

// splitFopenFlags splits a mode into flags, each with its "+" if it has one
// (Preg::split('#([^\+]\+?)#', ...) with empty pieces dropped).
func splitFopenFlags(mode string) []string {
	var flags []string
	for i := 0; i < len(mode); i++ {
		if i+1 < len(mode) && mode[i+1] == '+' {
			flags = append(flags, mode[i:i+2])
			i++
			continue
		}
		flags = append(flags, mode[i:i+1])
	}
	return flags
}

// compareFopenFlags orders "b" last, "t" just before it, the rest by value.
func compareFopenFlags(a, b string) int {
	switch {
	case a == b:
		return 0
	case a == "b":
		return 1
	case b == "b":
		return -1
	case a == "t":
		return 1
	case b == "t":
		return -1
	case a < b:
		return -1
	}
	return 1
}
