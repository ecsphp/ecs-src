package rules

import (
	"slices"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

<<<<<<< HEAD
// PHP-CS-Fixer (deprecated): https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/LanguageConstruct/ClassKeywordRemoveFixer.php
//
// ClassKeywordRemove converts `Foo\Bar::class` into the FQCN string 'Foo\Bar',
// resolving the name against the current namespace and use imports. Group and
// comma imports are left unresolved (the simple, common case is covered).
=======
// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/LanguageConstruct/ClassKeywordRemoveFixer.php
//
// ClassKeywordRemove converts "::class" usages to fully-qualified-class-name
// string literals, resolving names through the file's use imports (plain,
// aliased and PHP 7 group imports) and the current namespace, one
// namespace-section at a time. It is deprecated upstream with no successor.
>>>>>>> 8f741397a6 (Blink: port ClassKeywordRemoveFixer from PHP-CS-Fixer)
type ClassKeywordRemove struct{}

func (ClassKeywordRemove) Name() string {
	return `PhpCsFixer\Fixer\LanguageConstruct\ClassKeywordRemoveFixer`
}

func (ClassKeywordRemove) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/LanguageConstruct/ClassKeywordRemoveFixer.php"
}

<<<<<<< HEAD
// IsCandidate rules out a file with no `::class`: the "class" word must be present.
=======
// IsCandidate rules out a file without a "::class" usage (CT::T_CLASS_CONSTANT).
>>>>>>> 8f741397a6 (Blink: port ClassKeywordRemoveFixer from PHP-CS-Fixer)
func (ClassKeywordRemove) IsCandidate(s *tokens.Stream) bool {
	return s.Index().HasWord("class")
}

<<<<<<< HEAD
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
=======
// ckrImport is one stored import: value is the imported name as written, and
// alias is set (hasAlias) only for an aliased or group-aliased import.
type ckrImport struct {
	alias    string
	hasAlias bool
	value    string
}

func (ClassKeywordRemove) Fix(s *tokens.Stream) bool {
	sc := sharedSymScan(s)
	// Section boundaries are taken from the namespace declarations once, as in
	// applyFix: each replace grows the stream without compacting, so later
	// (stale) indices drift just as they do upstream.
	changed := false
	prev := 0
	for _, d := range sc.nsDecls {
		if ckrReplaceSection(s, "", prev, d.scopeStart) {
			changed = true
		}
		if ckrReplaceSection(s, d.name, d.scopeStart, d.scopeEnd) {
			changed = true
		}
		prev = d.scopeEnd
	}
	if ckrReplaceSection(s, "", prev, s.Len()-1) {
		changed = true
	}
	if changed {
		s.ClearEmptyTokens()
	}
	return changed
}

func ckrReplaceSection(s *tokens.Stream, namespace string, startIndex, endIndex int) bool {
	if endIndex-startIndex < 3 {
		return false
	}

	imports := ckrStoreImports(s, startIndex, endIndex)

	lo, hi := startIndex, endIndex
	if lo < 0 {
		lo = 0
	}
	if hi > s.Len()-1 {
		hi = s.Len() - 1
	}
	var classIndexes []int
	for i := lo; i <= hi; i++ {
		if ckrIsClassConstant(s, i) {
			classIndexes = append(classIndexes, i)
		}
	}

	changed := false
	for _, classIndex := range slices.Backward(classIndexes) {
		if ckrReplaceClassKeyword(s, namespace, classIndex, imports) {
			changed = true
		}
	}
	return changed
}

// ckrIsClassConstant reports CT::T_CLASS_CONSTANT: the "class" of "Foo::class".
func ckrIsClassConstant(s *tokens.Stream, i int) bool {
	t := s.At(i)
	if t.Kind != token.Ident || !strings.EqualFold(t.Value, "class") {
		return false
	}
	return isPunctVal(s, sigPrev(s, i), "::")
}

