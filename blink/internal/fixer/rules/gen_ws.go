package rules

import (
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Basic/EncodingFixer.php
//
// Encoding removes a leading UTF-8 BOM (EF BB BF) from the start of the file.
type Encoding struct{}

const utf8BOM = "\uFEFF"

func (Encoding) Name() string {
	return `PhpCsFixer\Fixer\Basic\EncodingFixer`
}

func (Encoding) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Basic/EncodingFixer.php"
}

func (Encoding) Fix(s *tokens.Stream) bool {
	if s.Len() == 0 {
		return false
	}
	v := s.At(0).Value
	if !strings.HasPrefix(v, utf8BOM) {
		return false
	}
	rest := v[len(utf8BOM):]
	if rest == "" {
		s.RemoveAt(0)
	} else {
		s.SetValue(0, rest)
	}
	return true
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/LanguageConstruct/DeclareParenthesesFixer.php
//
// DeclareParentheses removes whitespace around a declare() header's parentheses:
// "declare ( strict_types=1 )" -> "declare(strict_types=1)".
type DeclareParentheses struct{}

func (DeclareParentheses) Name() string {
	return `PhpCsFixer\Fixer\LanguageConstruct\DeclareParenthesesFixer`
}

func (DeclareParentheses) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/LanguageConstruct/DeclareParenthesesFixer.php"
}

func (DeclareParentheses) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind != token.Keyword || !strings.EqualFold(t.Value, "declare") {
			continue
		}
		op := i + 1
		wsBetween := -1
		if op < s.Len() && s.At(op).Kind == token.Whitespace {
			wsBetween = op
			op++
		}
		if op >= s.Len() || s.At(op).Kind != token.Punct || s.At(op).Value != "(" {
			continue
		}
		cp := s.MatchForward(op)
		if cp < 0 {
			continue
		}
		// collect whitespace to drop, delete high-to-low to keep indices valid
		var drop []int
		if cp-1 > op && s.At(cp-1).Kind == token.Whitespace {
			drop = append(drop, cp-1)
		}
		if op+1 < cp && s.At(op+1).Kind == token.Whitespace {
			drop = append(drop, op+1)
		}
		if wsBetween >= 0 {
			drop = append(drop, wsBetween)
		}
		for j := 0; j < len(drop); j++ {
			for k := j + 1; k < len(drop); k++ {
				if drop[k] > drop[j] {
					drop[j], drop[k] = drop[k], drop[j]
				}
			}
		}
		for _, idx := range drop {
			s.RemoveAt(idx)
			changed = true
		}
	}
	return changed
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Comment/MultilineCommentOpeningClosingFixer.php
//
// MultilineCommentOpeningClosing normalizes block-comment delimiters: an opening
// with extra asterisks ("/***") collapses to "/*" (doc blocks keep their "/**"),
// and a closing with extra asterisks ("***/") collapses to "*/".
type MultilineCommentOpeningClosing struct{}

func (MultilineCommentOpeningClosing) Name() string {
	return `PhpCsFixer\Fixer\Comment\MultilineCommentOpeningClosingFixer`
}

func (MultilineCommentOpeningClosing) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Comment/MultilineCommentOpeningClosingFixer.php"
}

func (MultilineCommentOpeningClosing) Fix(s *tokens.Stream) bool {
	changed := false
	for i := range s.Len() {
		t := s.At(i)
		if t.Kind != token.Comment && t.Kind != token.DocComment {
			continue
		}
		v := t.Value
		if !strings.HasPrefix(v, "/*") {
			continue
		}
		nv := v
		// the opening fix applies to plain comments only, never to doc blocks
		if !isDocBlockOpening(nv) {
			nv = fixCommentOpening(nv)
		}
		nv = fixCommentClosing(nv)
		if nv != v {
			s.SetValue(i, nv)
			changed = true
		}
	}
	return changed
}

// isDocBlockOpening mirrors PHP's tokenizer: "/**" followed by a non-asterisk,
// non-slash character is a doc block; "/***" or "/**/" is a plain comment.
func isDocBlockOpening(v string) bool {
	return len(v) >= 4 && v[:3] == "/**" && v[3] != '*' && v[3] != '/'
}

// fixCommentOpening implements /^\/\*{2,}(?!\/)/ -> "/*".
func fixCommentOpening(v string) string {
	if !strings.HasPrefix(v, "/*") {
		return v
	}
	a := 0
	for 1+a < len(v) && v[1+a] == '*' {
		a++
	}
	if a < 2 {
		return v
	}
	var next byte
	if 1+a < len(v) {
		next = v[1+a]
	}
	if next != '/' {
		return "/*" + v[1+a:]
	}
	if a >= 3 {
		return "/*" + v[a:]
	}
	return v
}

// fixCommentClosing implements /(?<!\/)\*{2,}\/$/ -> "*/".
func fixCommentClosing(v string) string {
	if !strings.HasSuffix(v, "/") {
		return v
	}
	k := len(v) - 2
	b := 0
	for k >= 0 && v[k] == '*' {
		b++
		k--
	}
	if b < 2 {
		return v
	}
	var prev byte
	if k >= 0 {
		prev = v[k]
	}
	if prev != '/' {
		return v[:k+1] + "*/"
	}
	if b >= 3 {
		return v[:k+2] + "*/"
	}
	return v
}
