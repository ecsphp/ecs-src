package rules

import (
	"slices"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer (deprecated): https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/LanguageConstruct/ClassKeywordRemoveFixer.php
//
// ClassKeywordRemove converts `Foo\Bar::class` into the FQCN string 'Foo\Bar',
// resolving the name against the current namespace and use imports. Group and
// comma imports are left unresolved (the simple, common case is covered).
type ClassKeywordRemove struct{}

func (ClassKeywordRemove) Name() string {
	return `PhpCsFixer\Fixer\LanguageConstruct\ClassKeywordRemoveFixer`
}

func (ClassKeywordRemove) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/LanguageConstruct/ClassKeywordRemoveFixer.php"
}

// IsCandidate rules out a file with no `::class`: the "class" word must be present.
func (ClassKeywordRemove) IsCandidate(s *tokens.Stream) bool {
	return s.Index().HasWord("class")
}

type ckrImport struct {
	aliased bool
	alias   string
	full    string
}

type ckrSection struct {
	prefix     string
	start, end int
}

type ckrHit struct {
	begin, classIdx int
	fqn             string
}

func (ClassKeywordRemove) Fix(s *tokens.Stream) bool {
	s.ClearEmptyTokens()
	var hits []ckrHit
	seen := map[int]bool{}
	for _, sec := range ckrSections(s) {
		if sec.end-sec.start < 3 {
			continue
		}
		imports := ckrImportsIn(s, sec.start, sec.end)
		for i := sec.start; i <= sec.end && i < s.Len(); i++ {
			t := s.At(i)
			if t.Kind != token.Ident || !strings.EqualFold(t.Value, "class") {
				continue
			}
			if seen[i] {
				continue
			}
			colon := sigPrev(s, i)
			if !isPunctVal(s, colon, "::") {
				continue
			}
			end := sigPrev(s, colon)
			if end < 0 || s.At(end).Kind != token.Ident {
				continue
			}
			switch strings.ToLower(s.At(end).Value) {
			case "self", "parent", "static":
				continue
			}
			begin := end
			for {
				prev := sigPrev(s, begin)
				if prev < 0 {
					break
				}
				pt := s.At(prev)
				if pt.Kind == token.Ident || (pt.Kind == token.Punct && pt.Value == `\`) {
					begin = prev
					continue
				}
				break
			}
			leading := s.At(begin).Kind == token.Punct && s.At(begin).Value == `\`
			nameStart := begin
			if leading {
				nameStart = sigNext(s, begin)
			}
			var name strings.Builder
			for j := nameStart; j >= 0 && j <= end; j++ {
				jt := s.At(j)
				if jt.Kind == token.Ident || (jt.Kind == token.Punct && jt.Value == `\`) {
					name.WriteString(jt.Value)
				}
			}
			seen[i] = true
			hits = append(hits, ckrHit{begin: begin, classIdx: i, fqn: ckrResolve(sec.prefix, imports, name.String(), leading)})
		}
	}
	if len(hits) == 0 {
		return false
	}
	// apply high index first so earlier offsets stay valid, as PHP-CS-Fixer does
	for _, h := range slices.Backward(hits) {
		for i := h.begin; i <= h.classIdx; i++ {
			t := s.At(i)
			if t.Kind == token.Comment || t.Kind == token.DocComment {
				continue
			}
			if t.Kind == token.Whitespace && strings.Contains(t.Value, "\n") {
				continue
			}
			clearTokenAt(s, i)
		}
		s.InsertAt(h.begin, token.Token{Kind: token.String, Value: "'" + h.fqn + "'"})
	}
	s.ClearEmptyTokens()
	return true
}

// ckrSections splits the file into namespace sections, mirroring the per-scope
// loop in ClassKeywordRemoveFixer::applyFix. Empty-prefix gaps between
// declarations are kept so an index-range check matches PHP-CS-Fixer's.
func ckrSections(s *tokens.Stream) []ckrSection {
	decls := aliasNamespaceDeclarations(s)
	var out []ckrSection
	prevEnd := 0
	for _, d := range decls {
		out = append(out, ckrSection{prefix: "", start: prevEnd, end: d.scopeStart})
		name := ""
		if !d.global {
			name = ckrNamespaceName(s, d.scopeStart)
		}
		out = append(out, ckrSection{prefix: name, start: d.scopeStart, end: d.scopeEnd})
		prevEnd = d.scopeEnd
	}
	out = append(out, ckrSection{prefix: "", start: prevEnd, end: s.Len() - 1})
	return out
}

func ckrNamespaceName(s *tokens.Stream, nsKeyword int) string {
	var name strings.Builder
	for j := nsKeyword + 1; j < s.Len(); j++ {
		if isPunctVal(s, j, ";") || isPunctVal(s, j, "{") {
			break
		}
		t := s.At(j)
		if t.Kind == token.Ident || (t.Kind == token.Punct && t.Value == `\`) {
			name.WriteString(t.Value)
		}
	}
	return name.String()
}

// ckrImportsIn collects the class use imports whose `use` keyword sits in
// [start, end], mirroring ClassKeywordRemoveFixer::storeImports for single,
// non-group imports.
func ckrImportsIn(s *tokens.Stream, start, end int) []ckrImport {
	var out []ckrImport
	for _, use := range aliasImportUseIndexes(s) {
		if use < start || use > end {
			continue
		}
		var full strings.Builder
		aliased := false
		group := false
		j := sigNext(s, use)
		for j >= 0 {
			t := s.At(j)
			if isPunctVal(s, j, ";") || s.At(j).Kind == token.CloseTag {
				break
			}
			if isPunctVal(s, j, "{") {
				group = true
				break
			}
			if kwIs(t, "as") {
				aliased = true
				break
			}
			full.WriteString(t.Value)
			j = sigNext(s, j)
		}
		if group {
			continue
		}
		if aliased {
			aliasIdx := sigNext(s, j)
			if aliasIdx < 0 {
				continue
			}
			out = append(out, ckrImport{aliased: true, alias: s.At(aliasIdx).Value, full: full.String()})
			continue
		}
		out = append(out, ckrImport{full: full.String()})
	}
	return out
}

func ckrResolve(prefix string, imports []ckrImport, classString string, leading bool) string {
	matched := false
	importFull := ""
	if leading {
		prefix = ""
	} else {
		first := classString
		if before, _, ok := strings.Cut(classString, "\\"); ok {
			first = before
		}
		for _, e := range imports {
			if e.aliased && classString == e.alias {
				matched, importFull = true, e.full
				break
			}
			if strings.HasSuffix(e.full, first) {
				matched, importFull = true, e.full
				break
			}
		}
	}
	if !matched {
		if prefix != "" {
			return prefix + `\` + classString
		}
		return classString
	}
	sParts := strings.Split(classString, `\`)
	if len(sParts) == 1 {
		return importFull
	}
	iParts := strings.Split(importFull, `\`)
	head := ckrArraySlice(iParts, len(iParts)-len(sParts)+1)
	return strings.Join(append(head, sParts...), `\`)
}

// ckrArraySlice ports array_slice($arr, 0, $length) for PHP's makeClassFQN: a
// negative length keeps that many elements off the end.
func ckrArraySlice(arr []string, length int) []string {
	n := len(arr)
	end := length
	if length < 0 {
		end = n + length
	}
	if end < 0 {
		end = 0
	}
	if end > n {
		end = n
	}
	return append([]string(nil), arr[:end]...)
}
