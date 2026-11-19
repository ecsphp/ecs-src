package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PSR12/Sniffs/Files/ImportStatementSniff.php
//
// ImportStatement drops the leading backslash of an import: "use \Foo\Bar;" ->
// "use Foo\Bar;". Trait uses inside a class-like body and closure "use (...)"
// are left alone.
type ImportStatement struct{}

func (ImportStatement) Name() string {
	return `PHP_CodeSniffer\Standards\PSR12\Sniffs\Files\ImportStatementSniff`
}

func (ImportStatement) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PSR12/Sniffs/Files/ImportStatementSniff.php"
}

func (ImportStatement) Fix(s *tokens.Stream) bool {
	changed := false
	bodies := phpcsOOBodies(s)
	for i := 0; i < s.Len(); i++ {
		if t := s.At(i); t.Kind != token.Keyword || !strings.EqualFold(t.Value, "use") || phpcsInRanges(bodies, i) {
			continue
		}
		next := sigNext(s, i)
		if next < 0 {
			continue
		}
		if n := s.At(next); n.Kind == token.Keyword && (strings.EqualFold(n.Value, "function") || strings.EqualFold(n.Value, "const")) {
			next = sigNext(s, next)
		}
		if next < 0 || next+1 >= s.Len() || s.At(next).Kind != token.Punct || s.At(next).Value != `\` || s.At(next+1).Kind != token.Ident {
			continue
		}
		s.RemoveAt(next)
		bodies = phpcsOOBodies(s)
		changed = true
	}
	return changed
}

// phpcsOOBodies returns the [open, close] brace pairs of every class, anonymous
// class, interface, trait and enum body: PHP_CodeSniffer's OO scopes.
func phpcsOOBodies(s *tokens.Stream) [][2]int {
	var bodies [][2]int
	for i := range s.Len() {
		if open := phpcsOOScopeOpener(s, i); open >= 0 {
			if closeIdx := s.MatchForward(open); closeIdx >= 0 {
				bodies = append(bodies, [2]int{open, closeIdx})
			}
		}
	}
	return bodies
}

// phpcsOOScopeOpener returns the "{" opening the body of the class-like keyword
// at i, or -1 when i is not one (or it has no body yet).
func phpcsOOScopeOpener(s *tokens.Stream, i int) int {
	if !cdIsClassy(s, i) {
		return -1
	}
	for j := i + 1; j < s.Len(); j++ {
		t := s.At(j)
		if t.Kind != token.Punct {
			continue
		}
		switch t.Value {
		case "(":
			if m := s.MatchForward(j); m >= 0 {
				j = m
				continue
			}
			return -1
		case "{":
			return j
		case ";", "}":
			return -1
		}
	}
	return -1
}

func phpcsInRanges(ranges [][2]int, i int) bool {
	for _, r := range ranges {
		if i > r[0] && i < r[1] {
			return true
		}
	}
	return false
}
