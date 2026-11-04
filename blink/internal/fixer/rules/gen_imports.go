package rules

import (
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Import/NoUnneededImportAliasFixer.php
//
// NoUnneededImportAlias drops an alias that equals the imported name's last
// segment: "use App\Foo as Foo;" -> "use App\Foo  ;" (case-sensitive). Like
// PHP-CS-Fixer it merges the whitespace around "as" and the alias, and leaves a
// trait adaptation ("use T { foo as foo; }") alone.
type NoUnneededImportAlias struct{}

func (NoUnneededImportAlias) Name() string {
	return `PhpCsFixer\Fixer\Import\NoUnneededImportAliasFixer`
}

func (NoUnneededImportAlias) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Import/NoUnneededImportAliasFixer.php"
}

func (NoUnneededImportAlias) Fix(s *tokens.Stream) bool {
	changed := false
	for i := s.Len() - 1; i >= 0; i-- {
		if !kwIs(s.At(i), "as") {
			continue
		}
		alias := sigNext(s, i)
		if alias < 0 || s.At(alias).Kind != token.Ident {
			continue
		}
		imp := sigPrev(s, i)
		if imp < 0 || s.At(imp).Kind != token.Ident || s.At(imp).Value != s.At(alias).Value {
			continue
		}
		j := sigPrev(s, imp)
		for j >= 0 && (s.At(j).Kind == token.Ident || kwIs(s.At(j), "as") || isPunctVal(s, j, `\`) || isPunctVal(s, j, ",")) {
			j = sigPrev(s, j)
		}
		if j >= 0 && (kwIs(s.At(j), "function") || kwIs(s.At(j), "const")) {
			j = sigPrev(s, j)
		}
		// "use", or the "{" of a group import "use App\{Foo, Bar}"
		if j < 0 || !kwIs(s.At(j), "use") && (!isPunctVal(s, j, "{") || !isPunctVal(s, sigPrev(s, j), `\`)) {
			continue
		}
		clearTokenAndMergeWS(s, alias)
		clearTokenAndMergeWS(s, i)
		changed = true
	}
	if changed {
		s.ClearEmptyTokens()
	}
	return changed
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/NamespaceNotation/CleanNamespaceFixer.php
//
// CleanNamespace removes whitespace and comments between the parts of a
// namespace or use path: "A \ B" -> "A\B".
type CleanNamespace struct{}

func (CleanNamespace) Name() string {
	return `PhpCsFixer\Fixer\NamespaceNotation\CleanNamespaceFixer`
}

func (CleanNamespace) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/NamespaceNotation/CleanNamespaceFixer.php"
}

func (CleanNamespace) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind != token.Keyword {
			continue
		}
		lw := strings.ToLower(t.Value)
		if lw != "namespace" && lw != "use" {
			continue
		}
		j := skipWhitespace(s, i+1)
		if lw == "use" && j < s.Len() && s.At(j).Kind == token.Punct && s.At(j).Value == "(" {
			continue // closure use
		}
		end := s.Len()
		for k := i + 1; k < s.Len(); k++ {
			if s.At(k).Kind == token.Punct && (s.At(k).Value == ";" || s.At(k).Value == "{") {
				end = k
				break
			}
		}
		if collapseNamespaceSpaces(s, i, end) {
			changed = true
		}
	}
	return changed
}

// collapseNamespaceSpaces removes whitespace/comment tokens in (from, end) that
// sit directly between two path tokens (an identifier or a "\").
func collapseNamespaceSpaces(s *tokens.Stream, from, end int) bool {
	changed := false
	k := from + 1
	for k < end {
		t := s.At(k)
		if t.Kind == token.Whitespace || t.Kind == token.Comment || t.Kind == token.DocComment {
			p := prevPathToken(s, k, from)
			n := nextPathToken(s, k, end)
			if p >= 0 && n < end && isPathToken(s.At(p)) && isPathToken(s.At(n)) {
				s.RemoveAt(k)
				end--
				changed = true
				continue
			}
		}
		k++
	}
	return changed
}

func isPathToken(t token.Token) bool {
	return t.Kind == token.Ident || (t.Kind == token.Punct && t.Value == `\`)
}

func prevPathToken(s *tokens.Stream, i, from int) int {
	for i--; i > from; i-- {
		if k := s.At(i).Kind; k != token.Whitespace && k != token.Comment && k != token.DocComment {
			return i
		}
	}
	return -1
}

func nextPathToken(s *tokens.Stream, i, end int) int {
	for i++; i < end; i++ {
		if k := s.At(i).Kind; k != token.Whitespace && k != token.Comment && k != token.DocComment {
			return i
		}
	}
	return end
}
