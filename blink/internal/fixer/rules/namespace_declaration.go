package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PSR2/Sniffs/Namespaces/NamespaceDeclarationSniff.php
//
// NamespaceDeclaration puts exactly one blank line after a namespace declaration.
type NamespaceDeclaration struct{}

func (NamespaceDeclaration) Name() string {
	return `PHP_CodeSniffer\Standards\PSR2\Sniffs\Namespaces\NamespaceDeclarationSniff`
}

func (NamespaceDeclaration) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PSR2/Sniffs/Namespaces/NamespaceDeclarationSniff.php"
}

func (NamespaceDeclaration) Fix(s *tokens.Stream) bool {
	changed := false
	for ns := 0; ns < s.Len(); ns++ {
		if t := s.At(ns); t.Kind == token.Keyword && strings.EqualFold(t.Value, "namespace") {
			if namespaceBlankLineAfter(s, namespaceStatementEnd(s, ns)) {
				changed = true
			}
		}
	}
	return changed
}

// namespaceStatementEnd mirrors File::findEndOfStatement for a namespace: its
// ";", the "}" closing a braced namespace, or the last code before a PHP tag.
func namespaceStatementEnd(s *tokens.Stream, ns int) int {
	last := ns
	for i := ns + 1; i < s.Len(); i++ {
		t := s.At(i)
		switch {
		case t.Kind == token.Punct && t.Value == ";":
			return i
		case t.Kind == token.Punct && t.Value == "{":
			if c := s.MatchForward(i); c >= 0 {
				return c
			}
			return -1
		case t.Kind == token.OpenTag || t.Kind == token.CloseTag:
			return last
		case !isPhpcsEmpty(t):
			last = i
		}
	}
	return -1
}

// namespaceBlankLineAfter works on PHP_CodeSniffer's whitespace pieces (one per
// line): from the first piece on a line after the statement end up to the next
// code, exactly one line break must remain.
func namespaceBlankLineAfter(s *tokens.Stream, end int) bool {
	if end < 0 {
		return false
	}
	// locate the first piece that starts on a later line than end
	iTok, iOff, onEndLine := -1, 0, true
	for t := end + 1; t < s.Len() && iTok < 0; t++ {
		v := s.At(t).Value
		if s.At(t).Kind != token.Whitespace {
			if !onEndLine {
				iTok = t
			} else if strings.Contains(v, "\n") {
				onEndLine = false
			}
			continue
		}
		for off := 0; off < len(v); {
			if !onEndLine {
				iTok, iOff = t, off
				break
			}
			nl := strings.IndexByte(v[off:], '\n')
			if nl < 0 {
				break
			}
			off += nl + 1
			onEndLine = false
			if off == len(v) {
				break
			}
		}
	}
	if iTok < 0 {
		return false
	}
	next := iTok
	if iOff > 0 || s.At(iTok).Kind == token.Whitespace {
		next = iTok + 1
		for next < s.Len() && s.At(next).Kind == token.Whitespace {
			next++
		}
		if next >= s.Len() {
			return false
		}
	}
	var between strings.Builder
	if next != iTok {
		between.WriteString(s.At(iTok).Value[iOff:])
		for t := iTok + 1; t < next; t++ {
			between.WriteString(s.At(t).Value)
		}
	}
	gap := between.String()
	diff := strings.Count(gap, "\n")
	if diff == 1 {
		return false
	}
	eol := phpcsEolChar(s)
	if next == iTok {
		if p := iTok - 1; p >= 0 && s.At(p).Kind == token.Whitespace {
			s.SetValue(p, s.At(p).Value+eol)
		} else {
			s.InsertAt(iTok, token.Token{Kind: token.Whitespace, Value: eol})
		}
		return true
	}
	repl := eol + gap
	if diff > 1 {
		repl = eol + gap[strings.LastIndexByte(gap, '\n')+1:]
	}
	s.SetValue(iTok, s.At(iTok).Value[:iOff]+repl)
	for t := next - 1; t > iTok; t-- {
		s.RemoveAt(t)
	}
	return true
}
