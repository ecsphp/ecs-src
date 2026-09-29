package rules

import (
	"slices"
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/FunctionNotation/NoUnreachableDefaultArgumentValueFixer.php
//
// NoUnreachableDefaultArgumentValue removes default values of arguments that
// precede a required (non-default) argument, since such defaults are never used.
// A "= null" default on a non-nullable typed argument is kept, matching
// PHP-CS-Fixer.
type NoUnreachableDefaultArgumentValue struct{}

func (NoUnreachableDefaultArgumentValue) Name() string {
	return `PhpCsFixer\Fixer\FunctionNotation\NoUnreachableDefaultArgumentValueFixer`
}

func (NoUnreachableDefaultArgumentValue) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/FunctionNotation/NoUnreachableDefaultArgumentValueFixer.php"
}

type nudArg struct {
	start, end int // inclusive token range of the argument
	nameIndex  int
	variadic   bool
	equalsIdx  int // depth-0 "=" after the name, or -1
	defaultEnd int // last meaningful token of the default value
	nullDflt   bool
	hasType    bool
	nullable   bool
}

func (NoUnreachableDefaultArgumentValue) Fix(s *tokens.Stream) bool {
	changed := false
	for i := s.Len() - 1; i >= 0; i-- {
		t := s.At(i)
		if t.Kind != token.Keyword {
			continue
		}
		if !strings.EqualFold(t.Value, "function") && !strings.EqualFold(t.Value, "fn") {
			continue
		}
		if nudFixFunction(s, i) {
			changed = true
		}
	}
	return changed
}

func nudFixFunction(s *tokens.Stream, fnIndex int) bool {
	openParen := nudNextParen(s, fnIndex)
	if openParen == -1 {
		return false
	}
	closeParen := s.MatchForward(openParen)
	if closeParen == -1 {
		return false
	}
	args := nudParseArgs(s, openParen, closeParen)

	changed := false
	removeDefault := false
	for _, a := range slices.Backward(args) {
		if a.variadic {
			continue
		}
		if a.equalsIdx == -1 {
			removeDefault = true
			continue
		}
		if !removeDefault {
			continue
		}
		if a.nullDflt && a.hasType && !a.nullable {
			continue
		}
		// remove "= <value>" and the whitespace before "="
		for j := a.defaultEnd; j >= a.equalsIdx; j-- {
			s.RemoveAt(j)
		}
		if a.equalsIdx-1 >= 0 && s.At(a.equalsIdx-1).Kind == token.Whitespace {
			prevNonWs := prevSignificantIndex(s, a.equalsIdx-1)
			if prevNonWs != -1 && s.At(prevNonWs).Kind != token.Comment {
				s.RemoveAt(a.equalsIdx - 1)
			}
		}
		changed = true
	}
	return changed
}

// nudParseArgs splits the parameter list (openParen..closeParen) into arguments,
// analysing each. It runs before any mutation, so all indices are valid.
func nudParseArgs(s *tokens.Stream, openParen, closeParen int) []nudArg {
	var args []nudArg
	depth := 0
	start := openParen + 1
	flush := func(end int) {
		if start > end {
			return
		}
		// trim to a non-empty, meaningful range
		if a, ok := nudAnalyzeArg(s, start, end); ok {
			args = append(args, a)
		}
	}
	for i := openParen + 1; i < closeParen; i++ {
		t := s.At(i)
		if t.Kind == token.Punct {
			switch t.Value {
			case "(", "[", "{":
				depth++
			case ")", "]", "}":
				depth--
			case ",":
				if depth == 0 {
					flush(i - 1)
					start = i + 1
				}
			}
		}
	}
	flush(closeParen - 1)
	return args
}

func nudAnalyzeArg(s *tokens.Stream, start, end int) (nudArg, bool) {
	a := nudArg{start: start, end: end, nameIndex: -1, equalsIdx: -1}
	depth := 0
	for i := start; i <= end; i++ {
		t := s.At(i)
		if t.Kind == token.Punct {
			switch t.Value {
			case "(", "[", "{":
				depth++
				continue
			case ")", "]", "}":
				depth--
				continue
			}
		}
		if depth != 0 {
			continue
		}
		if t.Kind == token.Variable && a.nameIndex == -1 && a.equalsIdx == -1 {
			a.nameIndex = i
			continue
		}
		if t.Kind == token.Punct && t.Value == "=" && a.nameIndex != -1 && a.equalsIdx == -1 {
			a.equalsIdx = i
			continue
		}
	}
	if a.nameIndex == -1 {
		return a, false
	}

	// variadic: "..." right before the name
	prev := nudPrevMeaningfulFrom(s, a.nameIndex, start)
	if prev != -1 && s.At(prev).Kind == token.Punct && s.At(prev).Value == "..." {
		a.variadic = true
	}

	// type: meaningful tokens before the name, excluding "&" and "..."
	for i := start; i < a.nameIndex; i++ {
		t := s.At(i)
		if t.Kind == token.Whitespace || t.Kind == token.Comment || t.Kind == token.DocComment {
			continue
		}
		if t.Kind == token.Punct && (t.Value == "&" || t.Value == "...") {
			continue
		}
		a.hasType = true
		if t.Kind == token.Punct && t.Value == "?" {
			a.nullable = true
		}
		if t.Kind == token.Ident && strings.EqualFold(t.Value, "null") {
			a.nullable = true
		}
	}

	// default value range + whether it is exactly "null"
	if a.equalsIdx != -1 {
		a.defaultEnd = end
		for a.defaultEnd > a.equalsIdx {
			k := s.At(a.defaultEnd).Kind
			if k == token.Whitespace || k == token.Comment || k == token.DocComment {
				a.defaultEnd--
				continue
			}
			break
		}
		var meaningful []int
		for i := a.equalsIdx + 1; i <= end; i++ {
			k := s.At(i).Kind
			if k == token.Whitespace || k == token.Comment || k == token.DocComment {
				continue
			}
			meaningful = append(meaningful, i)
		}
		if len(meaningful) == 1 {
			v := s.At(meaningful[0])
			if (v.Kind == token.Ident || v.Kind == token.Keyword) && strings.EqualFold(v.Value, "null") {
				a.nullDflt = true
			}
		}
	}
	return a, true
}

func nudNextParen(s *tokens.Stream, index int) int {
	for j := index + 1; j < s.Len(); j++ {
		if s.At(j).Kind == token.Punct && s.At(j).Value == "(" {
			return j
		}
	}
	return -1
}

func nudPrevMeaningfulFrom(s *tokens.Stream, i, low int) int {
	for j := i - 1; j >= low; j-- {
		k := s.At(j).Kind
		if k == token.Whitespace || k == token.Comment || k == token.DocComment {
			continue
		}
		return j
	}
	return -1
}
