package rules

import (
	"slices"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// Symplify: https://github.com/ecsphp/ecs-src/blob/main/packages/coding-standard/src/Fixer/Spacing/MethodChainingNewlineFixer.php
//
// MethodChainingNewline puts each "->" of a call chain on its own line, by
// inserting a bare "\n" before it; MethodChainingIndentation sets the indent.
// It skips chains that already span lines, chains in conditions and
// comparisons, a short method without arguments, calls inside arguments or
// arrays, and a single call on a function result.
type MethodChainingNewline struct{}

func (MethodChainingNewline) Name() string {
	return `Symplify\CodingStandard\Fixer\Spacing\MethodChainingNewlineFixer`
}

func (MethodChainingNewline) SourceURL() string {
	return "https://github.com/ecsphp/ecs-src/blob/main/packages/coding-standard/src/Fixer/Spacing/MethodChainingNewlineFixer.php"
}

// chainTok is one token as PHP-CS-Fixer sees it. The Symplify checks compare
// token contents, so a cast must be one opaque token and the "{" and "}" of a
// "{$...}" interpolation must be tokens of their own, as in PHP.
type chainTok struct {
	val  string
	kind token.Kind
	at   int  // stream index of the "->" (only read for those)
	ct   bool // a ")" PHP-CS-Fixer retypes, which equals(')') no longer matches
}

func (MethodChainingNewline) Fix(s *tokens.Stream) bool {
	if !slices.ContainsFunc(s.Tokens(), func(t token.Token) bool { return t.Kind == token.Punct && t.Value == "->" }) {
		return false
	}
	v := chainTokens(s)
	changed := false
	inserted := 0
	// PHP caches count($tokens) before the loop, so insertions push the last
	// tokens out of reach
	for k, count := 1, len(v); k < count; k++ {
		if v[k].kind != token.Punct || v[k].val != "->" || !chainShouldPrefixNewline(v, k) {
			continue
		}
		at := v[k].at + inserted
		if prev := s.At(at - 1); prev.Kind == token.Whitespace {
			s.SetValue(at-1, prev.Value+"\n")
		} else {
			s.InsertAt(at, token.Token{Kind: token.Whitespace, Value: "\n"})
			inserted++
		}
		v = slices.Insert(v, k, chainTok{val: "\n", kind: token.Whitespace})
		changed = true
		k++
	}
	return changed
}

func chainTokens(s *tokens.Stream) []chainTok {
	v := make([]chainTok, 0, s.Len())
	var curly []bool // per open interpolated string: does its last part end in "{$"?
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if closeIdx, _, ok := aliasCastAt(s, i); ok {
			var b strings.Builder
			for j := i; j <= closeIdx; j++ {
				b.WriteString(s.At(j).Value)
			}
			v = append(v, chainTok{val: b.String(), kind: token.Unknown})
			i = closeIdx
			continue
		}
		if t.Kind != token.String || t.Part == token.Whole {
			v = append(v, chainTok{val: t.Value, kind: t.Kind, at: i, ct: t.Kind == token.Punct && t.Value == ")" && isClassInstantiationCloseParen(s, i)})
			continue
		}
		val := t.Value
		if t.Part != token.Head && len(curly) > 0 {
			if curly[len(curly)-1] && strings.HasPrefix(val, "}") {
				v = append(v, chainTok{val: "}", kind: token.Punct})
				val = val[1:]
			}
			curly = curly[:len(curly)-1]
		}
		opens := t.Part != token.Tail && strings.HasSuffix(val, "{") && i+1 < s.Len() && strings.HasPrefix(s.At(i+1).Value, "$")
		if opens {
			val = val[:len(val)-1]
		}
		if val != "" {
			v = append(v, chainTok{val: val, kind: token.String})
		}
		if opens {
			v = append(v, chainTok{val: "{", kind: token.Punct})
		}
		if t.Part != token.Tail {
			curly = append(curly, opens)
		}
	}
	return v
}

func chainShouldPrefixNewline(v []chainTok, op int) bool {
	for i := op; i >= 0; i-- {
		switch {
		case v[i].kind == token.Punct && v[i].val == ")" && !v[i].ct:
			return chainShouldBracketPrefix(v, i, op)
		case v[i].kind == token.Variable, v[i].kind == token.Keyword && strings.EqualFold(v[i].val, "new"):
			return false
		case v[i].val == "(":
			return false
		}
	}
	return false
}

func chainShouldBracketPrefix(v []chainTok, closeIdx, op int) bool {
	if chainHasOperator(v, op, -1) || chainHasOperator(v, op, 1) ||
		chainInControlCondition(v, op) ||
		chainShortNoArgMethod(v, op) ||
		chainPartOfCallOrArray(v, closeIdx) ||
		chainPrecededByFuncCall(v, closeIdx) {
		return false
	}
	if open := chainBlockStart(v, closeIdx); open >= 0 && chainMultiline(v, open, closeIdx) {
		return false
	}
	return !chainMultiline(v, closeIdx, op)
}

func chainHasOperator(v []chainTok, op, step int) bool {
	nesting := 0
	for i := op + step; i >= 0 && i < len(v); i += step {
		c := v[i].val
		opens, closes := c == "(" || c == "[", c == ")" || c == "]"
		if step < 0 {
			opens, closes = closes, opens
		}
		switch {
		case opens:
			nesting++
		case closes:
			if nesting == 0 {
				return false
			}
			nesting--
		case nesting != 0:
		case c == ";" || c == "{" || c == "}":
			return false
		case chainIsComparison(v[i]):
			return true
		}
	}
	return false
}

