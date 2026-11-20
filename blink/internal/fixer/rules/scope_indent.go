package rules

import (
	"math"
	"slices"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/WhiteSpace/ScopeIndentSniff.php
//
// ScopeIndent indents every line by Indent spaces per scope level, a line-by-line
// port of the sniff's single pass from the first open tag.
type ScopeIndent struct {
	Indent    int
	Exact     bool
	TabIndent bool
	// SwitchNotIndented leaves switch bodies unindented, as PEAR's subclass
	// sets $nonIndentingScopes = [T_SWITCH].
	SwitchNotIndented bool
}

func (ScopeIndent) Name() string {
	return `PHP_CodeSniffer\Standards\Generic\Sniffs\WhiteSpace\ScopeIndentSniff`
}

func (ScopeIndent) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/WhiteSpace/ScopeIndentSniff.php"
}

func (r ScopeIndent) WithConfig(config map[string]any) fixer.Fixer {
	r.Indent = configInt(config, "indent", r.Indent)
	if v, ok := config["exact"].(bool); ok {
		r.Exact = v
	}
	if v, ok := config["tabIndent"].(bool); ok {
		r.TabIndent = v
	}
	return r
}

func (ScopeIndent) phpcsOnePass() {}

func (r ScopeIndent) nonIndenting(code string) bool {
	return r.SwitchNotIndented && code == "switch"
}

func (r ScopeIndent) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		m := newSiModel(f)
		for i := range f.toks {
			if m.code[i] == "open_tag" {
				r.process(m, i)
				return
			}
		}
	})
}

// siModel is the PHP_CodeSniffer token data the sniff reads: refined token
// codes, conditions and parenthesis owners on top of the scope map.
type siModel struct {
	f      *slvFile
	sc     *phpcsScope
	code   []string
	conds  [][]phpcsCond
	levels []int
	pOwner []int
	docEnd []int
}

func newSiModel(f *slvFile) *siModel {
	sc := newPhpcsScope(f)
	n := len(f.toks)
	m := &siModel{f: f, sc: sc, code: make([]string, n), pOwner: make([]int, n), docEnd: make([]int, n)}
	for i := range n {
		m.code[i] = m.classify(i)
		m.pOwner[i] = -1
		m.docEnd[i] = -1
	}
	m.conds, m.levels = sc.levelMap()
	for i := range n {
		if !f.is(i, "(") || f.toks[i].closer < 0 {
			continue
		}
		owner := sc.pOwner[i]
		if owner < 0 {
			p := f.prevEffective(i - 1)
			if f.is(p, "&") {
				p = f.prevEffective(p - 1)
			}
			if p >= 0 && (sc.code[p] == "fn" || m.code[p] == "anon_class" || siCallOwners[strings.ToLower(f.toks[p].orig)] && !siIsMember(f, p)) {
				owner = p
			}
		}
		m.pOwner[i], m.pOwner[f.toks[i].closer] = owner, owner
	}
	open := -1
	for i := range n {
		switch f.toks[i].doc {
		case slvDocOpen:
			open = i
		case slvDocClose:
			if open >= 0 {
				m.docEnd[open] = i
				open = -1
			}
		}
	}
	return m
}

func (m *siModel) classify(i int) string {
	f := m.f
	t := f.toks[i]
	switch t.kind {
	case token.String:
		switch {
		case t.heredoc && (i == 0 || !f.toks[i-1].heredoc):
			return "start_heredoc"
		case t.heredoc && (i+1 == len(f.toks) || !f.toks[i+1].heredoc):
			return "end_heredoc"
		case t.heredoc:
			return "heredoc"
		case t.interp:
			return "double_quoted_string"
		}
		return "constant_encapsed_string"
	case token.DocComment:
		switch t.doc {
		case slvDocOpen:
			return "doc_open"
		case slvDocWhitespace:
			return "doc_whitespace"
		}
		return "doc"
	case token.OpenTag:
		if strings.HasPrefix(t.orig, "<?=") {
			return "open_tag_with_echo"
		}
		return "open_tag"
	case token.Punct:
		switch {
		case t.orig == "[" && f.isShortArrayOpen(i):
			return "open_short_array"
		case t.orig == "]" && t.closer >= 0 && f.isShortArrayOpen(t.closer):
			return "close_short_array"
		}
	case token.Keyword:
		switch strings.ToLower(t.orig) {
		case "function":
			if f.isClosure(i) {
				return "closure"
			}
		case "class":
			if phpcsIsAnonClassKeyword(f, i) {
				return "anon_class"
			}
		}
	}
	return m.sc.code[i]
}