// ckrStoreImports mirrors storeImports: it walks every "use" token in the window
// and records the imported names, keeping group imports and aliases apart.
func ckrStoreImports(s *tokens.Stream, startIndex, endIndex int) []ckrImport {
	var imports []ckrImport
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind != token.Keyword || !strings.EqualFold(t.Value, "use") {
			continue
		}
		if i < startIndex || i > endIndex {
			continue
		}

		var b strings.Builder
		brk := -1
		for idx := sigNext(s, i); idx >= 0; idx = sigNext(s, idx) {
			tk := s.At(idx)
			groupOpen := tk.Kind == token.Punct && tk.Value == "{" && isPunctVal(s, sigPrev(s, idx), `\`)
			if (tk.Kind == token.Punct && tk.Value == ";") || groupOpen || (tk.Kind == token.Keyword && strings.EqualFold(tk.Value, "as")) {
				brk = idx
				break
			}
			b.WriteString(tk.Value)
		}
		importStr := b.String()

		if brk < 0 {
			imports = append(imports, ckrImport{value: importStr})
			continue
		}

		bt := s.At(brk)
		switch {
		case bt.Kind == token.Punct && bt.Value == "{":
			groupEnd := s.MatchForward(brk)
			if groupEnd < 0 {
				imports = append(imports, ckrImport{value: importStr})
				continue
			}
			for piece := range strings.SplitSeq(ckrPartialCode(s, brk+1, groupEnd-1), ",") {
				piece = strings.TrimSpace(piece)
				if parts := strings.Split(piece, " as "); len(parts) == 2 {
					imports = append(imports, ckrImport{alias: strings.TrimSpace(parts[1]), hasAlias: true, value: importStr + strings.TrimSpace(parts[0])})
				} else {
					imports = append(imports, ckrImport{value: importStr + piece})
				}
			}
		case bt.Kind == token.Keyword && strings.EqualFold(bt.Value, "as"):
			if aliasIdx := sigNext(s, brk); aliasIdx >= 0 {
				imports = append(imports, ckrImport{alias: s.At(aliasIdx).Value, hasAlias: true, value: importStr})
			} else {
				imports = append(imports, ckrImport{value: importStr})
			}
		default:
			imports = append(imports, ckrImport{value: importStr})
		}
	}
	return imports
}

func ckrReplaceClassKeyword(s *tokens.Stream, namespacePrefix string, classIndex int, imports []ckrImport) bool {
	classEndIndex := sigPrev(s, sigPrev(s, classIndex))
	if classEndIndex < 0 || s.At(classEndIndex).Kind != token.Ident {
		return false
	}
	switch strings.ToLower(s.At(classEndIndex).Value) {
	case "self", "static", "parent":
		return false
	}

	classBeginIndex := classEndIndex
	for {
		p := sigPrev(s, classBeginIndex)
		if p < 0 {
			break
		}
		pt := s.At(p)
		isNamePart := pt.Kind == token.Ident || (pt.Kind == token.Punct && pt.Value == `\`)
		if !isNamePart {
			break
		}
		classBeginIndex = p
	}

	leadingNs := s.At(classBeginIndex).Kind == token.Punct && s.At(classBeginIndex).Value == `\`
	nameStart := classBeginIndex
	if leadingNs {
		nameStart = sigNext(s, classBeginIndex)
	}
	classString := ckrPartialCode(s, nameStart, classEndIndex)

	classImport, haveImport := "", false
	if leadingNs {
		namespacePrefix = ""
	} else {
		first := classString
		if before, _, ok := strings.Cut(classString, `\`); ok {
			first = before
		}
		for _, imp := range imports {
			if imp.hasAlias && classString == imp.alias {
				classImport, haveImport = imp.value, true
				break
			}
			if first == ckrSubstrTail(imp.value, len(first)) {
				classImport, haveImport = imp.value, true
				break
			}
		}
	}

	for i := classBeginIndex; i <= classIndex; i++ {
		t := s.At(i)
		if t.Kind == token.Comment || t.Kind == token.DocComment {
			continue
		}
		if t.Kind == token.Whitespace && strings.Contains(t.Value, "\n") {
			continue
		}
		clearTokenAt(s, i)
	}
	s.InsertAt(classBeginIndex, token.Token{Kind: token.String, Value: "'" + ckrMakeClassFQN(namespacePrefix, haveImport, classImport, classString) + "'"})
	return true
}

// ckrMakeClassFQN mirrors makeClassFQN: without an import it prefixes the
// namespace; with one it splices the import's prefix onto the written name.
func ckrMakeClassFQN(namespacePrefix string, haveImport bool, classImport, classString string) string {
	if !haveImport {
		if namespacePrefix != "" {
			return namespacePrefix + `\` + classString
		}
		return classString
	}

	classStringArray := strings.Split(classString, `\`)
	classImportArray := strings.Split(classImport, `\`)
	if len(classStringArray) == 1 {
		return classImport
	}

	head := ckrSliceHead(classImportArray, len(classImportArray)-len(classStringArray)+1)
	merged := make([]string, 0, len(head)+len(classStringArray))
	merged = append(merged, head...)
	merged = append(merged, classStringArray...)
	return strings.Join(merged, `\`)
}

// ckrSliceHead is array_slice(arr, 0, length), honoring PHP's negative length
// (stop that many elements from the end).
func ckrSliceHead(arr []string, length int) []string {
	end := length
	if end < 0 {
		end = len(arr) + length
>>>>>>> 8f741397a6 (Blink: port ClassKeywordRemoveFixer from PHP-CS-Fixer)
	}
	if end < 0 {
		end = 0
	}
<<<<<<< HEAD
	if end > n {
		end = n
	}
	return append([]string(nil), arr[:end]...)
=======
	if end > len(arr) {
		end = len(arr)
	}
	return arr[:end]
}

// ckrSubstrTail is substr(str, -n): the last n bytes, or the whole string when
// n is not positive or exceeds its length.
func ckrSubstrTail(str string, n int) string {
	if n <= 0 || n >= len(str) {
		return str
	}
	return str[len(str)-n:]
}

// ckrPartialCode concatenates the raw values of tokens [from, to], like
// Tokens::generatePartialCode.
func ckrPartialCode(s *tokens.Stream, from, to int) string {
	if from < 0 || from > to {
		return ""
	}
	var b strings.Builder
	for j := from; j <= to && j < s.Len(); j++ {
		b.WriteString(s.At(j).Value)
	}
	return b.String()
>>>>>>> 8f741397a6 (Blink: port ClassKeywordRemoveFixer from PHP-CS-Fixer)
}
