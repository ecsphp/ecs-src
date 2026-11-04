package rules

import (
	"maps"
	"regexp"
	"slices"
	"strings"

	"blink/internal/fixer"
	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Import/GlobalNamespaceImportFixer.php
//
// GlobalNamespaceImport imports (true), fully qualifies (false) or ignores
// (null) global classes, functions and constants in a file with a single
// namespace: "\Foo" becomes "Foo" plus a "use Foo;" import, unless the short
// name is already taken by another import, a declaration or an unqualified use.
// Docblock types are imported along with the classes.
type GlobalNamespaceImport struct {
	// Each option is nil (ignore), or true (import) / false (fully qualify).
	// ImportClasses defaults to true; the others to nil.
	ImportConstants *bool
	ImportFunctions *bool
	ImportClasses   *bool
	// classesSet records that import_classes was given (possibly as null).
	classesSet bool
}

func (f GlobalNamespaceImport) WithConfig(config map[string]any) fixer.Fixer {
	opt := func(key string) (*bool, bool) {
		raw, ok := config[key]
		if !ok {
			return nil, false
		}
		if b, isBool := raw.(bool); isBool {
			return &b, true
		}
		return nil, true
	}
	if v, ok := opt("import_constants"); ok {
		f.ImportConstants = v
	}
	if v, ok := opt("import_functions"); ok {
		f.ImportFunctions = v
	}
	if v, ok := opt("import_classes"); ok {
		f.ImportClasses = v
		f.classesSet = true
	}
	return f
}

func (GlobalNamespaceImport) Name() string {
	return `PhpCsFixer\Fixer\Import\GlobalNamespaceImportFixer`
}

func (GlobalNamespaceImport) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Import/GlobalNamespaceImportFixer.php"
}

func (f GlobalNamespaceImport) importClasses() *bool {
	if f.classesSet {
		return f.ImportClasses
	}
	t := true
	return &t
}

var shebangRe = regexp.MustCompile(`^#!.+\n?$`)

// isMonolithicPHPFile mirrors Tokens::isMonolithicPhp. PHP folds the single newline
// after "?>" into T_CLOSE_TAG; blink lexes it as inline HTML, which is ignored.
func isMonolithicPHPFile(s *tokens.Stream) bool {
	openTags, html := 0, 0
	for i := range s.Len() {
		switch s.Kind(i) {
		case token.OpenTag:
			openTags++
		case token.InlineHTML:
			v := s.At(i).Value
			if i > 0 && s.Kind(i-1) == token.CloseTag {
				if strings.HasPrefix(v, "\r\n") {
					v = v[2:]
				} else if strings.HasPrefix(v, "\n") {
					v = v[1:]
				}
			}
			if v != "" {
				html++
			}
		}
	}
	if openTags != 1 {
		return false
	}
	return html == 0 || (html == 1 && s.Len() > 0 && s.Kind(0) == token.InlineHTML && shebangRe.MatchString(s.At(0).Value))
}