func chainIsComparison(t chainTok) bool {
	if t.val == "<" || t.val == ">" {
		return true
	}
	switch t.kind {
	case token.Punct:
		switch t.val {
		case "&&", "||", "==", "!=", "<>", "===", "!==", "<=", ">=", "<=>":
			return true
		}
	case token.Keyword:
		switch strings.ToLower(t.val) {
		case "and", "or", "xor":
			return true
		}
	}
	return false
}

func chainInControlCondition(v []chainTok, op int) bool {
	nesting := 0
	for i := op; i >= 0; i-- {
		switch c := v[i].val; {
		case c == ")" || c == "]":
			nesting++
		case c == "(" || c == "[":
			if nesting != 0 {
				nesting--
				continue
			}
			if c == "[" {
				return false
			}
			p := chainPrevMeaningful(v, i)
			if p < 0 || v[p].kind != token.Keyword {
				return false
			}
			switch strings.ToLower(v[p].val) {
			case "if", "elseif", "while", "switch":
				return true
			}
			return false
		case nesting == 0 && (c == ";" || c == "{" || c == "}"):
			return false
		}
	}
	return false
}

func chainShortNoArgMethod(v []chainTok, op int) bool {
	name := chainNextMeaningful(v, op)
	if name < 0 || v[name].kind != token.Ident || len(v[name].val) > 5 {
		return false
	}
	open := chainNextMeaningful(v, name)
	if open < 0 || v[open].val != "(" {
		return false
	}
	closeIdx := chainNextMeaningful(v, open)
	return closeIdx >= 0 && v[closeIdx].val == ")"
}

// chainPartOfCallOrArray counts the ")" it starts on, so the "(" that matches
// it does not stop the scan.
func chainPartOfCallOrArray(v []chainTok, closeIdx int) bool {
	nesting := 0
	for i := closeIdx; i >= 0; i-- {
		t := v[i]
		if chainIsNewline(t) {
			return false
		}
		switch {
		case t.val == "[" || t.val == "." || t.kind == token.Punct && t.val == "::":
			return true
		case t.kind == token.Keyword && strings.EqualFold(t.val, "array"):
			// PHP-CS-Fixer retypes an "array" type hint; only "array(" stays T_ARRAY
			if n := chainNextMeaningful(v, i); n >= 0 && v[n].val == "(" {
				return true
			}
		case t.val == ")":
			nesting--
		case t.val == "(":
			if nesting == 0 {
				return true
			}
			nesting++
		}
	}
	return false
}

func chainPrecededByFuncCall(v []chainTok, closeIdx int) bool {
	for i := closeIdx; i >= 0; i-- {
		switch {
		case v[i].val == "clone":
			return true
		case v[i].val == "(":
			return chainCallNeedsNewline(v, i)
		case chainIsNewline(v[i]):
			return false
		}
	}
	return false
}

// chainCallNeedsNewline mirrors NewlineAnalyzer: the name before "(" follows
// "{", "return" or "::". It skips whitespace only, not comments.
func chainCallNeedsNewline(v []chainTok, open int) bool {
	name := chainPrevNonWhitespace(v, open)
	if name < 0 || !chainIsTString(v, name) {
		return false
	}
	p := chainPrevNonWhitespace(v, name)
	if p < 0 {
		return false
	}
	return v[p].val == "{" ||
		v[p].kind == token.Keyword && strings.EqualFold(v[p].val, "return") ||
		v[p].kind == token.Punct && v[p].val == "::"
}

// chainIsTString reports whether PHP lexes the name at i as T_STRING; blink
// lexes "exit", "die" and "eval" as names, PHP as constructs.
func chainIsTString(v []chainTok, i int) bool {
	if v[i].kind != token.Ident {
		return false
	}
	switch strings.ToLower(v[i].val) {
	case "exit", "die", "eval":
		p := chainPrevMeaningful(v, i)
		return p >= 0 && (v[p].val == "->" || v[p].val == "?->" || v[p].val == "::")
	}
	return true
}

func chainBlockStart(v []chainTok, closeIdx int) int {
	depth := 0
	for i := closeIdx; i >= 0; i-- {
		if v[i].kind != token.Punct {
			continue
		}
		switch v[i].val {
		case ")":
			depth++
		case "(":
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func chainMultiline(v []chainTok, from, to int) bool {
	for i := from; i <= to; i++ {
		if strings.Contains(v[i].val, "\n") {
			return true
		}
	}
	return false
}

func chainIsNewline(t chainTok) bool {
	return t.kind == token.Whitespace && strings.Contains(t.val, "\n")
}

func chainPrevNonWhitespace(v []chainTok, i int) int {
	for j := i - 1; j >= 0; j-- {
		if v[j].kind != token.Whitespace {
			return j
		}
	}
	return -1
}

func chainPrevMeaningful(v []chainTok, i int) int {
	for j := i - 1; j >= 0; j-- {
		switch v[j].kind {
		case token.Whitespace, token.Comment, token.DocComment:
			continue
		}
		return j
	}
	return -1
}

func chainNextMeaningful(v []chainTok, i int) int {
	for j := i + 1; j < len(v); j++ {
		switch v[j].kind {
		case token.Whitespace, token.Comment, token.DocComment:
			continue
		}
		return j
	}
	return -1
}
