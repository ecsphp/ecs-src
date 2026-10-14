package rules

import (
	"strings"

	"blink/internal/fixer"
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
// "use A, B;" -> "use A;\nuse B;". With group_to_single_imports (the default) a
// brace group "use A\{B, C};" is also expanded; set it to false to keep groups.
// Closure "use (...)" is always left alone.
type SingleImportPerStatement struct {
	// keepGroups leaves brace group imports untouched (group_to_single_imports:false).
	keepGroups bool
}

func (SingleImportPerStatement) Name() string {
	return `PhpCsFixer\Fixer\Import\SingleImportPerStatementFixer`
}

func (SingleImportPerStatement) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Import/SingleImportPerStatementFixer.php"
}

func (f SingleImportPerStatement) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["group_to_single_imports"].(bool); ok {
		f.keepGroups = !v
	}
	return f
}

func (f SingleImportPerStatement) Fix(s *tokens.Stream) bool {
	return splitUseStatements(s, false, !f.keepGroups)
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
	return splitUseStatements(s, true, false)
}

// splitUseStatements splits grouped "use" statements one per line. inClass picks
// the scope: false handles top-level imports, true handles in-class trait use.
// expandGroups also expands brace group imports ("use A\{B, C};").
func splitUseStatements(s *tokens.Stream, inClass, expandGroups bool) bool {
	changed := false
	i := 0
	for i < s.Len() {
		if s.At(i).Kind != token.Keyword || strings.ToLower(s.At(i).Value) != "use" {
			i++
			continue
		}
		next, c := splitUseAt(s, i, inClass, expandGroups)
		i = next
		if c {
			changed = true
		}
	}
	return changed
}

// splitUseAt splits the "use" statement at index i when its scope matches
// inClass. It returns the index to continue from and whether it changed.
func splitUseAt(s *tokens.Stream, i int, inClass, expandGroups bool) (int, bool) {
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

	semi, group, openBrace := -1, false, -1
	var commas []int
	for k := j; k < s.Len(); k++ {
		if s.At(k).Kind != token.Punct {
			continue
		}
		switch s.At(k).Value {
		case "{":
			group = true
			openBrace = k
		case ";":
			semi = k
		case ",":
			commas = append(commas, k)
		}
		if group || semi >= 0 {
			break
		}
	}
	if group {
		if !expandGroups {
			return i + 1, false
		}
		return expandGroupImport(s, i, j, openBrace, modifier)
	}
	if semi < 0 || len(commas) == 0 {
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

// expandGroupImport expands a brace group import "use Prefix\{A, function b, const C};"
// into one single import per member, carrying the common prefix and any outer or
// per-member "function"/"const" modifier. prefixStart points at the first prefix
// token (after "use" and an outer modifier), openBrace at the "{".
func expandGroupImport(s *tokens.Stream, useIdx, prefixStart, openBrace int, outerModifier string) (int, bool) {
	closeBrace := -1
	depth := 0
	for k := openBrace; k < s.Len(); k++ {
		if s.At(k).Kind != token.Punct {
			continue
		}
		switch s.At(k).Value {
		case "{":
			depth++
		case "}":
			depth--
			if depth == 0 {
				closeBrace = k
			}
		}
		if closeBrace >= 0 {
			break
		}
	}
	if closeBrace < 0 {
		return useIdx + 1, false
	}

	semi := -1
	for k := closeBrace + 1; k < s.Len(); k++ {
		if s.At(k).Kind == token.Whitespace || s.At(k).Kind == token.Comment || s.At(k).Kind == token.DocComment {
			continue
		}
		if s.At(k).Kind == token.Punct && s.At(k).Value == ";" {
			semi = k
		}
		break
	}
	if semi < 0 {
		return useIdx + 1, false
	}

	var prefix []token.Token
	for k := prefixStart; k < openBrace; k++ {
		prefix = append(prefix, s.At(k))
	}
	prefix = trimWhitespace(prefix)

	items := splitGroupItems(s, openBrace+1, closeBrace)
	indent := indentBefore(s, useIdx)

	var repl []token.Token
	emitted := 0
	for _, item := range items {
		item = trimWhitespace(item)
		if len(item) == 0 {
			continue // trailing comma
		}
		modifier := outerModifier
		rest := item
		if item[0].Kind == token.Keyword {
			if lw := strings.ToLower(item[0].Value); lw == "function" || lw == "const" {
				modifier = item[0].Value
				rest = trimWhitespace(item[1:])
			}
		}
		if emitted > 0 {
			repl = append(repl, token.Token{Kind: token.Whitespace, Value: "\n" + indent})
		}
		repl = append(repl, token.Token{Kind: token.Keyword, Value: "use"}, token.Token{Kind: token.Whitespace, Value: " "})
		if modifier != "" {
			repl = append(repl, token.Token{Kind: token.Keyword, Value: modifier}, token.Token{Kind: token.Whitespace, Value: " "})
		}
		repl = append(repl, prefix...)
		repl = append(repl, rest...)
		repl = append(repl, token.Token{Kind: token.Punct, Value: ";"})
		emitted++
	}
	if emitted == 0 {
		return useIdx + 1, false
	}

	s.ReplaceRange(useIdx, semi, repl)
	return useIdx + len(repl), true
}

// splitGroupItems slices the tokens in (open, close) at top-level commas.
func splitGroupItems(s *tokens.Stream, from, close int) [][]token.Token {
	var items [][]token.Token
	var cur []token.Token
	depth := 0
	for k := from; k < close; k++ {
		t := s.At(k)
		if t.Kind == token.Punct {
			switch t.Value {
			case "{", "[", "(":
				depth++
			case "}", "]", ")":
				depth--
			case ",":
				if depth == 0 {
					items = append(items, cur)
					cur = nil
					continue
				}
			}
		}
		cur = append(cur, t)
	}
	items = append(items, cur)
	return items
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