// siCallOwners are the language constructs PHP_CodeSniffer gives the
// parentheses after them, whatever blink lexes them as.
var siCallOwners = phpcsSet("eval", "isset", "empty", "unset", "array", "list")

func siIsMember(f *slvFile, i int) bool {
	p := f.prevEffective(i - 1)
	return f.is(p, "->") || f.is(p, "?->") || f.is(p, "::") || f.isKeyword(p, "function")
}

var siScopeOpeners = phpcsSet("class", "anon_class", "interface", "trait", "enum", "namespace", "function", "closure",
	"if", "switch", "case", "declare", "default", "while", "else", "elseif", "for", "foreach", "do", "try", "catch",
	"finally", "use", "match")

var siMethodModifiers = phpcsSet("private", "public", "protected", "abstract", "static", "final")

func (m *siModel) lastCond(i int) int {
	if c := m.conds[i]; len(c) > 0 {
		return c[len(c)-1].key
	}
	return -1
}

// firstOnLine is File::findFirstOnLine with exclude: the first token on the
// line of start whose code is not in skip, or -1.
func (m *siModel) firstOnLine(start int, skip ...string) int {
	found := -1
	for i := start; i >= 0; i-- {
		if m.f.toks[i].line < m.f.toks[start].line {
			break
		}
		if !slices.Contains(skip, m.code[i]) {
			found = i
		}
	}
	return found
}

// siAt maps PHP's $tokens[false] to the first token.
func siAt(i int) int { return max(i, 0) }

func (m *siModel) nextNonEmpty(i int) int {
	for ; i < len(m.f.toks); i++ {
		if !m.f.isEmpty(i) {
			return i
		}
	}
	return -1
}

// prevLocal is File::findPrevious with local set.
func (m *siModel) prevLocal(start int, types ...string) int {
	f, sc := m.f, m.sc
	for i := start; i >= 0; i-- {
		if slices.Contains(types, m.code[i]) {
			return i
		}
		t := f.toks[i]
		switch {
		case sc.opener[i] >= 0 && i == sc.closer[i]:
			i = sc.opener[i]
		case (f.is(i, "]") || f.is(i, "}")) && t.closer >= 0 && t.closer < i:
			i = t.closer
		case f.is(i, ")") && t.closer >= 0 && t.closer < i:
			i = t.closer
		case m.code[i] == ";":
			return -1
		}
	}
	return -1
}

func (r ScopeIndent) ceil(v int) int {
	return int(math.Ceil(float64(v)/float64(r.Indent))) * r.Indent
}

func (r ScopeIndent) floor(v int) int {
	return int(math.Floor(float64(v)/float64(r.Indent))) * r.Indent
}

// siOrdered is a PHP array keyed by scope closer, kept in insertion order.
type siOrdered struct {
	keys []int
	vals map[int]int
}

func (o *siOrdered) set(k, v int) {
	if _, ok := o.vals[k]; !ok {
		o.keys = append(o.keys, k)
	}
	o.vals[k] = v
}

func (o *siOrdered) has(k int) bool {
	_, ok := o.vals[k]
	return ok
}

func (o *siOrdered) pop() int {
	if len(o.keys) == 0 {
		return -1
	}
	k := o.keys[len(o.keys)-1]
	o.keys = o.keys[:len(o.keys)-1]
	v := o.vals[k]
	delete(o.vals, k)
	return v
}

func (o *siOrdered) last() int {
	if len(o.keys) == 0 {
		return -1
	}
	return o.vals[o.keys[len(o.keys)-1]]
}

