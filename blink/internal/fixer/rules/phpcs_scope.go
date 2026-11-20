package rules

import (
	"slices"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
)

// phpcsScope is PHP_CodeSniffer's scope map of a file: the scope_condition,
// scope_opener and scope_closer its tokenizer gives each token (-1 when
// unset), built by a port of Tokenizer::recurseScopeMap and the CASE/DEFAULT
// fixes of PHP::processAdditional.
type phpcsScope struct {
	f       *slvFile
	code    []string
	cond    []int
	opener  []int
	closer  []int
	pCloser []int // parenthesis_closer of a parenthesis owner
	pOwner  []int // parenthesis_owner of a "("
	nested  []int // the innermost "(" holding the token
	scopes  []phpcsScopeRange
	// the scope map as createLevelMap sees it, before processAdditional
	preCond, preOpener, preCloser []int
	// CASE/DEFAULT statements fixCases gave a braced body: case, new closer, old closer
	caseFixes [][3]int
}

// phpcsScopeRange is a scope the level map counts: the tokens strictly
// between opener and closer have cond among their conditions.
type phpcsScopeRange struct{ cond, opener, closer int }

type phpcsScopeOpener struct {
	start, end, with map[string]bool
	strict, shared   bool
}

func phpcsSet(v ...string) map[string]bool {
	m := make(map[string]bool, len(v))
	for _, s := range v {
		m[s] = true
	}
	return m
}

var phpcsScopeOpeners = func() map[string]phpcsScopeOpener {
	brace := func(strict bool) phpcsScopeOpener {
		return phpcsScopeOpener{start: phpcsSet("{"), end: phpcsSet("}"), with: phpcsSet(), strict: strict}
	}
	alt := func(end string, with ...string) phpcsScopeOpener {
		return phpcsScopeOpener{start: phpcsSet("{", ":"), end: phpcsSet("}", end), with: phpcsSet(with...)}
	}
	caseEnds := phpcsSet("break", "return", "continue", "throw", "exit", "goto")
	m := map[string]phpcsScopeOpener{
		"try": brace(true), "catch": brace(true), "finally": brace(true), "interface": brace(true),
		"function": brace(true), "class": brace(true), "trait": brace(true), "enum": brace(true),
		"do": brace(true), "match": brace(true), "use": brace(false), "namespace": brace(false),
		"for": alt("endfor"), "foreach": alt("endforeach"), "declare": alt("enddeclare"), "while": alt("endwhile"),
		"if":            {start: phpcsSet("{", ":"), end: phpcsSet("}", "endif", "else", "elseif"), with: phpcsSet("else", "elseif")},
		"elseif":        {start: phpcsSet("{", ":"), end: phpcsSet("}", "endif", "else", "elseif"), with: phpcsSet("if", "else")},
		"else":          {start: phpcsSet("{", ":"), end: phpcsSet("}", "endif"), with: phpcsSet("if", "elseif")},
		"switch":        {start: phpcsSet("{", ":"), end: phpcsSet("}", "endswitch"), with: phpcsSet(), strict: true},
		"case":          {start: phpcsSet(":", ";", "close_tag"), end: caseEnds, with: phpcsSet("default", "case", "switch"), strict: true, shared: true},
		"default":       {start: phpcsSet(":", ";", "close_tag"), end: caseEnds, with: phpcsSet("case", "switch"), strict: true, shared: true},
		"start_heredoc": {start: phpcsSet("start_heredoc"), end: phpcsSet("end_heredoc"), with: phpcsSet(), strict: true},
	}
	return m
}()

// phpcsScopeOpenerCodes is Tokens::SCOPE_OPENERS.
var phpcsScopeOpenerCodes = phpcsSet("class", "interface", "trait", "enum", "namespace", "function", "if", "switch",
	"case", "declare", "default", "while", "else", "elseif", "for", "foreach", "do", "try", "catch", "finally", "use", "match")

var phpcsEndScopeTokens = phpcsSet("}", "endif", "endfor", "endforeach", "endwhile", "endswitch", "enddeclare", "break")

var phpcsParenOpeners = phpcsSet("array", "list", "function", "use", "while", "for", "foreach", "switch", "if",
	"elseif", "catch", "declare", "match", "isset", "empty", "unset", "eval", "exit")

