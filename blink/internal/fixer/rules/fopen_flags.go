package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/FunctionNotation/FopenFlagsFixer.php
//
// FopenFlags normalizes the mode flags of a global fopen() call whose mode (the
// second of two to four arguments) is a single valid string literal: it drops
// "t" and, per the b_mode option (default true), ensures "b" is present or
// removed. Risky: fopen may be overridden.
type FopenFlags struct {
	BMode bool
}

func (FopenFlags) Name() string {
	return `PhpCsFixer\Fixer\FunctionNotation\FopenFlagsFixer`
}

func (FopenFlags) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/FunctionNotation/FopenFlagsFixer.php"
}

func (f FopenFlags) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["b_mode"].(bool); ok {
		f.BMode = v
	}
	return f
}

func (f FopenFlags) Fix(s *tokens.Stream) bool {
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
		if f.fixFopenFlagsToken(s, args[1][0], args[1][1]) {
			changed = true
		}
	}
}

// fixFopenFlagsToken normalizes the flags of the mode argument spanning
// [start, end] (FopenFlagsFixer::fixFopenFlagToken).
func (f FopenFlags) fixFopenFlagsToken(s *tokens.Stream, start, end int) bool {
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
	if !isValidFopenMode(mode) {
		return false
	}
	mode = strings.ReplaceAll(mode, "t", "")
	if f.BMode {
		if !strings.Contains(mode, "b") {
			mode += "b"
		}
	} else {
		mode = strings.ReplaceAll(mode, "b", "")
	}
	if fixed := prefix + quote + mode + quote; fixed != content {
		s.SetValue(flagIdx, fixed)
		return true
	}
	return false
}
