package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Strict/StrictParamFixer.php
//
// StrictParam fills in the missing `$strict` argument for a handful of functions
// (array_keys, array_search, base64_decode, in_array, mb_detect_encoding) so
// they are called with strict comparison. Only global calls are touched, and a
// call that still lacks an earlier required argument is left alone. Risky.
type StrictParam struct{}

func (StrictParam) Name() string {
	return `PhpCsFixer\Fixer\Strict\StrictParamFixer`
}

func (StrictParam) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Strict/StrictParamFixer.php"
}

func strictParamTrue() []token.Token {
	return []token.Token{{Kind: token.Ident, Value: "true"}}
}

// strictParamMap maps each handled function to its full argument list. A nil
// slot is a required argument that must already be present; a non-nil slot is
// the token(s) the fixer appends to reach the strict call.
func strictParamMap() map[string][][]token.Token {
	mbDetectOrder := []token.Token{
		{Kind: token.Ident, Value: "mb_detect_order"},
		{Kind: token.Punct, Value: "("},
		{Kind: token.Punct, Value: ")"},
	}
	return map[string][][]token.Token{
		"array_keys":         {nil, nil, strictParamTrue()},
		"array_search":       {nil, nil, strictParamTrue()},
		"base64_decode":      {nil, strictParamTrue()},
		"in_array":           {nil, nil, strictParamTrue()},
		"mb_detect_encoding": {nil, mbDetectOrder, strictParamTrue()},
	}
}

func (StrictParam) Fix(s *tokens.Stream) bool {
	paramMap := strictParamMap()
	changed := false
	for index := s.Len() - 1; index >= 0; index-- {
		if s.At(index).Kind != token.Ident {
			continue
		}
		open := aliasNextMeaningful(s, index)
		if !isPunctVal(s, open, "(") {
			continue
		}
		slots, ok := paramMap[strings.ToLower(s.At(index).Value)]
		if !ok {
			continue
		}
		if !aliasIsGlobalFunctionCall(s, index) {
			continue
		}
		if strictParamFixFunction(s, open, slots) {
			changed = true
		}
	}
	return changed
}

// strictParamFixFunction appends the missing arguments described by slots to the
// call whose "(" is at open, mirroring StrictParamFixer::fixFunction.
func strictParamFixFunction(s *tokens.Stream, open int, slots [][]token.Token) bool {
	closeIdx := s.MatchForward(open)
	if closeIdx < 0 {
		return false
	}
	have := strictParamCount(s, open, closeIdx)
	if have == len(slots) {
		return false
	}
	var insert []token.Token
	for i := have; i < len(slots); i++ {
		if slots[i] == nil {
			return false // an earlier required argument is absent
		}
		insert = append(insert, token.Token{Kind: token.Punct, Value: ","}, aliasSpace())
		insert = append(insert, slots[i]...)
	}
	beforeEnd := aliasPrevMeaningful(s, closeIdx)
	if isPunctVal(s, beforeEnd, ",") {
		// a trailing comma: drop the leading comma and append one instead
		insert = append(insert[1:], token.Token{Kind: token.Punct, Value: ","})
	}
	s.InsertSliceAt(beforeEnd+1, insert)
	return true
}

// strictParamCount counts the top-level arguments between the parentheses,
// skipping nested parentheses and bracket blocks, like StrictParamFixer does.
func strictParamCount(s *tokens.Stream, open, closeIdx int) int {
	count := 0
	expectParam := true
	for i := open + 1; i < closeIdx; i++ {
		t := s.At(i)
		if expectParam && t.Value != "" && t.Kind != token.Whitespace && t.Kind != token.Comment && t.Kind != token.DocComment {
			count++
			expectParam = false
		}
		if t.Kind == token.Punct && (t.Value == "(" || t.Value == "[") {
			if end := s.MatchForward(i); end >= 0 {
				i = end
			}
			continue
		}
		if t.Kind == token.Punct && t.Value == "," {
			expectParam = true
		}
	}
	return count
}
