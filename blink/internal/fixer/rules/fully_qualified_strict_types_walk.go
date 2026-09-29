package rules

import (
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// fqProcessNamespace collects every class-reference shortening for one namespace
// scope. Default config only: no symbol import, so a single non-discovery pass.
func fqProcessNamespace(s *tokens.Stream, reg fqRegion) []fqReplace {
	uses := fqCollectUsesRegion(s, reg)
	var repls []fqReplace
	seen := map[int]bool{} // dedupe overlapping dispatch of the same name run

	add := func(rs []fqReplace) {
		for _, r := range rs {
			if seen[r.start] {
				continue
			}
			seen[r.start] = true
			repls = append(repls, r)
		}
	}

	reserved := map[string]bool{}
	depth := 0
	for i := reg.start; i < reg.end && i < s.Len(); i++ {
		t := s.At(i)
		switch {
		case t.Kind == token.Punct && t.Value == "{":
			depth++
		case t.Kind == token.Punct && t.Value == "}":
			depth--
		case t.Kind == token.Variable:
			if p := sigPrev(s, i); p >= 0 && s.At(p).Kind == token.Ident {
				if r, ok := fqPrevName(s, i, uses, reg.name, reserved); ok {
					add([]fqReplace{r})
				}
			}
		case t.Kind == token.Punct && t.Value == "::":
			if r, ok := fqPrevName(s, i, uses, reg.name, reserved); ok {
				add([]fqReplace{r})
			}
		case t.Kind == token.Keyword:
			switch strings.ToLower(t.Value) {
			case "function", "fn":
				add(fqFunction(s, i, reg.end, uses, reg.name, reserved))
			case "catch":
				add(fqCatch(s, i, uses, reg.name, reserved))
			case "extends", "implements":
				add(fqExtendsImplements(s, i, uses, reg.name, reserved))
			case "new", "instanceof":
				if r, ok := fqNextName(s, i, uses, reg.name, reserved); ok {
					add([]fqReplace{r})
				}
			case "use":
				// only trait-use (inside a class body) shortens; a top-level
				// `use` is an import declaration and must never be touched
				if depth >= 1 {
					if r, ok := fqNextName(s, i, uses, reg.name, reserved); ok {
						add([]fqReplace{r})
					}
				}
			}
		case t.Kind == token.DocComment:
			for _, id := range fqDocTemplateNames(t.Value) {
				reserved[id] = true
			}
			if r, ok := fqPhpDoc(s, i, uses, reg.name, reserved); ok {
				add([]fqReplace{r})
			}
		}
	}
	return repls
}

// fqCollectUsesRegion builds the class-import map for a namespace region: `use`
// imports at the region base depth (not `use function`/`use const`, closure use
// or grouped use).
func fqCollectUsesRegion(s *tokens.Stream, reg fqRegion) *fqUses {
	u := fqNewUses()
	depth := 0
	for i := reg.start; i < reg.end && i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind == token.Punct {
			switch t.Value {
			case "{":
				depth++
			case "}":
				depth--
			}
			continue
		}
		if depth != 0 || !kwIs(t, "use") {
			continue
		}
		n := sigNext(s, i)
		if n < 0 {
			continue
		}
		if s.At(n).Kind == token.Keyword {
			lv := strings.ToLower(s.At(n).Value)
			if lv == "function" || lv == "const" {
				continue
			}
		}
		fqcn, alias, end, ok := fqReadImport(s, n)
		if !ok {
			continue
		}
		i = end
		short := alias
		if short == "" {
			if idx := strings.LastIndexByte(fqcn, '\\'); idx >= 0 {
				short = fqcn[idx+1:]
			} else {
				short = fqcn
			}
		}
		u.add(fqcn, short)
	}
	u.build()
	return u
}

// fqReadImport reads `Name\Space[ as Alias];` from the first name token.
func fqReadImport(s *tokens.Stream, start int) (fqcn, alias string, end int, ok bool) {
	var b strings.Builder
	i := start
	if isPunctVal(s, i, "\\") {
		i++
	}
	expectName := true
	for i < s.Len() {
		t := s.At(i)
		switch t.Kind {
		case token.Whitespace, token.Comment, token.DocComment:
			i++
			continue
		}
		if expectName {
			if t.Kind != token.Ident {
				return "", "", 0, false
			}
			b.WriteString(t.Value)
			expectName = false
			i++
			continue
		}
		if t.Kind == token.Punct && t.Value == "\\" {
			b.WriteByte('\\')
			expectName = true
			i++
			continue
		}
		if t.Kind == token.Punct && t.Value == ";" {
			return b.String(), "", i, true
		}
		if kwIs(t, "as") {
			a := sigNext(s, i)
			if a < 0 || s.At(a).Kind != token.Ident {
				return "", "", 0, false
			}
			semi := sigNext(s, a)
			if semi < 0 || !isPunctVal(s, semi, ";") {
				return "", "", 0, false
			}
			return b.String(), s.At(a).Value, semi, true
		}
		return "", "", 0, false
	}
	return "", "", 0, false
}