func (r ScopeIndent) process(m *siModel, stackPtr int) {
	f, sc, code := m.f, m.sc, m.code
	toks := f.toks
	n := len(toks)
	tabWidth := slvTabWidth

	lastOpenTag := stackPtr
	lastCloseTag := -1
	openScopes := &siOrdered{vals: map[int]int{}}
	adjustments := map[int]int{}
	setIndents := map[int]int{}
	disableExactStack := map[int]bool{}
	disableExactEnd := 0

	currentIndent := 0
	first := -1
	for i := stackPtr; i >= 0 && toks[i].line == toks[stackPtr].line; i-- {
		if code[i] == "inline_html" {
			first = i
		}
	}
	first = siAt(first)
	if trimmed := phpcsLtrim(toks[first].content); trimmed == "" {
		currentIndent = toks[stackPtr].column - 1
	} else {
		currentIndent = len(toks[first].content) - len(trimmed)
	}

	tokenIndent := 0
	for i := stackPtr + 1; i < n; i++ {
		checkToken := -1
		checkIndent := -1
		hasCheckIndent := false
		exact := r.Exact

		if f.is(i, "(") && toks[i].closer >= 0 {
			disableExactStack[toks[i].closer] = true
			disableExactEnd = max(disableExactEnd, toks[i].closer)
		}
		if exact && i < disableExactEnd {
			exact = false
		}

		if toks[i].column == 1 {
			trimmed := phpcsLtrim(toks[i].content)
			if trimmed == "" {
				if i+1 < n && toks[i].line == toks[i+1].line {
					checkToken = i + 1
					tokenIndent = toks[i+1].column - 1
				}
			} else {
				checkToken = i
				tokenIndent = len(toks[i].content) - len(trimmed)
			}
		}

		if checkToken >= 0 && f.is(checkToken, ")") && toks[checkToken].closer >= 0 ||
			f.is(i, ")") && toks[i].closer >= 0 {
			parenCloser := i
			if checkToken >= 0 {
				parenCloser = checkToken
			}
			parenOpener := toks[parenCloser].closer
			if toks[parenCloser].line != toks[parenOpener].line {
				parens := 0
				if nested := sc.nested[parenCloser]; nested >= 0 {
					parens = nested
				}
				condition := 0
				if lc := m.lastCond(parenCloser); lc >= 0 && (m.pOwner[parenCloser] < 0 || parens > 0) {
					condition = lc
				}
				if parens > condition {
					parenOpener = parens
					condition = 0
				} else if condition > 0 {
					parenOpener = condition
					parens = 0
				}
				exact = false

				lastOpenTagCondition := m.lastCond(lastOpenTag)
				switch {
				case condition > 0 && lastOpenTagCondition == condition:
					first = siAt(m.firstOnLine(lastOpenTag, "whitespace", "inline_html"))
					checkIndent, hasCheckIndent = toks[first].column-1, true
					checkIndent += adjustments[condition]
					currentIndent = checkIndent
				case condition > 0 && sc.opener[condition] >= 0 && siHas(setIndents, sc.opener[condition]):
					checkIndent, hasCheckIndent = setIndents[sc.opener[condition]], true
					checkIndent += adjustments[condition]
					currentIndent = checkIndent
				default:
					first = siAt(m.firstOnLine(parenOpener, "whitespace"))
					checkIndent, hasCheckIndent = toks[first].column-1, true
					checkIndent += adjustments[first]
					if first == toks[parenCloser].closer && toks[siAt(first-1)].line == toks[first].line {
						first--
					}
					prev := sc.startOfStatementIgnoring(first, ",")
					if prev != first {
						first = m.firstOnLine(prev, "whitespace", "inline_html")
						if first >= 0 {
							prev = sc.startOfStatementIgnoring(first, ",")
							first = siAt(m.firstOnLine(prev, "whitespace", "inline_html"))
						} else {
							first = prev
						}
					}
					if sc.closer[first] >= 0 && sc.closer[first] == first {
						if sc.cond[first] >= 0 {
							scopeCloser := first
							first = siAt(m.firstOnLine(sc.cond[scopeCloser], "whitespace"))
							currentIndent = toks[first].column - 1
							currentIndent += adjustments[first]
							if code[sc.cond[scopeCloser]] != "closure" {
								currentIndent = r.ceil(currentIndent)
							}
							setIndents[first] = currentIndent
						}
					} else {
						currentIndent = toks[first].column - 1
						currentIndent += adjustments[first]
						setIndents[first] = currentIndent
					}
				}
			}
		}

		if code[i] == "close_short_array" || checkToken >= 0 && code[checkToken] == "close_short_array" {
			arrayCloser := i
			if checkToken >= 0 {
				arrayCloser = checkToken
			}
			arrayOpener := toks[arrayCloser].closer
			if toks[arrayCloser].line != toks[arrayOpener].line {
				first = siAt(m.firstOnLine(arrayOpener, "whitespace"))
				exact = false
				if first == arrayOpener {
					first--
				}
				prev := sc.startOfStatementIgnoring(first, ",", "=>")
				if prev != first {
					first = siAt(m.firstOnLine(prev, "whitespace"))
					prev = sc.startOfStatementIgnoring(first, ",", "=>")
					first = siAt(m.firstOnLine(prev, "whitespace"))
				} else if code[first] == "whitespace" {
					first = siAt(m.nextNonWhitespace(first + 1))
				}
				checkIndent, hasCheckIndent = toks[first].column-1, true
				checkIndent += adjustments[first]
				if sc.closer[first] >= 0 && sc.closer[first] == first {
					if v, ok := setIndents[first]; ok {
						currentIndent = v
					}
				} else {
					currentIndent = toks[first].column - 1
					currentIndent += adjustments[first]
					setIndents[first] = currentIndent
				}
			}
		}

		if checkToken >= 0 && !exact && (len(m.conds[checkToken]) > 0 || sc.opener[checkToken] >= 0 && sc.opener[checkToken] == checkToken) {
			condition := sc.cond[checkToken]
			if len(m.conds[checkToken]) > 0 {
				condition = m.lastCond(checkToken)
			}
			first = siAt(m.firstOnLine(condition, "whitespace"))
			// the sniff re-indents the line here only when $fixer->enabled, which
			// ECS never sets
			if adj, ok := adjustments[first]; ok && (adj < 0 && tokenIndent > currentIndent || adj > 0 && tokenIndent < currentIndent) {
				adjustments[checkToken] = adj
			}
		}

		if checkToken >= 0 && (openScopes.has(checkToken) ||
			sc.cond[checkToken] >= 0 && sc.closer[checkToken] == checkToken && sc.opener[checkToken] >= 0 &&
				toks[checkToken].line != toks[sc.opener[checkToken]].line) ||
			checkToken < 0 && openScopes.has(i) {
			scopeCloser := checkToken
			if scopeCloser < 0 {
				scopeCloser = i
			}
			openScopes.pop()
			if sc.cond[scopeCloser] >= 0 {
				first = siAt(m.firstOnLine(sc.cond[scopeCloser], "whitespace", "inline_html"))
				for code[first] == "constant_encapsed_string" && first > 0 && code[first-1] == "constant_encapsed_string" {
					first = siAt(m.firstOnLine(first-1, "whitespace"))
				}
				currentIndent = toks[first].column - 1
				currentIndent += adjustments[first]
				setIndents[scopeCloser] = currentIndent
				if f.is(scopeCloser, "}") {
					exact = true
				} else {
					checkToken = -1
				}
			}
		}

		if checkToken >= 0 && siScopeOpeners[code[checkToken]] && !r.nonIndenting(code[checkToken]) && sc.opener[checkToken] >= 0 {
			exact = true
			if disableExactEnd > checkToken {
				for end := range disableExactStack {
					if end < checkToken {
						continue
					}
					if slices.Equal(m.conds[checkToken], m.conds[end]) {
						exact = false
						break
					}
				}
			}
			lastOpener := openScopes.last()
			if lastOpener >= 0 && sc.closer[lastOpener] >= 0 &&
				m.levels[lastOpener] == m.levels[checkToken] &&
				sc.closer[lastOpener] == sc.closer[checkToken] {
				currentIndent -= r.Indent
				setIndents[lastOpener] = currentIndent
			}
			if code[checkToken] == "closure" && tokenIndent > currentIndent {
				checkIndent, hasCheckIndent = r.ceil(tokenIndent), true
				exact = false
			}
		}

		if checkToken >= 0 && siMethodModifiers[code[checkToken]] && (checkToken+1 >= n || code[checkToken+1] != "::") {
			next := m.nextNonEmpty(checkToken + 1)
			if next < 0 || code[next] != "closure" && code[next] != "variable" && code[next] != "fn" {
				isMethodPrefix := true
				if deepest := sc.nested[checkToken]; deepest >= 0 {
					if owner := m.pOwner[deepest]; owner >= 0 && code[owner] == "function" {
						isMethodPrefix = false
					}
				}
				if isMethodPrefix {
					exact = true
				}
			}
		}

		if checkToken >= 0 && (code[checkToken] == "open_tag" || code[checkToken] == "open_tag_with_echo") {
			checkIndent, hasCheckIndent = toks[checkToken].column-1, true
			if len(m.conds[checkToken]) > 0 {
				closeTag := -1
				for j := checkToken - 1; j >= 0; j-- {
					if code[j] == "close_tag" {
						closeTag = j
						break
					}
				}
				if closeTag >= 0 && slices.Equal(m.conds[checkToken], m.conds[closeTag]) {
					lastCondition := m.lastCond(checkToken)
					lastOpener, lastCloser := sc.opener[lastCondition], sc.closer[lastCondition]
					if v, ok := setIndents[lastOpener]; ok && toks[siAt(lastCloser)].line != toks[checkToken].line {
						checkIndent = v
					}
				}
			}
		}

		if checkToken >= 0 && code[checkToken] == "close_tag" {
			exact = true
			checkIndent, hasCheckIndent = r.ceil(currentIndent), true
		}

		if checkToken >= 0 && code[checkToken] == "else" {
			exact = true
		}

		if checkToken >= 0 && (code[checkToken] == "->" || code[checkToken] == "?->") && exact {
			exact = false
		}

		if !hasCheckIndent {
			checkIndent = currentIndent
		}

		if checkToken >= 0 && code[checkToken] != "inline_html" &&
			(tokenIndent != checkIndent && exact || tokenIndent < checkIndent && !exact) {
			adjustments[checkToken] = checkIndent - tokenIndent
			r.adjustIndent(m, checkToken, checkIndent, checkIndent-tokenIndent, tabWidth)
		}

		if checkToken >= 0 {
			i = checkToken
		}

		if code[i] == "open_short_array" {
			disableExactStack[toks[i].closer] = true
			disableExactEnd = max(disableExactEnd, toks[i].closer)
		}

		if code[i] == "start_heredoc" {
			end := -1
			for j := i + 1; j < n; j++ {
				if code[j] == "end_heredoc" {
					end = j
					break
				}
			}
			if end < 0 {
				return
			}
			i = end
			if next := m.nextNonEmpty(i + 1); code[siAt(next)] == "," {
				i = siAt(next)
			}
			continue
		}

		if code[i] == "constant_encapsed_string" || code[i] == "double_quoted_string" {
			for j := i + 1; j < n; j++ {
				if code[j] != code[i] {
					i = j - 1
					break
				}
			}
			continue
		}

		if code[i] == "doc_open" {
			if m.docEnd[i] >= 0 {
				i = m.docEnd[i]
			}
			continue
		}

		if code[i] == "open_tag" || code[i] == "open_tag_with_echo" {
			if checkToken < 0 {
				first = siAt(m.firstOnLine(i, "whitespace"))
				currentIndent = len(toks[first].content) - len(phpcsLtrim(toks[first].content))
			} else {
				currentIndent = toks[i].column - 1
			}
			lastOpenTag = i
			currentIndent += adjustments[i]
			currentIndent = r.ceil(currentIndent)
			setIndents[i] = currentIndent
			continue
		}

		if code[i] == "close_tag" {
			if toks[lastOpenTag].line != toks[i].line {
				currentIndent = toks[i].column - 1
				lastCloseTag = i
			} else if lastCloseTag < 0 {
				currentIndent = 0
			} else {
				currentIndent = toks[lastCloseTag].column - 1
			}
			currentIndent += adjustments[i]
			currentIndent = r.ceil(currentIndent)
			setIndents[i] = currentIndent
			continue
		}

		if code[i] == "closure" || code[i] == "anon_class" {
			closer := sc.closer[i]
			if closer < 0 {
				continue
			}
			if toks[i].line == toks[closer].line {
				i = closer
				continue
			}
			first = siAt(m.firstOnLine(i, "whitespace"))
			for code[first] == "constant_encapsed_string" && first > 0 && code[first-1] == "constant_encapsed_string" {
				first = siAt(m.firstOnLine(first-1, "whitespace"))
			}
			currentIndent = toks[first].column - 1 + r.Indent
			openScopes.set(closer, sc.cond[i])
			currentIndent += adjustments[first]
			currentIndent = r.floor(currentIndent)
			i = sc.opener[i]
			setIndents[i] = currentIndent
			continue
		}

		if sc.cond[i] >= 0 && sc.opener[i] >= 0 && sc.opener[i] == i {
			closer := sc.closer[i]
			if toks[i].line == toks[siAt(closer)].line {
				i = closer
				continue
			}
			if siScopeOpeners[code[sc.cond[i]]] && !r.nonIndenting(code[sc.cond[i]]) {
				currentIndent += r.Indent
				setIndents[i] = currentIndent
				openScopes.set(closer, sc.cond[i])
				continue
			}
		}

		if sc.cond[i] >= 0 && sc.closer[i] == i {
			switch code[sc.cond[i]] {
			case "closure", "anon_class", "match", "fn":
			default:
				continue
			}
			prev := -1
			parens := 0
			if nested := sc.nested[i]; nested >= 0 {
				parens = nested
			}
			condition := 0
			if lc := m.lastCond(i); lc >= 0 {
				condition = lc
			}
			if parens > condition {
				prev = f.prevEffective(parens - 1)
				condition = 0
			} else if condition > 0 {
				prev = condition
				parens = 0
			}
			if prev < 0 {
				prev = m.prevLocal(sc.cond[i]-1, "=", "return")
				if prev < 0 {
					prev = i
				}
			}
			first = siAt(m.firstOnLine(prev, "whitespace"))
			prev = sc.startOfStatementIgnoring(first)
			if prev != first {
				first = siAt(m.firstOnLine(prev, "whitespace"))
			}
			currentIndent = toks[first].column - 1
			if condition > 0 {
				currentIndent += r.Indent
			}
			if sc.closer[first] >= 0 && sc.closer[first] == first {
				if condition == 0 || sc.opener[condition] < first {
					currentIndent = setIndents[first]
				}
			}
			currentIndent = r.ceil(currentIndent)
			setIndents[first] = currentIndent
		}
	}
}