var phpcsAssignDisallowed = phpcsSet("=", "&=", "|=", ".=", "/=", "-=", "**=", "%=", "*=", "+=", "^=", "=>",
	"<<=", ">>=", "??=", "$", "variable", "->", "?->", ",", "(")

func newPhpcsScope(f *slvFile) *phpcsScope {
	n := len(f.toks)
	sc := &phpcsScope{f: f, code: make([]string, n), cond: make([]int, n), opener: make([]int, n),
		closer: make([]int, n), pCloser: make([]int, n), pOwner: make([]int, n), nested: make([]int, n)}
	insideUseGroup := false
	for i := range n {
		sc.cond[i], sc.opener[i], sc.closer[i], sc.pCloser[i], sc.pOwner[i] = -1, -1, -1, -1, -1
		sc.code[i] = sc.classify(i)
		switch sc.code[i] {
		case "open_use_group":
			insideUseGroup = true
		case "close_use_group":
			insideUseGroup = false
		case "function", "const":
			// "use function" and "use const" import keywords are T_STRING
			if insideUseGroup || f.isKeyword(f.prevEffective(i-1), "use") {
				sc.code[i] = "string"
			}
		}
	}
	sc.inlineElses()
	sc.parentheses()
	for i := 0; i < n; i++ {
		if _, ok := phpcsScopeOpeners[sc.code[i]]; ok {
			if sc.cond[i] >= 0 {
				continue
			}
			ignore := 0
			i = sc.recurse(i, 1, &ignore)
		}
	}
	sc.preCond, sc.preOpener, sc.preCloser = slices.Clone(sc.cond), slices.Clone(sc.opener), slices.Clone(sc.closer)
	sc.fixCases()
	for i := range n {
		if sc.cond[i] == i && sc.opener[i] > i && sc.closer[i] > sc.opener[i] {
			sc.scopes = append(sc.scopes, phpcsScopeRange{i, sc.opener[i], sc.closer[i]})
		}
	}
	sc.arrows()
	return sc
}

// arrows gives arrow functions their scope and retags their "=>" and that of
// match arms, as PHP::processAdditional does after the level map.
func (sc *phpcsScope) arrows() {
	f := sc.f
	for i := len(f.toks) - 1; i >= 0; i-- {
		if !f.isKeyword(i, "fn") {
			continue
		}
		end := f.arrowEnd(i)
		if end < 0 {
			continue
		}
		arrow := i + 1
		for arrow < end && (!f.is(arrow, "=>") || sc.nested[arrow] != sc.nested[i]) {
			arrow++
		}
		sc.code[i], sc.code[arrow] = "fn", "fn_arrow"
		sc.set(i, i, arrow, end)
		sc.set(arrow, i, arrow, end)
		sc.set(end, i, arrow, end)
	}
	for _, r := range sc.scopes {
		if sc.code[r.cond] != "match" {
			continue
		}
		for j := r.opener + 1; j < r.closer; j++ {
			switch {
			case sc.code[j] == "fn" && sc.closer[j] > j:
				j = sc.closer[j]
			case (f.is(j, "(") || f.is(j, "[") || f.is(j, "{")) && f.toks[j].closer > j:
				j = f.toks[j].closer
			case f.is(j, "=>"):
				sc.code[j] = "match_arrow"
			}
		}
	}
}

// conditions lists the scope conditions holding i, outermost first.
func (sc *phpcsScope) conditions(i int) []int {
	var out []int
	for _, r := range sc.scopes {
		if r.opener < i && i < r.closer {
			out = append(out, r.cond)
		}
	}
	return out
}

func (sc *phpcsScope) isEnd(i int) bool {
	switch sc.code[i] {
	case "close_tag", ":", ",", "=>", "match_arrow", ";":
		return true
	}
	return false
}

// startOfStatement is File::findStartOfStatement.
func (sc *phpcsScope) startOfStatement(start int) int {
	return sc.startOfStatementIgnoring(start)
}

