package rules

import (
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// Symplify: https://github.com/symplify/coding-standard/blob/main/src/Fixer/Spacing/MethodChainingNewlineFixer.php
//
// MethodChainingNewline puts each chained method call on its own line. A "->"
// that directly follows a method call's ")" starts a new line, indented one
// level past the chain. The first call after the chain root (a variable or a
// grouped/constructor expression) stays inline; chains that are call/array
// arguments on the same line, that follow a "::"/"["/"." on the line, that
// follow a multi-line call, or that are already split are left untouched - so it
// is a no-op on code ECS has already formatted.
type MethodChainingNewline struct{}

func (MethodChainingNewline) Name() string {
	return `Symplify\CodingStandard\Fixer\Spacing\MethodChainingNewlineFixer`
}

func (MethodChainingNewline) SourceURL() string {
	return "https://github.com/symplify/coding-standard/blob/main/src/Fixer/Spacing/MethodChainingNewlineFixer.php"
}

func (MethodChainingNewline) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 1; i < s.Len(); i++ {
		if s.At(i).Kind != token.Punct || s.At(i).Value != "->" {
			continue
		}
		prev := sigPrev(s, i)
		if prev < 0 || s.At(prev).Kind != token.Punct || s.At(prev).Value != ")" {
			continue // "->" must follow a ")"
		}
		// the ")" must close a method call, not a grouping/constructor paren -
		// this keeps the first call after "(new X)" or "(expr)" inline
		if !chainIsCallClose(s, prev) {
			continue
		}
		// already split here, or the preceding call spans multiple lines: leave it
		if rangeHasNewline(s, prev, i) {
			continue
		}
		if open := s.MatchBackward(prev); open >= 0 && rangeHasNewline(s, open, prev) {
			continue
		}
		// "::", "[", "." or an enclosing "(" earlier on the line means the chain is
		// part of a call/array (symplify's isPartOfMethodCallOrArray) - leave it
		if chainLineHasBreakingChar(s, prev) {
			continue
		}
		// a chain used in a boolean/comparison expression stays inline
		if chainIsPartOfBooleanOrComparison(s, i) {
			continue
		}
		// a chain inside an if/elseif/while/switch condition stays inline
		if chainIsInsideControlCondition(s, i) {
			continue
		}
		// a short (<=5 chars) no-argument trailing call stays inline ("->yes()")
		if chainIsShortNoArgTrailingMethod(s, i) {
			continue
		}
		// a chain rooted in a function call that already requires a newline stays inline
		if chainIsPrecededByFuncCall(s, prev) {
			continue
		}
		nl := "\n" + chainFirstLineIndent(s, i) + "    "
		if s.At(i-1).Kind == token.Whitespace {
			s.SetValue(i-1, nl)
		} else {
			s.InsertAt(i, token.Token{Kind: token.Whitespace, Value: nl})
			i++
		}
		changed = true
	}
	return changed
}

// chainIsShortNoArgTrailingMethod reports whether the method called via the "->"
// at opIdx has a name of at most five characters and no arguments ("->yes()").
// Symplify keeps such short predicate-like accessors inline.
func chainIsShortNoArgTrailingMethod(s *tokens.Stream, opIdx int) bool {
	name := nextSignificantIndex(s, opIdx)
	if name < 0 || s.At(name).Kind != token.Ident || len(s.At(name).Value) > 5 {
		return false
	}
	open := nextSignificantIndex(s, name)
	if open < 0 || s.At(open).Value != "(" {
		return false
	}
	close := nextSignificantIndex(s, open)
	return close >= 0 && s.At(close).Value == ")"
}

// chainIsPartOfBooleanOrComparison reports whether the chain at opIdx is an
// operand of a boolean or comparison operator on the same statement level.
func chainIsPartOfBooleanOrComparison(s *tokens.Stream, opIdx int) bool {
	return chainHasOperatorInDirection(s, opIdx, -1) || chainHasOperatorInDirection(s, opIdx, 1)
}

func chainHasOperatorInDirection(s *tokens.Stream, pos, step int) bool {
	nesting := 0
	for i := pos + step; i >= 0 && i < s.Len(); i += step {
		c := s.At(i).Value
		if (step < 0 && (c == ")" || c == "]")) || (step > 0 && (c == "(" || c == "[")) {
			nesting++
			continue
		}
		if (step < 0 && (c == "(" || c == "[")) || (step > 0 && (c == ")" || c == "]")) {
			if nesting == 0 {
				return false
			}
			nesting--
			continue
		}
		if nesting != 0 {
			continue
		}
		if c == ";" || c == "{" || c == "}" {
			return false
		}
		if isBooleanOrComparisonToken(s.At(i)) {
			return true
		}
	}
	return false
}

// isBooleanOrComparisonToken reports whether t is a boolean or comparison
// operator. The word operators and/or/xor count only as keywords, never as a
// method name like "->and(" or "->or(".
func isBooleanOrComparisonToken(t token.Token) bool {
	if t.Kind == token.Punct {
		switch t.Value {
		case "&&", "||", "==", "!=", "===", "!==", "<=", ">=", "<=>", "<", ">":
			return true
		}
		return false
	}
	if t.Kind == token.Keyword {
		switch strings.ToLower(t.Value) {
		case "and", "or", "xor":
			return true
		}
	}
	return false
}