// fqReadRunForward reads the contiguous Ident/`\` name run starting at start.
func fqReadRunForward(s *tokens.Stream, start int) (content string, end int, ok bool) {
	if start < 0 || start >= s.Len() {
		return "", 0, false
	}
	t := s.At(start)
	if t.Kind != token.Ident && (t.Kind != token.Punct || t.Value != "\\") {
		return "", 0, false
	}
	var b strings.Builder
	i := start
	last := start
	for i < s.Len() {
		ct := s.At(i)
		if ct.Kind == token.Ident {
			b.WriteString(ct.Value)
			last = i
			i++
			continue
		}
		if ct.Kind == token.Punct && ct.Value == "\\" {
			b.WriteByte('\\')
			last = i
			i++
			continue
		}
		break
	}
	return b.String(), last, true
}

// fqReadRunBackward reads the contiguous Ident/`\` name run ending at end.
func fqReadRunBackward(s *tokens.Stream, end int) (content string, start int, ok bool) {
	if end < 0 || end >= s.Len() {
		return "", 0, false
	}
	i := end
	first := end
	for i >= 0 {
		ct := s.At(i)
		if ct.Kind == token.Ident || (ct.Kind == token.Punct && ct.Value == "\\") {
			first = i
			i--
			continue
		}
		break
	}
	var b strings.Builder
	for j := first; j <= end; j++ {
		b.WriteString(s.At(j).Value)
	}
	if b.Len() == 0 {
		return "", 0, false
	}
	return b.String(), first, true
}

// fqNextName shortens the name run immediately after a keyword.
func fqNextName(s *tokens.Stream, kw int, uses *fqUses, ns string, reserved map[string]bool) (fqReplace, bool) {
	n := sigNext(s, kw)
	if n < 0 {
		return fqReplace{}, false
	}
	content, end, ok := fqReadRunForward(s, n)
	if !ok {
		return fqReplace{}, false
	}
	repl, changed := fqDetermineShort(content, uses, ns, reserved)
	if !changed {
		return fqReplace{}, false
	}
	return fqReplace{start: n, end: end, repl: repl}, true
}

// fqPrevName shortens the name run immediately before a token (a variable or `::`).
func fqPrevName(s *tokens.Stream, idx int, uses *fqUses, ns string, reserved map[string]bool) (fqReplace, bool) {
	p := sigPrev(s, idx)
	if p < 0 || s.At(p).Kind != token.Ident {
		return fqReplace{}, false
	}
	content, start, ok := fqReadRunBackward(s, p)
	if !ok {
		return fqReplace{}, false
	}
	// a name after an object operator (`$this->grammar::`) is a member, not a class
	if b := sigPrev(s, start); b >= 0 && s.At(b).Kind == token.Punct && (s.At(b).Value == "->" || s.At(b).Value == "?->") {
		return fqReplace{}, false
	}
	repl, changed := fqDetermineShort(content, uses, ns, reserved)
	if !changed {
		return fqReplace{}, false
	}
	return fqReplace{start: start, end: p, repl: repl}, true
}

// fqExtendsImplements shortens each type in an extends/implements list.
func fqExtendsImplements(s *tokens.Stream, kw int, uses *fqUses, ns string, reserved map[string]bool) []fqReplace {
	var out []fqReplace
	i := sigNext(s, kw)
	for i >= 0 && i < s.Len() {
		t := s.At(i)
		if t.Kind == token.Punct && t.Value == "{" {
			break
		}
		if t.Kind == token.Keyword {
			break // extends X implements Y - the other keyword handles its list
		}
		if t.Kind == token.Ident || (t.Kind == token.Punct && t.Value == "\\") {
			content, end, ok := fqReadRunForward(s, i)
			if ok {
				if repl, changed := fqDetermineShort(content, uses, ns, reserved); changed {
					out = append(out, fqReplace{start: i, end: end, repl: repl})
				}
				i = sigNext(s, end)
				continue
			}
		}
		i = sigNext(s, i)
	}
	return out
}

