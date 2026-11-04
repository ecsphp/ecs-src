package rules

import (
	"slices"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Alias/SetTypeToCastFixer.php
//
// SetTypeToCast replaces a statement "settype($v, 'type');" with a cast
// assignment "$v = (type) $v;" ("$v = null;" for "null"), risky because
// settype may be overridden. Only global settype calls used as a statement,
// on a plain variable and with a literal known type, are converted.
type SetTypeToCast struct{}

func (SetTypeToCast) Name() string {
	return `PhpCsFixer\Fixer\Alias\SetTypeToCastFixer`
}

func (SetTypeToCast) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Alias/SetTypeToCastFixer.php"
}

var setTypeCasts = map[string]string{
	"array":   "(array)",
	"bool":    "(bool)",
	"boolean": "(bool)",
	"double":  "(float)",
	"float":   "(float)",
	"int":     "(int)",
	"integer": "(int)",
	"object":  "(object)",
	"string":  "(string)",
}

// IsCandidate rules out a file without the trigger: only a "settype" call is replaced.
func (SetTypeToCast) IsCandidate(s *tokens.Stream) bool {
	return s.Index().HasWord("settype")
}

func (SetTypeToCast) Fix(s *tokens.Stream) bool {
	// indices below assume no cleared placeholders, as after PHP-CS-Fixer's
	// clearEmptyTokens between fixers
	s.ClearEmptyTokens()
	type call struct{ fn, open, close int }
	var candidates []call
	for i := 1; i < s.Len(); i++ {
		fn, open, closeIdx, ok := aliasFind(s, "settype", i, s.Len())
		if !ok {
			break
		}
		i = open
		candidates = append(candidates, call{fn, open, closeIdx})
	}

	changed := false
	for _, c := range slices.Backward(candidates) {
		if c.close < 0 {
			continue
		}
		arguments := aliasArguments(s, c.open, c.close)
		if len(arguments) != 2 {
			continue // function must be overridden or used incorrectly
		}
		prev := aliasPrevMeaningful(s, c.fn)
		if !isPunctVal(s, prev, ";") && !isPunctVal(s, prev, "{") && !isPunctVal(s, prev, "}") && !aliasIsOpenTag(s, prev) {
			continue // return value of the function is used
		}

		// the first argument must be a variable (settype takes it by reference)
		firstArgumentStart := arguments[0][0]
		if aliasIsMeaningless(s, firstArgumentStart) {
			firstArgumentStart = aliasNextMeaningful(s, firstArgumentStart)
		}
		if !aliasIsVariable(s, firstArgumentStart) {
			continue
		}
		commaIndex := aliasNextMeaningful(s, firstArgumentStart)
		if !isPunctVal(s, commaIndex, ",") {
			continue // first argument is a complex statement
		}

		// the second argument must be a single literal string
		secondArgumentStart, secondArgumentEnd := arguments[1][0], arguments[1][1]
		if aliasIsMeaningless(s, secondArgumentStart) {
			secondArgumentStart = aliasNextMeaningful(s, secondArgumentStart)
		}
		if secondArgumentStart < 0 || !aliasIsConstantString(s.At(secondArgumentStart)) ||
			aliasNextMeaningful(s, secondArgumentStart) < secondArgumentEnd {
			continue
		}

		typ := strings.ToLower(strings.Trim(s.At(secondArgumentStart).Value, `"'`))
		cast, known := setTypeCasts[typ]
		if typ != "null" && !known {
			continue // we don't know how to map
		}

		argument := s.At(firstArgumentStart)
		setTypeRemoveCall(s, c.fn, c.open, firstArgumentStart, commaIndex, secondArgumentStart, c.close)
		var inserted []token.Token
		if typ == "null" {
			inserted = []token.Token{argument, aliasSpace(), aliasPunct("="), aliasSpace(), {Kind: token.Ident, Value: "null"}}
		} else {
			inserted = []token.Token{argument, aliasSpace(), aliasPunct("="), aliasSpace(), aliasPunct(cast), aliasSpace(), argument}
		}
		s.InsertSliceAt(c.fn, inserted)
		aliasRemoveTrailingWhitespace(s, c.fn+len(inserted)-1)
		changed = true
	}
	if changed {
		aliasRelex(s)
	}
	return changed
}

func setTypeRemoveCall(s *tokens.Stream, fn, open, firstArgumentStart, comma, secondArgumentStart, closeIdx int) {
	clearTokenAndMergeWS(s, closeIdx)
	if p := aliasPrevMeaningful(s, closeIdx); isPunctVal(s, p, ",") {
		clearTokenAndMergeWS(s, p)
	}
	clearTokenAndMergeWS(s, secondArgumentStart)
	clearTokenAndMergeWS(s, comma)
	clearTokenAndMergeWS(s, firstArgumentStart)
	clearTokenAndMergeWS(s, open)
	clearTokenAt(s, fn)
	// we'll be inserting here so no need to merge the space tokens
	s.ClearEmptyTokens()
}

// aliasRemoveTrailingWhitespace ports Tokens::removeTrailingWhitespace: drop the
// whitespace token after index, keeping the line break that ends a "//" or "#"
// comment right before it.
func aliasRemoveTrailingWhitespace(s *tokens.Stream, index int) {
	ws := nonEmptySibling(s, index, 1)
	if ws < 0 || s.At(ws).Kind != token.Whitespace {
		return
	}
	newContent, check := "", s.At(ws).Value
	if p := ws - 1; p >= 0 && s.At(p).Kind == token.Comment && !strings.HasPrefix(s.At(p).Value, "/*") {
		nl := aliasLeadingLineBreak(check)
		if nl == "" {
			// no leading line break: Preg::split yields no parts and the
			// whitespace is dropped whole
			clearTokenAt(s, ws)
			return
		}
		newContent, check = nl, check[len(nl):]
		if check == "" {
			return
		}
	}
	if strings.Trim(check, " \t\n\r\x00\v") != "" {
		return
	}
	if newContent == "" {
		clearTokenAt(s, ws)
	} else {
		s.SetValue(ws, newContent)
	}
}

// aliasLeadingLineBreak returns the \R line break that starts v, or "".
func aliasLeadingLineBreak(v string) string {
	switch {
	case strings.HasPrefix(v, "\r\n"):
		return "\r\n"
	case strings.HasPrefix(v, "\n"), strings.HasPrefix(v, "\r"), strings.HasPrefix(v, "\v"), strings.HasPrefix(v, "\f"):
		return v[:1]
	}
	return ""
}
