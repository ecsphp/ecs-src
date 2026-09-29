package rules

import (
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Operator/TernaryToNullCoalescingFixer.php
//
// TernaryToNullCoalescing rewrites "isset($a) ? $a : $b" as "$a ?? $b" when the
// single isset argument is exactly the ternary's true branch. Multi-argument
// isset, a mismatched true branch, or a span containing comments are left alone.
type TernaryToNullCoalescing struct{}

func (TernaryToNullCoalescing) Name() string {
	return `PhpCsFixer\Fixer\Operator\TernaryToNullCoalescingFixer`
}

func (TernaryToNullCoalescing) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Operator/TernaryToNullCoalescingFixer.php"
}

func (TernaryToNullCoalescing) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind != token.Keyword || strings.ToLower(t.Value) != "isset" {
			continue
		}
		b := sigNext(s, i)
		if b < 0 || s.At(b).Kind != token.Punct || s.At(b).Value != "(" {
			continue
		}
		c := s.MatchForward(b)
		if c < 0 {
			continue
		}
		d := sigNext(s, c)
		if d < 0 || s.At(d).Kind != token.Punct || s.At(d).Value != "?" {
			continue // not a ternary (also skips "??" and "?->", which are one token)
		}
		e := ternaryColon(s, d+1)
		if e < 0 || sigNext(s, e) < 0 {
			continue // no matching ":" or no default branch
		}
		if !spanClean(s, i, e) {
			continue // a comment inside the span would be dropped
		}
		if hasTopLevelComma(s, b+1, c-1) {
			continue // isset($a, $b) - not convertible
		}
		issetToks := sigSlice(s, b+1, c-1)
		trueToks := sigSlice(s, d+1, e-1)
		if len(issetToks) == 0 || !tokensEqualAt(s, issetToks, trueToks) {
			continue
		}
		// replace "isset ( expr ) ? expr :" with "expr ?? "
		repl := trimmedCopy(s, b+1, c-1)
		repl = append(repl,
			token.Token{Kind: token.Whitespace, Value: " "},
			token.Token{Kind: token.Punct, Value: "??"},
			token.Token{Kind: token.Whitespace, Value: " "},
		)
		end := e
		if e+1 < s.Len() && s.At(e+1).Kind == token.Whitespace {
			end = e + 1 // swallow the space after ":" so spacing stays single
		}
		s.ReplaceRange(i, end, repl)
		changed = true
	}
	return changed
}

// ternaryColon returns the index of the ":" that closes the ternary begun by a
// "?" before `from`, at bracket depth 0 and accounting for nested "?:".
func ternaryColon(s *tokens.Stream, from int) int {
	depth := 0
	tern := 0
	for j := from; j < s.Len(); j++ {
		t := s.At(j)
		if t.Kind != token.Punct {
			continue
		}
		switch t.Value {
		case "(", "[", "{":
			depth++
		case ")", "]", "}":
			if depth == 0 {
				return -1
			}
			depth--
		case "?":
			if depth == 0 {
				tern++
			}
		case ":":
			if depth == 0 {
				if tern == 0 {
					return j
				}
				tern--
			}
		case ";":
			if depth == 0 {
				return -1
			}
		}
	}
	return -1
}

// hasTopLevelComma reports whether [lo, hi] holds a comma at bracket depth 0.
func hasTopLevelComma(s *tokens.Stream, lo, hi int) bool {
	depth := 0
	for j := lo; j <= hi && j < s.Len(); j++ {
		t := s.At(j)
		if t.Kind != token.Punct {
			continue
		}
		switch t.Value {
		case "(", "[", "{":
			depth++
		case ")", "]", "}":
			depth--
		case ",":
			if depth == 0 {
				return true
			}
		}
	}
	return false
}

// sigSlice returns the indices of the significant (non-trivia) tokens in [lo, hi].
func sigSlice(s *tokens.Stream, lo, hi int) []int {
	var out []int
	for j := lo; j <= hi && j < s.Len(); j++ {
		switch s.At(j).Kind {
		case token.Whitespace, token.Comment, token.DocComment:
			continue
		}
		out = append(out, j)
	}
	return out
}

// tokensEqualAt reports whether two significant-token index slices carry the
// same kinds and values.
func tokensEqualAt(s *tokens.Stream, a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for k := range a {
		ta, tb := s.At(a[k]), s.At(b[k])
		if ta.Kind != tb.Kind || ta.Value != tb.Value {
			return false
		}
	}
	return true
}

// trimmedCopy deep-copies the tokens in [lo, hi], dropping edge whitespace.
func trimmedCopy(s *tokens.Stream, lo, hi int) []token.Token {
	for lo <= hi && s.At(lo).Kind == token.Whitespace {
		lo++
	}
	for hi >= lo && s.At(hi).Kind == token.Whitespace {
		hi--
	}
	out := make([]token.Token, 0, hi-lo+1)
	for j := lo; j <= hi; j++ {
		out = append(out, s.At(j))
	}
	return out
}
