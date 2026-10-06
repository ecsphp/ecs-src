package rules

import (
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Import/NoLeadingImportSlashFixer.php
//
// NoLeadingImportSlash removes a leading backslash from an import:
// "use \Foo\Bar;" -> "use Foo\Bar;", including "use function" / "use const".
type NoLeadingImportSlash struct{}

func (NoLeadingImportSlash) Name() string {
	return `PhpCsFixer\Fixer\Import\NoLeadingImportSlashFixer`
}

func (NoLeadingImportSlash) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Import/NoLeadingImportSlashFixer.php"
}

func (NoLeadingImportSlash) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind != token.Keyword || strings.ToLower(t.Value) != "use" {
			continue
		}
		if inClassLikeBody(s, i) {
			continue // trait use keeps its leading backslash
		}
		j := skipWhitespace(s, i+1)
		if j < s.Len() && s.At(j).Kind == token.Punct && s.At(j).Value == "(" {
			continue // closure use
		}
		if j < s.Len() && s.At(j).Kind == token.Keyword {
			if lw := strings.ToLower(s.At(j).Value); lw == "function" || lw == "const" {
				j = skipWhitespace(s, j+1)
			}
		}
		if j < s.Len() && s.At(j).Kind == token.Punct && s.At(j).Value == `\` {
			s.RemoveAt(j)
			changed = true
		}
	}
	return changed
}

func skipWhitespace(s *tokens.Stream, i int) int {
	for i < s.Len() && s.At(i).Kind == token.Whitespace {
		i++
	}
	return i
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Import/SingleImportPerStatementFixer.php
//
// SingleImportPerStatement splits a grouped import into one per line:
// "use A, B;" -> "use A;\nuse B;". Group imports ("use A\{B, C};") and closure
// "use (...)" are left alone.
type SingleImportPerStatement struct{}

func (SingleImportPerStatement) Name() string {
	return `PhpCsFixer\Fixer\Import\SingleImportPerStatementFixer`
}

func (SingleImportPerStatement) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Import/SingleImportPerStatementFixer.php"
}

func (SingleImportPerStatement) Fix(s *tokens.Stream) bool {
	return splitUseStatements(s, false)
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ClassNotation/SingleTraitInsertPerStatementFixer.php
//
// SingleTraitInsertPerStatement splits a grouped trait use inside a class body
// into one per line: "use A, B;" -> "use A;\nuse B;".
type SingleTraitInsertPerStatement struct{}

func (SingleTraitInsertPerStatement) Name() string {
	return `PhpCsFixer\Fixer\ClassNotation\SingleTraitInsertPerStatementFixer`
}

func (SingleTraitInsertPerStatement) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ClassNotation/SingleTraitInsertPerStatementFixer.php"
}

func (SingleTraitInsertPerStatement) Fix(s *tokens.Stream) bool {
	return splitUseStatements(s, true)
}

// splitUseStatements splits grouped "use" statements one per line. inClass picks
// the scope: false handles top-level imports, true handles in-class trait use.
func splitUseStatements(s *tokens.Stream, inClass bool) bool {
	changed := false
	i := 0
	for i < s.Len() {
		if s.At(i).Kind != token.Keyword || strings.ToLower(s.At(i).Value) != "use" {
			i++
			continue
		}
		next, c := splitUseAt(s, i, inClass)
		i = next
		if c {
			changed = true
		}
	}
	return changed
}

// splitUseAt splits the "use" statement at index i when its scope matches
// inClass. It returns the index to continue from and whether it changed.
func splitUseAt(s *tokens.Stream, i int, inClass bool) (int, bool) {
	if inClassLikeBody(s, i) != inClass {
		return i + 1, false
	}

	j := skipWhitespace(s, i+1)
	modifier := ""
	if j < s.Len() && s.At(j).Kind == token.Keyword {
		if lw := strings.ToLower(s.At(j).Value); lw == "function" || lw == "const" {
			modifier = s.At(j).Value
			j = skipWhitespace(s, j+1)
		}
	}
	if j < s.Len() && s.At(j).Kind == token.Punct && s.At(j).Value == "(" {
		return i + 1, false // closure "use (...)"
	}

	semi, group := -1, false
	var commas []int
	for k := j; k < s.Len(); k++ {
		if s.At(k).Kind != token.Punct {
			continue
		}
		switch s.At(k).Value {
		case "{":
			group = true
		case ";":
			semi = k
		case ",":
			commas = append(commas, k)
		}
		if group || semi >= 0 {
			break
		}
	}
	if group || semi < 0 || len(commas) == 0 {
		return i + 1, false
	}

	indent := indentBefore(s, i)
	parts := splitImportParts(s, j, semi, commas)

	var repl []token.Token
	for p, part := range parts {
		if p > 0 {
			repl = append(repl, token.Token{Kind: token.Whitespace, Value: "\n" + indent})
		}
		repl = append(repl, token.Token{Kind: token.Keyword, Value: "use"}, token.Token{Kind: token.Whitespace, Value: " "})
		if modifier != "" {
			repl = append(repl, token.Token{Kind: token.Keyword, Value: modifier}, token.Token{Kind: token.Whitespace, Value: " "})
		}
		repl = append(repl, part...)
		repl = append(repl, token.Token{Kind: token.Punct, Value: ";"})
	}
	s.ReplaceRange(i, semi, repl)
	return i + len(repl), true
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Import/SingleLineAfterImportsFixer.php
//
// SingleLineAfterImports ensures one blank line after the last import in a block
// of consecutive top-level use statements.
type SingleLineAfterImports struct{}

func (SingleLineAfterImports) Name() string {
	return `PhpCsFixer\Fixer\Import\SingleLineAfterImportsFixer`
}

func (SingleLineAfterImports) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Import/SingleLineAfterImportsFixer.php"
}

func (SingleLineAfterImports) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind != token.Keyword || strings.ToLower(t.Value) != "use" {
			continue
		}
		if inClassLikeBody(s, i) {
			continue // trait use
		}
		j := skipWhitespace(s, i+1)
		if j < s.Len() && s.At(j).Kind == token.Punct && s.At(j).Value == "(" {
			continue // closure use
		}
		semi := -1
		for k := i + 1; k < s.Len(); k++ {
			if s.At(k).Kind == token.Punct && s.At(k).Value == ";" {
				semi = k
				break
			}
		}
		if semi < 0 || semi+1 >= s.Len() {
			continue
		}
		nx := skipWhitespace(s, semi+1)
		if nx >= s.Len() {
			continue
		}
		next := s.At(nx)
		// only the last import in a block, and only when code follows
		if next.Kind == token.Keyword && strings.ToLower(next.Value) == "use" {
			continue
		}
		if next.Kind == token.Punct && next.Value == "}" {
			continue
		}
		ws := s.At(semi + 1)
		if ws.Kind == token.Whitespace && hasNewline(ws.Value) {
			// preserve the import line's own indentation after the blank line, so a
			// braced-namespace body keeps its level; a whitespace that already holds
			// exactly one blank line (two newlines) is left untouched, as php-cs-fixer
			if strings.Count(ws.Value, "\n") != 2 {
				target := "\n\n" + indentBefore(s, i)
				if ws.Value != target {
					s.SetValue(semi+1, target)
					changed = true
				}
			}
		}
	}
	return changed
}

// indentBefore returns the indentation (spaces/tabs after the last newline) of
// the token preceding index i.
func indentBefore(s *tokens.Stream, i int) string {
	if i == 0 || s.At(i-1).Kind != token.Whitespace {
		return ""
	}
	v := s.At(i - 1).Value
	if idx := strings.LastIndexByte(v, '\n'); idx >= 0 {
		return v[idx+1:]
	}
	return ""
}

// splitImportParts slices tokens (j, semi) at the given comma indices, trimming
// surrounding whitespace from each part.
func splitImportParts(s *tokens.Stream, j, semi int, commas []int) [][]token.Token {
	bounds := append([]int{j - 1}, commas...)
	bounds = append(bounds, semi)
	var parts [][]token.Token
	for b := 0; b+1 < len(bounds); b++ {
		var part []token.Token
		for k := bounds[b] + 1; k < bounds[b+1]; k++ {
			part = append(part, s.At(k))
		}
		parts = append(parts, trimWhitespace(part))
	}
	return parts
}

// inClassLikeBody reports whether index i is directly inside a class, interface,
// trait or enum body (as opposed to file scope or a bracketed namespace).
func inClassLikeBody(s *tokens.Stream, i int) bool {
	depth := 0
	for j := i - 1; j >= 0; j-- {
		t := s.At(j)
		if t.Kind != token.Punct {
			continue
		}
		switch t.Value {
		case "}":
			depth++
		case "{":
			if depth == 0 {
				return braceOpensClassLike(s, j)
			}
			depth--
		}
	}
	return false
}

// braceOpensClassLike reports whether the "{" at index brace opens a class-like
// body, by finding a class/interface/trait/enum keyword in its header.
func braceOpensClassLike(s *tokens.Stream, brace int) bool {
	for j := brace - 1; j >= 0; j-- {
		t := s.At(j)
		if t.Kind == token.Punct && (t.Value == ";" || t.Value == "{" || t.Value == "}") {
			return false
		}
		if t.Kind == token.Keyword && classLikeKeywords[strings.ToLower(t.Value)] {
			return true
		}
	}
	return false
}

func trimWhitespace(toks []token.Token) []token.Token {
	for len(toks) > 0 && toks[0].Kind == token.Whitespace {
		toks = toks[1:]
	}
	for len(toks) > 0 && toks[len(toks)-1].Kind == token.Whitespace {
		toks = toks[:len(toks)-1]
	}
	return toks
}