// startOfStatementIgnoring is File::findStartOfStatement with its $ignore codes.
func (sc *phpcsScope) startOfStatementIgnoring(start int, ignore ...string) int {
	f := sc.f
	isEnd := func(i int) bool { return sc.isEnd(i) && !slices.Contains(ignore, sc.code[i]) }
	if conds := sc.conditions(start); len(conds) > 0 {
		match := conds[len(conds)-1]
		if sc.code[match] == "match" && sc.nested[match] == sc.nested[start] {
			if v, ok := sc.matchStart(start, match); ok {
				return v
			}
		}
	}
	for isEnd(start) || sc.cond[start] >= 0 && start == sc.closer[start] {
		if sc.cond[start] >= 0 {
			start = sc.cond[start]
		} else {
			start--
		}
	}
	lastNotEmpty := start
	for i := start; i >= 0; i-- {
		switch sc.code[i] {
		case "{", "[", "(", "open_tag":
			if !slices.Contains(ignore, sc.code[i]) {
				return lastNotEmpty
			}
		}
		if isEnd(i) {
			return lastNotEmpty
		}
		if sc.opener[i] >= 0 && i == sc.closer[i] {
			switch sc.code[i] {
			case ")", "end_heredoc", "break", "return", "continue", "throw", "exit", "goto":
			default:
				return lastNotEmpty
			}
		}
		switch {
		case (f.is(i, "]") || f.is(i, "}")) && f.toks[i].closer >= 0 && f.toks[i].closer < i:
			i = f.toks[i].closer
		case f.is(i, ")") && f.toks[i].closer >= 0 && f.toks[i].closer < i:
			i = f.toks[i].closer
		case sc.code[i] == "close_use_group":
			for j := i - 1; j >= 0; j-- {
				if sc.code[j] == "open_use_group" {
					i = j
					break
				}
			}
		}
		if !f.isEmpty(i) {
			lastNotEmpty = i
		}
	}
	return 0
}

func (sc *phpcsScope) matchStart(start, match int) (int, bool) {
	f := sc.f
	lastComma := -1
	prevMatch := start
	for ; prevMatch > sc.opener[match]; prevMatch-- {
		if prevMatch != start && sc.code[prevMatch] == "match_arrow" {
			break
		}
		if prevMatch != start && sc.code[prevMatch] == "," {
			lastComma = prevMatch
			continue
		}
		t := f.toks[prevMatch]
		if (f.is(prevMatch, "]") || f.is(prevMatch, "}")) && t.closer >= 0 && t.closer < prevMatch {
			prevMatch = t.closer
			continue
		}
		if f.is(prevMatch, ")") && t.closer >= 0 && t.closer < prevMatch {
			prevMatch = t.closer
			continue
		}
		if (f.is(prevMatch, "[") || f.is(prevMatch, "{")) && t.closer > start {
			return 0, false
		}
	}
	if prevMatch <= sc.opener[match] {
		if next := f.nextEffective(sc.opener[match]+1, len(f.toks)); next >= 0 {
			return next, true
		}
		return start, true
	}
	if lastComma > prevMatch {
		prevMatch = lastComma
	}
	if next := f.nextEffective(prevMatch+1, len(f.toks)); next >= 0 {
		return next, true
	}
	return start, true
}

// endOfStatement is File::findEndOfStatement.
func (sc *phpcsScope) endOfStatement(start int) int {
	f := sc.f
	if sc.code[start] != "match_arrow" {
		for _, c := range sc.conditions(start) {
			if sc.code[c] != "match" {
				continue
			}
			beforeArrow := true
			prevArrow := -1
			for j := start - 1; j >= sc.opener[c]; j-- {
				if sc.code[j] == "match_arrow" {
					prevArrow = j
					break
				}
			}
			if prevArrow >= 0 {
				beforeArrow = false
				for j := prevArrow + 1; j < start; j++ {
					if sc.code[j] == "," {
						beforeArrow = true
						break
					}
				}
			}
			if beforeArrow {
				for j := start + 1; j < sc.closer[c]; j++ {
					if sc.code[j] == "match_arrow" {
						start = j
						break
					}
				}
			}
			break
		}
	}
	lastNotEmpty := start
	for i := start; i < len(f.toks); i++ {
		if i != start {
			switch sc.code[i] {
			case ")", "]", "}", "open_tag", "close_tag":
				return lastNotEmpty
			case ":", ",", "=>", ";":
				return i
			}
		}
		t := f.toks[i]
		switch {
		case sc.closer[i] >= 0 && (i == sc.opener[i] || i == sc.cond[i]):
			if sc.code[i] == "fn" {
				lastNotEmpty = sc.closer[i]
				i = sc.closer[i] - 1
				continue
			}
			if i == start && phpcsScopeOpenerCodes[sc.code[i]] {
				return sc.closer[i]
			}
			i = sc.closer[i]
		case (f.is(i, "[") || f.is(i, "{") || f.is(i, "(")) && t.closer > i:
			i = t.closer
		case sc.code[i] == "open_use_group":
			for j := i + 1; j < len(f.toks); j++ {
				if sc.code[j] == "close_use_group" {
					i = j
					break
				}
			}
		}
		if !f.isEmpty(i) {
			lastNotEmpty = i
		}
	}
	return len(f.toks) - 1
}

