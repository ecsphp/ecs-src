package rules

import (
	"fmt"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/PHP/GlobalKeywordSniff.php
//
// GlobalKeyword reports every use of the "global" keyword. It fixes nothing.
type GlobalKeyword struct{}

func (GlobalKeyword) Name() string {
	return `PHP_CodeSniffer\Standards\Squiz\Sniffs\PHP\GlobalKeywordSniff`
}

func (GlobalKeyword) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/PHP/GlobalKeywordSniff.php"
}

func (GlobalKeyword) Fix(*tokens.Stream) bool { return false }

// globalKeywordNameContexts are the keywords after which PHP_CodeSniffer's
// tokenizer makes a keyword a T_STRING (T_STRING_CONTEXTS). blink's lexer
// already handles the other contexts.
var globalKeywordNameContexts = map[string]bool{
	"class": true, "interface": true, "trait": true, "enum": true, "case": true,
	"extends": true, "implements": true, "new": true, "use": true,
	"namespace": true, "goto": true,
}

// globalKeywordIsName reports whether PHP_CodeSniffer sees the "global" at i as
// a name rather than a T_GLOBAL.
func globalKeywordIsName(s *tokens.Stream, i int) bool {
	if isPunctVal(s, sigNext(s, i), ":") {
		return true
	}
	p := sigPrev(s, i)
	if p < 0 {
		return false
	}
	t := s.At(p)
	switch {
	case t.Kind == token.Keyword && globalKeywordNameContexts[strings.ToLower(t.Value)]:
		return true
	case t.Kind == token.AttributeStart:
		return true
	case isPunctVal(s, p, "&"):
		return kwIs(s.At(max(sigPrev(s, p), 0)), "function")
	}
	return false
}

func (g GlobalKeyword) Errors(s *tokens.Stream) []fixer.CodingStandardError {
	var errs []fixer.CodingStandardError
	var encapsed []bool
	line, lineAt := 1, 0
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if !kwIs(t, "global") {
			continue
		}
		if globalKeywordIsName(s, i) {
			continue
		}
		if encapsed == nil {
			encapsed = encapsedTokens(s)
		}
		varName := strings.ReplaceAll(globalKeywordNextVariable(s, i, encapsed), "$", "")
		for ; lineAt < i; lineAt++ {
			line += strings.Count(s.At(lineAt).Value, "\n")
		}
		errs = append(errs, fixer.CodingStandardError{
			Line:    line,
			Message: fmt.Sprintf(`Use of the "global" keyword is forbidden; use "$GLOBALS['%s']" instead`, varName),
			Checker: g.Name() + ".NotAllowed",
		})
	}
	return errs
}

// globalKeywordNextVariable returns the first T_VARIABLE after i as
// PHP_CodeSniffer tokenizes it: a variable interpolated in a double-quoted
// string or heredoc is part of the string token. With none, findNext returns
// false and $tokens[false] is the file's first token.
func globalKeywordNextVariable(s *tokens.Stream, i int, encapsed []bool) string {
	for j := i + 1; j < s.Len(); j++ {
		if v := s.At(j); v.Kind == token.Variable && v.Value != "$" && !encapsed[j] {
			return v.Value
		}
	}
	if s.Len() == 0 {
		return ""
	}
	first := s.At(0)
	switch first.Kind {
	case token.OpenTag:
		return strings.TrimRight(first.Value, " \t\r\n")
	case token.InlineHTML:
		if nl := strings.IndexByte(first.Value, '\n'); nl >= 0 {
			return first.Value[:nl+1]
		}
	}
	return first.Value
}

// encapsedTokens marks the tokens inside a double-quoted string or heredoc
// that the lexer split around its interpolated parts. Each run of literal text
// is a String token that keeps the delimiters it touches ("{" and "}" of an
// interpolated expression included).
func encapsedTokens(s *tokens.Stream) []bool {
	type frame struct {
		label string // a heredoc literal's closing label, "" for a double-quoted one
		expr  bool   // an interpolated expression rather than literal text
		depth int    // braces open within an expression
	}
	in := make([]bool, s.Len())
	var stack []frame
	chunk := func(v string) {
		top := stack[len(stack)-1]
		switch {
		case literalCloses(v, top.label):
			stack = stack[:len(stack)-1]
		case strings.HasSuffix(v, "{"):
			stack = append(stack, frame{expr: true})
		}
	}
	for i := range s.Len() {
		t := s.At(i)
		in[i] = len(stack) > 0
		if len(stack) > 0 && !stack[len(stack)-1].expr {
			if t.Kind == token.String {
				chunk(t.Value)
			}
			continue
		}
		if len(stack) > 0 {
			top := &stack[len(stack)-1]
			switch {
			case isPunctVal(s, i, "{"):
				top.depth++
				continue
			case isPunctVal(s, i, "}") && top.depth > 0:
				top.depth--
				continue
			case t.Kind == token.String && top.depth == 0 && strings.HasPrefix(t.Value, "}"):
				stack = stack[:len(stack)-1]
				chunk(t.Value)
				continue
			}
		}
		if t.Kind != token.String {
			continue
		}
		if label, rest, ok := literalOpening(t.Value); ok && !literalCloses(rest, label) {
			stack = append(stack, frame{label: label})
			if strings.HasSuffix(rest, "{") {
				stack = append(stack, frame{expr: true})
			}
		}
	}
	return in
}

// literalOpening reports whether v starts an interpolating literal, with the
// heredoc's label ("" for a double-quoted string) and the rest of v.
func literalOpening(v string) (label, rest string, ok bool) {
	if len(v) > 0 && (v[0] == 'b' || v[0] == 'B') {
		v = v[1:]
	}
	switch {
	case strings.HasPrefix(v, `"`):
		return "", v[1:], true
	case strings.HasPrefix(v, "<<<"):
		nl := strings.IndexByte(v, '\n')
		if nl < 0 {
			return "", "", false
		}
		head := strings.TrimLeft(v[3:nl], " \t")
		if strings.HasPrefix(head, "'") {
			return "", "", false // a nowdoc does not interpolate
		}
		label = strings.TrimRight(strings.Trim(strings.TrimRight(head, "\r"), `"`), " \t")
		return label, v[nl:], label != ""
	}
	return "", "", false
}

// literalCloses reports whether the literal text v ends its literal.
func literalCloses(v, label string) bool {
	if label == "" {
		if !strings.HasSuffix(v, `"`) {
			return false
		}
		backslashes := 0
		for k := len(v) - 2; k >= 0 && v[k] == '\\'; k-- {
			backslashes++
		}
		return backslashes%2 == 0
	}
	body, ok := strings.CutSuffix(v, label)
	return ok && strings.HasSuffix(strings.TrimRight(body, " \t"), "\n")
}
