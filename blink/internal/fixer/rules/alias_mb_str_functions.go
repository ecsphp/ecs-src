package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Alias/MbStrFunctionsFixer.php
//
// MbStrFunctions replaces a global call to a byte-based string function with its
// multibyte equivalent (strlen -> mb_strlen, substr -> mb_substr, ...), only
// when the argument count matches the mb_ signature. A function imported with
// "use function" is renamed too. Risky. The version-gated additions
// (mb_str_pad, mb_trim, ...) are omitted; only the always-present base map is
// ported.
type MbStrFunctions struct{}

func (MbStrFunctions) Name() string {
	return `PhpCsFixer\Fixer\Alias\MbStrFunctionsFixer`
}

func (MbStrFunctions) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Alias/MbStrFunctionsFixer.php"
}

type mbStrEntry struct {
	replacement string
	counts      map[int]bool
}

func counts(ns ...int) map[int]bool {
	m := make(map[int]bool, len(ns))
	for _, n := range ns {
		m[n] = true
	}
	return m
}

var mbStrMap = map[string]mbStrEntry{
	"str_split":    {"mb_str_split", counts(1, 2, 3)},
	"stripos":      {"mb_stripos", counts(2, 3)},
	"stristr":      {"mb_stristr", counts(2, 3)},
	"strlen":       {"mb_strlen", counts(1)},
	"strpos":       {"mb_strpos", counts(2, 3)},
	"strrchr":      {"mb_strrchr", counts(2)},
	"strripos":     {"mb_strripos", counts(2, 3)},
	"strrpos":      {"mb_strrpos", counts(2, 3)},
	"strstr":       {"mb_strstr", counts(2, 3)},
	"strtolower":   {"mb_strtolower", counts(1)},
	"strtoupper":   {"mb_strtoupper", counts(1)},
	"substr":       {"mb_substr", counts(2, 3)},
	"substr_count": {"mb_substr_count", counts(2, 3, 4)},
}

func (MbStrFunctions) Fix(s *tokens.Stream) bool {
	changed := false
	for index := s.Len() - 1; index >= 0; index-- {
		if s.At(index).Kind != token.Ident {
			continue
		}
		entry, ok := mbStrMap[strings.ToLower(s.At(index).Value)]
		if !ok {
			continue
		}
		open := aliasNextMeaningful(s, index)
		if isPunctVal(s, open, "(") && aliasIsGlobalFunctionCall(s, index) {
			closeIdx := s.MatchForward(open)
			if closeIdx >= 0 && entry.counts[len(aliasArguments(s, open, closeIdx))] {
				s.SetValue(index, entry.replacement)
				changed = true
			}
			continue
		}
		if mbStrIsFunctionImport(s, index) {
			s.SetValue(index, entry.replacement)
			changed = true
		}
	}
	return changed
}

// mbStrIsFunctionImport reports whether the name at index is the imported name of
// a "use function Name;" statement (CT::T_FUNCTION_IMPORT), optionally written
// with a leading namespace separator ("use function \Name;").
func mbStrIsFunctionImport(s *tokens.Stream, index int) bool {
	p := aliasPrevMeaningful(s, index)
	if isPunctVal(s, p, `\`) {
		p = aliasPrevMeaningful(s, p)
	}
	return aliasIsKeyword(s, p, "function") && aliasIsKeyword(s, aliasPrevMeaningful(s, p), "use")
}