// fqCatch shortens each exception type in a catch clause.
func fqCatch(s *tokens.Stream, kw int, uses *fqUses, ns string, reserved map[string]bool) []fqReplace {
	open := sigNext(s, kw)
	if open < 0 || !isPunctVal(s, open, "(") {
		return nil
	}
	var out []fqReplace
	i := sigNext(s, open)
	for i >= 0 && i < s.Len() {
		t := s.At(i)
		if (t.Kind == token.Punct && t.Value == ")") || t.Kind == token.Variable {
			break
		}
		if t.Kind == token.Ident || (t.Kind == token.Punct && t.Value == "\\") {
			content, end, ok := fqReadRunForward(s, i)
			if ok {
				if repl, changed := fqDetermineShort(content, uses, ns, reserved); changed {
					out = append(out, fqReplace{start: i, end: end, repl: repl})
				}
				i = sigNext(s, end)
				continue
			}
		}
		i = sigNext(s, i)
	}
	return out
}

// fqFunction shortens parameter and return types of a function/closure/arrow fn.
func fqFunction(s *tokens.Stream, kw, regionEnd int, uses *fqUses, ns string, reserved map[string]bool) []fqReplace {
	open := fqNextPunct(s, kw, "(")
	if open < 0 {
		return nil
	}
	closeIdx := s.MatchForward(open)
	if closeIdx < 0 {
		return nil
	}
	var out []fqReplace
	// parameters: each type run inside (...) that precedes a $var, & or ...
	out = append(out, fqTypeRunsIn(s, open+1, closeIdx-1, uses, ns, reserved, true)...)
	// return type: after `)` an optional `:` then a type run up to `{` or `;`
	c := sigNext(s, closeIdx)
	if c >= 0 && isPunctVal(s, c, ":") {
		te := fqReturnTypeEnd(s, c, regionEnd)
		if te > c {
			out = append(out, fqTypeRunsIn(s, sigNext(s, c), te, uses, ns, reserved, false)...)
		}
	}
	return out
}

func fqReturnTypeEnd(s *tokens.Stream, colon, regionEnd int) int {
	for i := colon + 1; i < s.Len() && i < regionEnd; i++ {
		t := s.At(i)
		if t.Kind == token.Punct && (t.Value == "{" || t.Value == ";") {
			return sigPrev(s, i)
		}
	}
	return -1
}

// fqTypeRunsIn scans a range for class-name atoms in type position: name runs
// that are part of a type (skip runs used as default values, constants, etc.).
// A name run counts when it is preceded by a type boundary (`(` `,` `|` `&` `?`
// `:` start) and, for parameters, followed by `$var`/`&`/`...`/`|`/`&`/`)`.
func fqTypeRunsIn(s *tokens.Stream, from, to int, uses *fqUses, ns string, reserved map[string]bool, params bool) []fqReplace {
	var out []fqReplace
	i := from
	for i >= 0 && i <= to && i < s.Len() {
		t := s.At(i)
		if t.Kind == token.Ident || (t.Kind == token.Punct && t.Value == "\\") {
			// must be a type: previous meaningful token is a type boundary
			p := sigPrev(s, i)
			if p >= 0 && fqIsTypeBoundaryBefore(s, p) {
				content, end, ok := fqReadRunForward(s, i)
				if ok && fqIsTypeBoundaryAfter(s, end, params) {
					if repl, changed := fqDetermineShort(content, uses, ns, reserved); changed {
						out = append(out, fqReplace{start: i, end: end, repl: repl})
					}
					i = sigNext(s, end)
					continue
				}
			}
		}
		i = sigNext(s, i)
	}
	return out
}

func fqIsTypeBoundaryBefore(s *tokens.Stream, p int) bool {
	t := s.At(p)
	if t.Kind == token.Punct {
		switch t.Value {
		case "(", ",", "|", "&", "?", ":":
			return true
		}
	}
	if t.Kind == token.Keyword {
		switch strings.ToLower(t.Value) {
		case "public", "protected", "private", "readonly", "static", "var", "const", "function", "fn":
			return true
		}
	}
	return false
}

func fqIsTypeBoundaryAfter(s *tokens.Stream, end int, params bool) bool {
	n := sigNext(s, end)
	if n < 0 {
		return true
	}
	t := s.At(n)
	if t.Kind == token.Variable {
		return true
	}
	if t.Kind == token.Punct {
		switch t.Value {
		case "|", "&", ")", "{", ";", ",":
			return true
		case "...":
			return true
		}
	}
	// `&` variadic/reference before var
	return false
}

func fqNextPunct(s *tokens.Stream, from int, v string) int {
	for i := from + 1; i < s.Len(); i++ {
		if s.At(i).Kind == token.Punct && s.At(i).Value == v {
			return i
		}
		if s.At(i).Kind == token.Punct && (s.At(i).Value == ";" || s.At(i).Value == "{") {
			return -1
		}
	}
	return -1
}