func (f GlobalNamespaceImport) Fix(s *tokens.Stream) bool {
	if !isMonolithicPHPFile(s) {
		return false
	}
	sc := newSymScan(s)
	candidate, namespaces := false, 0
	for i := range sc.n {
		if sc.isKind(i, token.DocComment) || sc.isPunct(i, `\`) || sc.isImportUse(i) {
			candidate = true
		}
		if sc.isNamespaceKeyword(i) {
			namespaces++
		}
	}
	if !candidate || namespaces != 1 || len(sc.nsDecls) != 1 || sc.nsDecls[0].isGlobal() {
		return false
	}
	// Each phase sees the tokens the previous one changed, as in PHP-CS-Fixer.
	// They work on a copy: a file PHP-CS-Fixer fails on is left as it was.
	work := tokens.New(slices.Clone(s.Tokens()))
	changed := false
	phase := func(run func(sc *symScan, edits *streamEdits)) {
		edits := newStreamEdits()
		run(sc, edits)
		if edits.apply(work, false) {
			changed = true
			sc = newSymScan(work)
		}
	}
	type importSet struct {
		kind  string
		names map[string]string
	}
	var newImports []importSet
	if c := f.ImportConstants; c != nil {
		if *c {
			phase(func(sc *symScan, edits *streamEdits) {
				newImports = append(newImports, importSet{"const", gniImportConstants(sc, edits)})
			})
		} else {
			phase(gniFullyQualifyConstants)
		}
	}
	if fn := f.ImportFunctions; fn != nil {
		if *fn {
			phase(func(sc *symScan, edits *streamEdits) {
				newImports = append(newImports, importSet{"function", gniImportFunctions(sc, edits)})
			})
		} else {
			phase(gniFullyQualifyFunctions)
		}
	}
	if c := f.importClasses(); c != nil {
		if *c {
			phase(func(sc *symScan, edits *streamEdits) {
				newImports = append(newImports, importSet{"class", gniImportClasses(sc, edits)})
			})
		} else {
			phase(gniFullyQualifyClasses)
		}
	}
	for _, set := range newImports {
		// A docblock type made of a lone "\" (as in the string type "\ ")
		// yields an empty name: PHP-CS-Fixer throws on the file, which stays
		// unchanged. Never write "use ;".
		for _, n := range set.names {
			if n == "" {
				return false
			}
		}
	}
	if len(newImports) > 0 {
		atIndex := sc.nsDecls[0].declEnd + 1
		if len(sc.uses) > 0 {
			atIndex = sc.uses[len(sc.uses)-1].end + 1
		}
		if atIndex < sc.n {
			// ImportProcessor::insertImports inserts each kind at the same index,
			// so the last kind ends up first
			var items []token.Token
			for _, set := range slices.Backward(newImports) {
				for _, n := range slices.Sorted(maps.Values(set.names)) {
					items = append(items,
						token.Token{Kind: token.Whitespace, Value: "\n"},
						token.Token{Kind: token.Keyword, Value: "use"},
						token.Token{Kind: token.Whitespace, Value: " "})
					if set.kind != "class" {
						items = append(items,
							token.Token{Kind: token.Keyword, Value: set.kind},
							token.Token{Kind: token.Whitespace, Value: " "})
					}
					items = append(items, gniTokenizeName(n)...)
					items = append(items, token.Token{Kind: token.Punct, Value: ";"})
				}
			}
			if at := sc.tok(atIndex); at.Kind != token.Whitespace || !strings.Contains(at.Value, "\n") {
				items = append(items, token.Token{Kind: token.Whitespace, Value: "\n"})
			}
			if len(items) > 0 {
				work.InsertSliceAt(atIndex, items)
				changed = true
			}
		}
	}
	if changed {
		s.ReplaceRange(0, s.Len()-1, work.Tokens())
	}
	return changed
}

// gniTokenizeName is ImportProcessor::tokenizeName.
func gniTokenizeName(name string) []token.Token {
	var out []token.Token
	if strings.HasPrefix(name, `\`) {
		out = append(out, nsSeparatorToken())
		name = name[1:]
	}
	for i, part := range strings.Split(name, `\`) {
		if i > 0 {
			out = append(out, nsSeparatorToken())
		}
		out = append(out, token.Token{Kind: token.Ident, Value: part})
	}
	return out
}

// gniGlobal maps a global import's name to its alias ("" when not aliased).
type gniGlobal map[string]string

// gniFilterUses mirrors filterUseDeclarations().
func gniFilterUses(sc *symScan, kind string, caseSensitive, skipAliased bool) (gniGlobal, map[string]bool) {
	global, other := gniGlobal{}, map[string]bool{}
	norm := func(v string) string {
		if caseSensitive {
			return v
		}
		return strings.ToLower(v)
	}
	for _, u := range sc.uses {
		if u.kind != kind || (skipAliased && u.aliased) {
			continue
		}
		full := strings.TrimLeft(u.fullName, `\`)
		if strings.Contains(full, `\`) {
			other[norm(u.shortName)] = true
			continue
		}
		alias := ""
		if u.shortName != full {
			alias = u.shortName
		}
		global[norm(full)] = alias
	}
	return global, other
}

// gniPrepareImports mirrors prepareImports(): drop the leading "\" of each
// usage whose name is free, returning the names to import.
func gniPrepareImports(sc *symScan, edits *streamEdits, indices []int, global gniGlobal, other map[string]bool, caseSensitive bool) map[string]string {
	imports := map[string]string{}
	for _, i := range indices {
		name := sc.tok(i).Value
		check := name
		if !caseSensitive {
			check = strings.ToLower(name)
		}
		if other[check] {
			continue
		}
		if alias, ok := global[check]; !ok {
			imports[check] = name
		} else if alias != "" {
			edits.sets[i] = alias
		}
		edits.removes[sc.prevM(i)] = true
	}
	return imports
}

func gniImportConstants(sc *symScan, edits *streamEdits) map[string]string {
	global, other := gniFilterUses(sc, "const", true, false)
	for i := 0; i < sc.n; i++ {
		if sc.isClassy(i) {
			open := sc.nextPunct(i, "{")
			if open < 0 || sc.match[open] < 0 {
				break
			}
			i = sc.match[open]
			continue
		}
		if !sc.isConstKeyword(i) {
			continue
		}
		i = sc.nextM(i)
		if i < 0 {
			break
		}
		other[sc.tok(i).Value] = true
	}
	var indices []int
	for i := sc.n - 1; i >= 0; i-- {
		if !sc.isTString(i) {
			continue
		}
		name := sc.tok(i).Value
		if other[name] || !sc.isConstantInvocation(i) {
			continue
		}
		sep := sc.prevM(i)
		if !sc.isPunct(sep, `\`) {
			if _, ok := global[name]; !ok {
				other[name] = true
			}
			continue
		}
		if p := sc.prevM(sep); sc.isNamespaceOperator(p) || sc.isTString(p) {
			continue
		}
		indices = append(indices, i)
	}
	return gniPrepareImports(sc, edits, indices, global, other, true)
}

func gniImportFunctions(sc *symScan, edits *streamEdits) map[string]string {
	global, other := gniFilterUses(sc, "function", false, false)
	for _, d := range gniFunctionDeclarations(sc, 0, sc.n-1) {
		other[strings.ToLower(sc.tok(d).Value)] = true
	}
	fs := newFuncScan(sc)
	var indices []int
	for i := sc.n - 1; i >= 0; i-- {
		if !sc.isTString(i) {
			continue
		}
		name := strings.ToLower(sc.tok(i).Value)
		if other[name] || !fs.isGlobalFunctionUsage(i) {
			continue
		}
		if !sc.isPunct(sc.prevM(i), `\`) {
			if _, ok := global[name]; !ok {
				other[name] = true
			}
			continue
		}
		indices = append(indices, i)
	}
	return gniPrepareImports(sc, edits, indices, global, other, false)
}

// gniFunctionDeclarations mirrors findFunctionDeclarations(): functions declared
// outside classes, plus those declared inside method bodies.
func gniFunctionDeclarations(sc *symScan, start, end int) []int {
	var out []int
	for i := start; i <= end && i < sc.n; i++ {
		if sc.isClassy(i) {
			open := sc.nextPunct(i, "{")
			if open < 0 || sc.match[open] < 0 {
				return out
			}
			classEnd := sc.match[open]
			for j := open; j <= classEnd; j++ {
				if !sc.isFunctionKeyword(j) {
					continue
				}
				body := -1
				for k := j + 1; k < sc.n; k++ {
					if sc.isPunct(k, "{") || sc.isPunct(k, ";") {
						body = k
						break
					}
				}
				if body < 0 {
					break
				}
				if sc.isPunct(body, ";") || sc.match[body] < 0 {
					j = body
					continue
				}
				out = append(out, gniFunctionDeclarations(sc, body, sc.match[body])...)
				j = sc.match[body]
			}
			i = classEnd
			continue
		}
		if !sc.isFunctionKeyword(i) {
			continue
		}
		i = sc.nextM(i)
		if i < 0 {
			break
		}
		if sc.isReturnRef(i) {
			i = sc.nextM(i)
			if i < 0 {
				break
			}
		}
		if sc.isTString(i) {
			out = append(out, i)
		}
	}
	return out
}

func gniImportClasses(sc *symScan, edits *streamEdits) map[string]string {
	global, other := gniFilterUses(sc, "class", false, false)
	var docs []int
	for i := 0; i < sc.n; i++ {
		if sc.isKind(i, token.DocComment) {
			docs = append(docs, i)
			traverseDocTypes(sc.tok(i).Value, func(typ string) string {
				if !strings.Contains(typ, `\`) {
					name := strings.ToLower(typ)
					if _, ok := global[name]; !ok {
						other[name] = true
					}
				}
				return typ
			})
		}
		if !sc.isClassy(i) {
			continue
		}
		i = sc.nextM(i)
		if i < 0 {
			break
		}
		if sc.isTString(i) {
			other[strings.ToLower(sc.tok(i).Value)] = true
		}
	}
	var indices []int
	for i := sc.n - 1; i >= 0; i-- {
		if !sc.isTString(i) {
			continue
		}
		name := strings.ToLower(sc.tok(i).Value)
		if other[name] || !sc.isClassyInvocation(i) {
			continue
		}
		sep := sc.prevM(i)
		if !sc.isPunct(sep, `\`) {
			if _, ok := global[name]; !ok {
				other[name] = true
			}
			continue
		}
		if p := sc.prevM(sep); sc.isNamespaceOperator(p) || sc.isTString(p) {
			continue
		}
		indices = append(indices, i)
	}
	imports := map[string]string{}
	for _, d := range docs {
		content, changed := traverseDocTypes(sc.tok(d).Value, func(typ string) string {
			if !strings.HasPrefix(typ, `\`) {
				return typ
			}
			name := typ[1:]
			check := strings.ToLower(name)
			if strings.Contains(check, `\`) || other[check] {
				return typ
			}
			if alias, ok := global[check]; ok {
				if alias != "" {
					return alias
				}
				return name
			}
			imports[check] = name
			return name
		})
		if changed {
			edits.sets[d] = content
		}
	}
	maps.Copy(imports, gniPrepareImports(sc, edits, indices, global, other, false))
	return imports
}

func gniFullyQualifyConstants(sc *symScan, edits *streamEdits) {
	global, _ := gniFilterUses(sc, "const", true, true)
	if len(global) == 0 {
		return
	}
	for i := sc.n - 1; i >= 0; i-- {
		if !sc.isTString(i) {
			continue
		}
		if _, ok := global[sc.tok(i).Value]; !ok {
			continue
		}
		if sc.isPunct(sc.prevM(i), `\`) || !sc.isConstantInvocation(i) {
			continue
		}
		edits.inserts[i] = []token.Token{nsSeparatorToken()}
	}
}

func gniFullyQualifyFunctions(sc *symScan, edits *streamEdits) {
	global, _ := gniFilterUses(sc, "function", false, true)
	if len(global) == 0 {
		return
	}
	fs := newFuncScan(sc)
	for i := sc.n - 1; i >= 0; i-- {
		if !sc.isTString(i) {
			continue
		}
		if _, ok := global[strings.ToLower(sc.tok(i).Value)]; !ok {
			continue
		}
		if sc.isPunct(sc.prevM(i), `\`) || !fs.isGlobalFunctionUsage(i) {
			continue
		}
		edits.inserts[i] = []token.Token{nsSeparatorToken()}
	}
}

func gniFullyQualifyClasses(sc *symScan, edits *streamEdits) {
	global, _ := gniFilterUses(sc, "class", false, true)
	if len(global) == 0 {
		return
	}
	for i := sc.n - 1; i >= 0; i-- {
		if sc.isKind(i, token.DocComment) {
			content, changed := traverseDocTypes(sc.tok(i).Value, func(typ string) string {
				if _, ok := global[strings.ToLower(typ)]; !ok {
					return typ
				}
				return `\` + typ
			})
			if changed {
				edits.sets[i] = content
			}
			continue
		}
		if !sc.isTString(i) {
			continue
		}
		if _, ok := global[strings.ToLower(sc.tok(i).Value)]; !ok {
			continue
		}
		if sc.isPunct(sc.prevM(i), `\`) || !sc.isClassyInvocation(i) {
			continue
		}
		edits.inserts[i] = []token.Token{nsSeparatorToken()}
	}
}