// chainIsInsideControlCondition reports whether the chain at opIdx sits inside an
// if/elseif/while/switch condition.
func chainIsInsideControlCondition(s *tokens.Stream, opIdx int) bool {
	nesting := 0
	for i := opIdx; i >= 0; i-- {
		c := s.At(i).Value
		if c == ")" || c == "]" {
			nesting++
			continue
		}
		if c == "(" || c == "[" {
			if nesting != 0 {
				nesting--
				continue
			}
			if c == "[" {
				return false
			}
			b := prevSignificantIndex(s, i)
			if b < 0 || s.At(b).Kind != token.Keyword {
				return false
			}
			switch strings.ToLower(s.At(b).Value) {
			case "if", "elseif", "while", "switch":
				return true
			}
			return false
		}
		if nesting == 0 && (c == ";" || c == "{" || c == "}") {
			return false
		}
	}
	return false
}

// chainIsPrecededByFuncCall reports whether the ")" at closeIdx belongs to a
// function call that itself requires a newline (return app(), Foo::app(), clone),
// which keeps the following "->" inline.
func chainIsPrecededByFuncCall(s *tokens.Stream, closeIdx int) bool {
	for i := closeIdx; i >= 0; i-- {
		t := s.At(i)
		if t.Kind == token.Keyword && strings.EqualFold(t.Value, "clone") {
			return true
		}
		if t.Kind == token.Punct && t.Value == "(" {
			return chainContentBeforeBracketRequiresNewline(s, i)
		}
		if t.Kind == token.Whitespace && hasNewline(t.Value) {
			return false
		}
	}
	return false
}

// chainContentBeforeBracketRequiresNewline mirrors symplify's NewlineAnalyzer: a
// "(" preceded by "name" that itself follows "{", "return" or "::".
func chainContentBeforeBracketRequiresNewline(s *tokens.Stream, parenIdx int) bool {
	p := prevSignificantIndex(s, parenIdx)
	if p < 0 || s.At(p).Kind != token.Ident {
		return false
	}
	pp := prevSignificantIndex(s, p)
	if pp < 0 {
		return false
	}
	t := s.At(pp)
	if t.Value == "{" || t.Value == "::" {
		return true
	}
	return t.Kind == token.Keyword && strings.EqualFold(t.Value, "return")
}

// chainIsCallClose reports whether the ")" at closeIdx closes a function/method
// call (its "(" follows a name, variable or another call/subscript) rather than
// a grouping or constructor paren.
func chainIsCallClose(s *tokens.Stream, closeIdx int) bool {
	open := s.MatchBackward(closeIdx)
	if open < 0 {
		return false
	}
	p := sigPrev(s, open)
	if p < 0 {
		return false
	}
	t := s.At(p)
	if t.Kind == token.Ident || t.Kind == token.Variable {
		return true
	}
	return t.Kind == token.Punct && (t.Value == ")" || t.Value == "]")
}

// chainFirstLineIndent returns the indentation of the line the chain starts on,
// so every continuation aligns to the same column. It walks back over the whole
// chain expression (jumping across balanced brackets) to its root.
func chainFirstLineIndent(s *tokens.Stream, opIdx int) string {
	i := opIdx
	root := opIdx
	for i > 0 {
		t := s.At(i - 1)
		switch {
		case t.Kind == token.Whitespace || t.Kind == token.Comment || t.Kind == token.DocComment:
			i--
		case t.Kind == token.Ident || t.Kind == token.Variable:
			root = i - 1
			i--
		case t.Kind == token.Punct && (t.Value == "->" || t.Value == "?->" || t.Value == "::"):
			i--
		case t.Kind == token.Keyword && strings.ToLower(t.Value) == "new":
			root = i - 1
			i--
		case t.Kind == token.Punct && (t.Value == ")" || t.Value == "]" || t.Value == "}"):
			m := s.MatchBackward(i - 1)
			if m < 0 {
				return chainBaseIndent(s, root)
			}
			root = m
			i = m
		default:
			return chainBaseIndent(s, root) // boundary before the chain root
		}
	}
	return chainBaseIndent(s, root)
}

// chainBaseIndent returns the indentation after the nearest newline before index.
func chainBaseIndent(s *tokens.Stream, index int) string {
	for i := index - 1; i >= 0; i-- {
		if t := s.At(i); t.Kind == token.Whitespace && strings.Contains(t.Value, "\n") {
			if nl := strings.LastIndexByte(t.Value, '\n'); nl >= 0 {
				return t.Value[nl+1:]
			}
		}
	}
	return ""
}

// chainLineHasBreakingChar mirrors symplify's isPartOfMethodCallOrArray: walking
// back from the ")" at pos to the start of the line, a "[", "::", "." or "array"
// (or an unmatched enclosing "(") means the chain is treated as part of a call or
// array and is left inline.
func chainLineHasBreakingChar(s *tokens.Stream, pos int) bool {
	nesting := 0
	for i := pos; i >= 0; i-- {
		t := s.At(i)
		if t.Kind == token.Whitespace && strings.Contains(t.Value, "\n") {
			return false
		}
		if t.Kind == token.Punct && (t.Value == "[" || t.Value == "::" || t.Value == ".") {
			return true
		}
		if t.Kind == token.Keyword && strings.ToLower(t.Value) == "array" {
			return true
		}
		if t.Kind == token.Punct && t.Value == ")" {
			nesting--
		} else if t.Kind == token.Punct && t.Value == "(" {
			if nesting != 0 {
				nesting++
			} else {
				return true
			}
		}
	}
	return false
}