func siHas(m map[int]int, k int) bool {
	_, ok := m[k]
	return ok
}

func (m *siModel) nextNonWhitespace(i int) int {
	for ; i < len(m.code); i++ {
		if m.code[i] != "whitespace" {
			return i
		}
	}
	return -1
}

func (r ScopeIndent) padding(length, tabWidth int) string {
	if length <= 0 {
		return ""
	}
	if !r.TabIndent {
		return strings.Repeat(" ", length)
	}
	numTabs := length / tabWidth
	if numTabs == 0 {
		return ""
	}
	return strings.Repeat("\t", numTabs) + strings.Repeat(" ", length-numTabs*tabWidth)
}

func (r ScopeIndent) adjustIndent(m *siModel, stackPtr, length, change, tabWidth int) {
	f := m.f
	toks := f.toks
	if m.code[stackPtr] == "inline_html" {
		return
	}
	padding := r.padding(length, tabWidth)
	var accepted bool
	if toks[stackPtr].column == 1 {
		accepted = f.replaceToken(stackPtr, padding+phpcsLtrim(toks[stackPtr].content))
	} else {
		accepted = f.replaceToken(stackPtr-1, padding)
	}
	if !accepted || m.code[stackPtr] != "doc_open" {
		return
	}
	for x := stackPtr + 1; x < m.docEnd[stackPtr]; x++ {
		if toks[x].column != 1 {
			continue
		}
		l := 0
		if m.code[x] == "doc_whitespace" {
			l = toks[x].length
		}
		pad := ""
		if p := l + change; p > 0 {
			if r.TabIndent {
				numTabs := p / tabWidth
				pad = strings.Repeat("\t", numTabs) + strings.Repeat(" ", p-numTabs*tabWidth)
			} else {
				pad = strings.Repeat(" ", p)
			}
		}
		f.replaceToken(x, pad)
	}
}