func (sc *phpcsScope) classify(i int) string {
	f := sc.f
	t := f.toks[i]
	switch t.kind {
	case token.Whitespace:
		return "whitespace"
	case token.Comment, token.DocComment:
		return "comment"
	case token.OpenTag:
		return "open_tag"
	case token.CloseTag:
		return "close_tag"
	case token.InlineHTML:
		return "inline_html"
	case token.Variable:
		return "variable"
	case token.Punct:
		if t.orig == "{" && t.closer >= 0 {
			if p := f.prevEffective(i - 1); p >= 0 && strings.HasSuffix(f.toks[p].orig, `\`) {
				return "open_use_group"
			}
		}
		if t.orig == "}" && t.closer >= 0 && t.closer < i && sc.code[t.closer] == "open_use_group" {
			return "close_use_group"
		}
		return t.orig
	case token.String:
		switch {
		case t.heredoc && (i == 0 || !f.toks[i-1].heredoc):
			return "start_heredoc"
		case t.heredoc && (i+1 == len(f.toks) || !f.toks[i+1].heredoc):
			return "end_heredoc"
		}
	case token.Ident:
		switch strings.ToLower(strings.TrimPrefix(t.orig, `\`)) {
		case "exit", "die":
			if p := f.prevEffective(i - 1); !f.is(p, "->") && !f.is(p, "?->") && !f.is(p, "::") && !f.isKeyword(p, "function") {
				return "exit"
			}
		}
		return "string"
	case token.Keyword:
		kw := strings.ToLower(t.orig)
		switch kw {
		case "default":
			x := i + 1
			for x < len(f.toks) && (f.isEmpty(x) || f.is(x, ",")) {
				x++
			}
			if f.is(x, "=>") {
				return "match_default"
			}
		case "case":
			if open := f.enclosingScope(i); open >= 0 && f.isKeyword(f.braceOwner(open), "enum") {
				return "enum_case"
			}
		}
		return kw
	}
	return "other"
}

// inlineElses retags the ":" of a ternary as PHP_CodeSniffer's T_INLINE_ELSE.
func (sc *phpcsScope) inlineElses() {
	f := sc.f
	var inlineIf []int
	for i := range f.toks {
		switch {
		case f.is(i, "?") && sc.isInlineThen(i):
			inlineIf = append(inlineIf, i)
		case f.is(i, ":") && len(inlineIf) > 0 && sc.isInlineElse(i, inlineIf[len(inlineIf)-1]):
			inlineIf = inlineIf[:len(inlineIf)-1]
			sc.code[i] = "inline_else"
		}
	}
}

// phpcsMagicConstants are lexed as names by blink but have their own token in
// PHP (T_DIR, T_FILE, ...), so they are no type name after a "?".
var phpcsMagicConstants = phpcsSet("__dir__", "__file__", "__line__", "__class__", "__function__",
	"__method__", "__namespace__", "__trait__", "__property__")

func (sc *phpcsScope) isInlineThen(i int) bool {
	f := sc.f
	named := false
	for j := i + 1; j < len(f.toks); j++ {
		if f.isEmpty(j) {
			continue
		}
		t := f.toks[j]
		if t.kind == token.Ident && !phpcsMagicConstants[strings.ToLower(t.orig)] || f.isKeyword(j, "array") || f.isKeyword(j, "namespace") || f.is(j, `\`) {
			named = true
			continue
		}
		if !named {
			return !f.isKeyword(j, "callable")
		}
		return f.is(j, "::") || f.is(j, "(") || f.is(j, ":")
	}
	return false
}

func (sc *phpcsScope) isInlineElse(i, inlineThen int) bool {
	f := sc.f
	p := f.prevEffective(i - 1)
	if p >= 0 && f.toks[p].kind == token.Ident {
		if pp := f.prevEffective(p - 1); f.is(pp, "(") || f.is(pp, ",") {
			return false
		}
	}
	if f.is(p, ")") && f.toks[p].closer >= 0 {
		j := f.toks[p].closer - 1
		for j > 0 && (f.isEmpty(j) || f.toks[j].kind == token.Ident) {
			j--
		}
		if f.isKeyword(j, "function") || f.isKeyword(j, "fn") || f.isKeyword(j, "use") {
			return false
		}
	}
	for j := i; j > inlineThen; j-- {
		if f.isKeyword(j, "case") || f.isKeyword(j, "default") {
			return false
		}
		if f.is(j, ";") || f.is(j, "{") || f.is(j, "}") {
			break
		}
	}
	return true
}

// parentheses sets the parenthesis owners and nesting of createTokenMap and
// createParenthesisNestingMap.
func (sc *phpcsScope) parentheses() {
	f := sc.f
	var stack []int
	for i, t := range f.toks {
		sc.nested[i] = -1
		if len(stack) > 0 {
			sc.nested[i] = stack[len(stack)-1]
		}
		if f.is(i, "(") && t.closer >= 0 {
			stack = append(stack, i)
		} else if f.is(i, ")") && len(stack) > 0 && stack[len(stack)-1] == t.closer {
			stack = stack[:len(stack)-1]
			sc.nested[i] = -1
			if len(stack) > 0 {
				sc.nested[i] = stack[len(stack)-1]
			}
		}
		if !phpcsParenOpeners[sc.code[i]] {
			continue
		}
		j := i + 1
		for j < len(f.toks) && (f.isEmpty(j) || sc.code[i] == "function" && (sc.code[j] == "string" || f.is(j, "&"))) {
			j++
		}
		if f.is(j, "(") && f.toks[j].closer >= 0 {
			sc.pCloser[i] = f.toks[j].closer
			sc.pOwner[j] = i
		}
	}
}

func (sc *phpcsScope) set(tok, cond, opener, closer int) {
	sc.cond[tok], sc.opener[tok], sc.closer[tok] = cond, opener, closer
}

// recurse is Tokenizer::recurseScopeMap.
func (sc *phpcsScope) recurse(stackPtr, depth int, ignore *int) int {
	f := sc.f
	opener := -1
	currType := sc.code[stackPtr]
	curr := phpcsScopeOpeners[currType]
	startLine := f.toks[stackPtr].line
	originalIgnore := *ignore
	if curr.start[currType] {
		opener = stackPtr
	}
	for i := stackPtr + 1; i < len(f.toks); i++ {
		tokenType := sc.code[i]
		if (currType == "if" || currType == "else" || currType == "use") && opener < 0 &&
			(tokenType == ";" || tokenType == "close_tag") {
			return i
		}
		if opener < 0 && *ignore == 0 && tokenType == "}" && curr.end[tokenType] {
			return i - 1
		}
		if opener >= 0 && (sc.opener[i] < 0 || curr.shared) && curr.end[tokenType] {
			if *ignore > 0 && tokenType == "}" {
				*ignore--
				continue
			}
			if sc.code[opener] == "{" && tokenType != "}" {
				// a "{" opener only takes a "}" closer
			} else {
				scopeCloser := i
				todo := []int{stackPtr, opener}
				validCloser := true
				if (currType == "if" || currType == "elseif") && (tokenType == "else" || tokenType == "elseif") {
					i = sc.recurse(i, depth+1, ignore)
					if sc.opener[scopeCloser] < 0 || sc.code[sc.opener[scopeCloser]] != sc.code[opener] {
						validCloser = false
					}
				} else {
					todo = append(todo, scopeCloser)
				}
				if validCloser {
					for _, t := range todo {
						sc.set(t, stackPtr, opener, scopeCloser)
					}
					if curr.shared {
						*ignore = originalIgnore
						return opener
					}
					if _, ok := phpcsScopeOpeners[tokenType]; scopeCloser == i && ok {
						sc.cond[scopeCloser] = -1
						return i - 1
					}
					return i
				}
				continue
			}
		}
		if next, ok := phpcsScopeOpeners[tokenType]; ok {
			if opener < 0 {
				if tokenType == "use" {
					continue
				}
				if tokenType == "function" && currType != "function" || tokenType == "class" {
					if sc.closer[i] >= 0 {
						i = sc.closer[i]
						continue
					}
					i = sc.recurse(i, depth+1, ignore)
					continue
				}
				if (currType == "if" || currType == "elseif" || currType == "else") && (tokenType == "else" || tokenType == "elseif") {
					return i - 1
				}
				return stackPtr
			}
			if sc.cond[i] >= 0 {
				if !next.shared && sc.closer[i] >= 0 {
					i = sc.closer[i]
				}
				continue
			} else if currType == tokenType && !next.shared && opener < 0 {
				return stackPtr
			} else {
				oldIgnore := *ignore
				if next.end["}"] {
					*ignore = 0
				}
				oldDepth := depth
				if next.shared && next.with[currType] {
					depth--
				}
				if depth > 50 {
					return len(f.toks)
				}
				i = sc.recurse(i, depth+1, ignore)
				depth = oldDepth
				if next.end["}"] {
					*ignore = oldIgnore
				}
			}
		}
		if curr.start[tokenType] && opener < 0 {
			if tokenType == "{" {
				if pc := sc.pCloser[stackPtr]; pc >= 0 && i < pc {
					*ignore++
				} else {
					x := i - 1
					for x > 0 && f.isEmpty(x) {
						x--
					}
					if phpcsAssignDisallowed[sc.code[x]] {
						*ignore++
					}
				}
			}
			if *ignore == 0 || tokenType != "{" {
				if sc.nested[i] == sc.nested[stackPtr] {
					opener = i
				}
			}
		} else if tokenType == ";" && opener < 0 && (sc.pCloser[stackPtr] < 0 || i > sc.pCloser[stackPtr]) {
			return i - 1
		} else if tokenType == "(" {
			if owner := sc.pOwner[i]; owner >= 0 && phpcsScopeOpenerCodes[sc.code[owner]] {
				startLine = f.toks[f.toks[i].closer].line
			}
		} else if tokenType == "{" && opener >= 0 {
			*ignore++
		} else if tokenType == "}" && *ignore > 0 {
			*ignore--
		} else if opener < 0 {
			if f.toks[i].line >= startLine+30 && sc.code[i-1] != "whitespace" && sc.code[i-1] != "comment" && !curr.strict {
				return stackPtr
			}
		} else if tokenType != "break" && phpcsEndScopeTokens[tokenType] && sc.cond[i] < 0 {
			if *ignore > 0 {
				*ignore--
			} else {
				for _, t := range []int{stackPtr, opener} {
					sc.set(t, stackPtr, opener, i)
				}
				return i - 1
			}
		}
	}
	return stackPtr
}

// fixCases is PHP::processAdditional's backward pass: it fills the scope
// conditions alternate syntax left unset and gives a CASE or DEFAULT whose
// body is a braced block that block as its scope.
func (sc *phpcsScope) fixCases() {
	f := sc.f
	for i := len(f.toks) - 1; i >= 0; i-- {
		if sc.opener[i] >= 0 && sc.cond[i] < 0 {
			sc.cond[i] = sc.cond[sc.opener[i]]
		}
		if sc.code[i] != "case" && sc.code[i] != "default" || sc.opener[i] < 0 {
			continue
		}
		scopeOpener, scopeCloser := sc.opener[i], sc.closer[i]
		x := scopeOpener + 1
		for x < len(f.toks) && f.isEmpty(x) {
			x++
		}
		if x >= len(f.toks) {
			continue
		}
		if sc.code[x] == "case" || sc.code[x] == "default" {
			sc.closer[i] = sc.closer[x]
			continue
		}
		if sc.code[x] != "{" || sc.cond[x] >= 0 || f.toks[x].closer < 0 {
			continue
		}
		newCloser := f.toks[x].closer
		sc.caseFixes = append(sc.caseFixes, [3]int{i, newCloser, scopeCloser})
		for _, idx := range []int{i, x, newCloser} {
			sc.set(idx, i, x, newCloser)
		}
		if sc.cond[scopeOpener] == i {
			sc.set(scopeOpener, -1, -1, -1)
		}
		if sc.cond[scopeCloser] == i {
			sc.set(scopeCloser, -1, -1, -1)
		} else if condition := sc.cond[scopeCloser]; condition >= 0 {
			for y := sc.opener[condition] + 1; y < scopeCloser; y++ {
				if sc.closer[y] == scopeCloser {
					sc.closer[y] = newCloser
				}
			}
		}
	}
}

func phpcsFirstEnd(code string) string {
	switch code {
	case "case", "default":
		return "break"
	case "start_heredoc":
		return "end_heredoc"
	}
	return "}"
}

// phpcsCond is one entry of a token's conditions: the scope condition's
// token, or, with caseIndex, the entry PHP::processAdditional appends to the
// old closer of a CASE it gives a braced body, which holds the CASE's index
// instead of its code.
type phpcsCond struct {
	key       int
	caseIndex bool
}

// levelMap is Tokenizer::createLevelMap plus the condition fixes of
// PHP::processAdditional: each token's conditions, outermost first, and level,
// which a shared closer can leave apart from len(conditions).
func (sc *phpcsScope) levelMap() (conds [][]phpcsCond, levels []int) {
	n := len(sc.code)
	conds, levels = make([][]phpcsCond, n), make([]int, n)
	nextFree := make([]int, n)
	remove := func(list []phpcsCond, key int) []phpcsCond {
		out := list[:0:0]
		for _, c := range list {
			if c.key != key || c.caseIndex {
				out = append(out, c)
			}
		}
		return out
	}
	level, free := 0, 0
	var conditions []phpcsCond
	var openers []int
	lastOpener := -1
	for i := range n {
		levels[i], conds[i], nextFree[i] = level, conditions, free
		if sc.preCond[i] < 0 {
			continue
		}
		if sc.preOpener[i] == i {
			stackPtr := sc.preCond[i]
			if lastOpener >= 0 && sc.preCloser[lastOpener] == sc.preCloser[i] {
				thisType := sc.code[sc.preCond[i]]
				opener := sc.preCond[lastOpener]
				isShared := phpcsScopeOpeners[thisType].with[sc.code[opener]]
				if isShared && phpcsFirstEnd(thisType) == phpcsFirstEnd(sc.code[opener]) {
					for x := sc.preCond[i]; x <= i; x++ {
						levels[x]--
						conds[x] = remove(conds[x], opener)
					}
					conditions = remove(conditions, opener)
					if k := slices.Index(openers, lastOpener); k >= 0 {
						openers = slices.Delete(openers, k, k+1)
					}
					level--
				}
			}
			level++
			if !slices.Contains(conditions, phpcsCond{key: stackPtr}) {
				conditions = append(slices.Clone(conditions), phpcsCond{key: stackPtr})
				free = max(free, stackPtr+1)
			}
			lastOpener = sc.preOpener[i]
			if !slices.Contains(openers, lastOpener) {
				openers = append(openers, lastOpener)
			}
			continue
		}
		if lastOpener < 0 || sc.preCloser[lastOpener] != i {
			continue
		}
		reversed := slices.Clone(openers)
		slices.Reverse(reversed)
		for _, opener := range reversed {
			if sc.preCloser[opener] != i {
				continue
			}
			oldOpener := openers[len(openers)-1]
			openers = openers[:len(openers)-1]
			lastOpener = -1
			if len(openers) > 0 {
				lastOpener = openers[len(openers)-1]
			}
			oldCondition := ""
			if len(conditions) > 0 {
				last := conditions[len(conditions)-1]
				oldCondition = sc.code[last.key]
				if last.key == free-1 {
					free--
				}
				conditions = slices.Clone(conditions[:len(conditions)-1])
			}
			condition := sc.code[sc.preCond[i]]
			if condition != oldCondition && !phpcsScopeOpeners[oldCondition].with[condition] {
				badToken := sc.preCond[oldOpener]
				for x := oldOpener + 1; x <= i; x++ {
					levels[x]--
					conds[x] = remove(conds[x], badToken)
				}
			}
			level--
			levels[i], conds[i], nextFree[i] = level, conditions, free
		}
	}
	for _, fix := range sc.caseFixes {
		i, newCloser, scopeCloser := fix[0], fix[1], fix[2]
		conds[scopeCloser] = append(slices.Clone(conds[scopeCloser]), phpcsCond{key: nextFree[scopeCloser], caseIndex: true})
		nextFree[scopeCloser]++
		for x := newCloser; x < scopeCloser; x++ {
			out := conds[x][:0:0]
			for _, c := range conds[x] {
				if c.caseIndex || sc.code[c.key] != sc.code[i] {
					out = append(out, c)
				}
			}
			conds[x] = out
		}
	}
	return conds, levels
}
