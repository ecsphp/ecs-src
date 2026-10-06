package rules

import (
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// This file implements three PHP-CS-Fixer Operator rules on the flat token
// stream. Each is deliberately narrower than the upstream fixer: only the
// token-adjacency cases that are provably semantics-preserving are rewritten,
// everything else is left untouched. Correctness (never corrupt, no-op on
// already-correct code, idempotent) is preferred over completeness.

// sigPrev returns the index of the first meaningful token before i (skipping
// whitespace and comments), or -1.
func sigPrev(s *tokens.Stream, i int) int {
	for j := i - 1; j >= 0; j-- {
		switch s.At(j).Kind {
		case token.Whitespace, token.Comment, token.DocComment:
			continue
		}
		return j
	}
	return -1
}

// sigNext returns the index of the first meaningful token after i, or -1.
func sigNext(s *tokens.Stream, i int) int {
	for j := i + 1; j < s.Len(); j++ {
		switch s.At(j).Kind {
		case token.Whitespace, token.Comment, token.DocComment:
			continue
		}
		return j
	}
	return -1
}

// spanClean reports whether the inclusive range [a, b] holds no comments, so a
// rewrite of that span cannot silently drop one.
func spanClean(s *tokens.Stream, a, b int) bool {
	for j := a; j <= b && j < s.Len(); j++ {
		if k := s.At(j).Kind; k == token.Comment || k == token.DocComment {
			return false
		}
	}
	return true
}

// inlineWSBetween reports whether every token strictly between a and b is
// whitespace with no newline (so the two are on the same line, gap removable).
func inlineWSBetween(s *tokens.Stream, a, b int) bool {
	for j := a + 1; j < b; j++ {
		t := s.At(j)
		if t.Kind != token.Whitespace || hasNewline(t.Value) {
			return false
		}
	}
	return true
}

// lvaluePrefix reports whether v attaches the following variable to a larger
// lvalue (member/static/dynamic/reference access), meaning it is not a plain
// standalone variable.
func lvaluePrefix(v string) bool {
	switch v {
	case "->", "?->", "::", "$", "&":
		return true
	}
	return false
}

// lvChainForward walks a variable-rooted lvalue chain starting at `start`
// ("$x", "$x->prop", "$x[sub]", and nestings), returning the index of the
// chain's last token or -1 if `start` is not a variable or the chain is
// malformed. A "::" static access stops the chain (returned before it), so
// static-property lvalues fall out of the match and are left untouched.
func lvChainForward(s *tokens.Stream, start int) int {
	if start < 0 || s.At(start).Kind != token.Variable {
		return -1
	}
	k := start
	for {
		n := sigNext(s, k)
		if n < 0 {
			return k
		}
		nt := s.At(n)
		if nt.Kind == token.Punct && (nt.Value == "->" || nt.Value == "?->") {
			m := sigNext(s, n)
			if m < 0 {
				return -1
			}
			if mk := s.At(m).Kind; mk == token.Ident || mk == token.Variable {
				k = m
				continue
			}
			return -1
		}
		if nt.Kind == token.Punct && nt.Value == "[" {
			cl := s.MatchForward(n)
			if cl < 0 {
				return -1
			}
			k = cl
			continue
		}
		return k
	}
}

// opAssignBoundary reports whether t ends the expression to the left of an
// lvalue, so the next significant token begins a fresh lvalue.
func opAssignBoundary(t token.Token) bool {
	if t.Kind == token.OpenTag {
		return true
	}
	if t.Kind == token.Keyword && strings.EqualFold(t.Value, "return") {
		return true
	}
	if t.Kind == token.Punct {
		switch t.Value {
		case ";", "{", "}", "(", ")", "[", ",", ":":
			return true
		}
	}
	return false
}

// lvChainRootBack scans backward from `end` (the last token of an lvalue that
// sits just before an operator) to the chain's root variable, jumping matched
// "[...]" subscripts. It returns the root index, or -1 when the span is not a
// clean variable-rooted chain bounded by a statement boundary.
func lvChainRootBack(s *tokens.Stream, end int) int {
	k := end
	last := end
	for k >= 0 {
		t := s.At(k)
		if isTrivia(s, k) {
			k--
			continue
		}
		if opAssignBoundary(t) {
			break
		}
		if t.Kind == token.Punct && t.Value == "]" {
			op := s.MatchBackward(k)
			if op < 0 {
				return -1
			}
			last = op
			k = op - 1
			continue
		}
		if t.Kind == token.Punct && t.Value == ")" {
			return -1
		}
		last = k
		k--
	}
	if s.At(last).Kind == token.Variable {
		return last
	}
	return -1
}

// sameSigRange reports whether the significant tokens of [a1,a2] equal those of
// [b1,b2] by kind and value.
func sameSigRange(s *tokens.Stream, a1, a2, b1, b2 int) bool {
	ai, bi := a1, b1
	for {
		for ai <= a2 && isTrivia(s, ai) {
			ai++
		}
		for bi <= b2 && isTrivia(s, bi) {
			bi++
		}
		aDone, bDone := ai > a2, bi > b2
		if aDone || bDone {
			return aDone && bDone
		}
		if s.At(ai).Kind != s.At(bi).Kind || s.At(ai).Value != s.At(bi).Value {
			return false
		}
		ai++
		bi++
	}
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Operator/StandardizeIncrementFixer.php
//
// StandardizeIncrement rewrites `$i += 1` to `++$i` and `$i -= 1` to `--$i`.
// The pre-increment form is value-equivalent to the compound assignment, so the
// rewrite is safe in any expression position. A variable-rooted lvalue chain
// ("$this->index", "$arr[$k]") is handled; "::" static access is left to the
// upstream fixer.
type StandardizeIncrement struct{}

func (StandardizeIncrement) Name() string {
	return `PhpCsFixer\Fixer\Operator\StandardizeIncrementFixer`
}

func (StandardizeIncrement) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Operator/StandardizeIncrementFixer.php"
}

// incExprEnd matches the tokens that may terminate the `1` operand.
func incExprEnd(v string) bool {
	switch v {
	case ";", ")", "]", ",", ":":
		return true
	}
	return false
}

func (StandardizeIncrement) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind != token.Punct || (t.Value != "+=" && t.Value != "-=") {
			continue
		}

		// Left side: a variable-rooted lvalue chain ending just before "+=".
		lv := sigPrev(s, i)
		if lv < 0 {
			continue
		}
		start := lvChainRootBack(s, lv)
		if start < 0 || lvChainForward(s, start) != lv {
			continue
		}

		// Right side: the literal 1, then an expression-end token.
		numIdx := sigNext(s, i)
		if numIdx < 0 || s.At(numIdx).Kind != token.Number || s.At(numIdx).Value != "1" {
			continue
		}
		endIdx := sigNext(s, numIdx)
		if endIdx < 0 || s.At(endIdx).Kind != token.Punct || !incExprEnd(s.At(endIdx).Value) {
			continue
		}

		// Refuse if a comment sits inside the rewritten span.
		if !spanClean(s, start, numIdx) {
			continue
		}

		op := "++"
		if t.Value == "-=" {
			op = "--"
		}

		// Drop `<op> 1` (operator..number), then the inline gap before it, then
		// prepend the increment operator at the chain root.
		for k := numIdx; k >= i; k-- {
			s.RemoveAt(k)
		}
		if lv+1 < s.Len() && s.At(lv+1).Kind == token.Whitespace && !hasNewline(s.At(lv+1).Value) {
			s.RemoveAt(lv + 1)
		}
		s.InsertAt(start, token.Token{Kind: token.Punct, Value: op})
		changed = true
		i = start
	}
	return changed
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Operator/IncrementStyleFixer.php
//
// IncrementStyle converts a post-increment statement to pre-increment
// (`$i++;` -> `++$i;`, `$i--;` -> `--$i;`) using the default `pre` style. Only
// standalone statements on a single plain variable are rewritten, where the
// return value is discarded and post/pre are equivalent.
type IncrementStyle struct{}

func (IncrementStyle) Name() string {
	return `PhpCsFixer\Fixer\Operator\IncrementStyleFixer`
}

func (IncrementStyle) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Operator/IncrementStyleFixer.php"
}

func (IncrementStyle) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind != token.Punct || (t.Value != "++" && t.Value != "--") {
			continue
		}

		// Operand: exactly one plain variable, on the same line as the operator.
		lv := sigPrev(s, i)
		if lv < 0 || s.At(lv).Kind != token.Variable {
			continue
		}
		if !inlineWSBetween(s, lv, i) {
			continue
		}

		// The operand must open a statement (previous meaningful token is a
		// boundary) and not be part of a larger lvalue.
		p := sigPrev(s, lv)
		if p < 0 {
			continue
		}
		pv := s.At(p)
		if lvaluePrefix(pv.Value) {
			continue
		}
		if pv.Value != ";" && pv.Value != "{" && pv.Value != "}" && pv.Kind != token.OpenTag {
			continue
		}

		// The operator must be a standalone statement, terminated by ';'.
		endIdx := sigNext(s, i)
		if endIdx < 0 || s.At(endIdx).Value != ";" {
			continue
		}

		// Move the operator in front of the variable, dropping the inline gap.
		for k := i; k >= lv+1; k-- {
			s.RemoveAt(k)
		}
		s.InsertAt(lv, token.Token{Kind: token.Punct, Value: t.Value})
		changed = true
		i = lv
	}
	return changed
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Operator/LongToShorthandOperatorFixer.php
//
// LongToShorthandOperator rewrites `$a = $a <op> <operand>;` to
// `$a <op>= <operand>;`. The lvalue may be a variable-rooted chain
// ("$this->index", "$arr[$k]") as long as both sides are token-identical; only
// the single-operand, semicolon-terminated right side is handled, where the
// lone operand plus the ";" terminator removes any precedence ambiguity.
// Anything with a chained or complex right side is left to the upstream fixer.
type LongToShorthandOperator struct{}

func (LongToShorthandOperator) Name() string {
	return `PhpCsFixer\Fixer\Operator\LongToShorthandOperatorFixer`
}

func (LongToShorthandOperator) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Operator/LongToShorthandOperatorFixer.php"
}

// shortOp reports whether v is an operator with a compound-assignment form.
func shortOp(v string) bool {
	switch v {
	case "+", "-", "*", "/", ".", "%", "&", "|", "^":
		return true
	}
	return false
}

// shortOperand reports whether t is a single self-contained right operand.
func shortOperand(t token.Token) bool {
	switch t.Kind {
	case token.Variable, token.Number, token.String, token.Ident:
		return true
	}
	return false
}

func (LongToShorthandOperator) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind != token.Punct || t.Value != "=" {
			continue
		}

		// Left side: a variable-rooted lvalue chain ending just before "=".
		lv := sigPrev(s, i)
		if lv < 0 {
			continue
		}
		lhsStart := lvChainRootBack(s, lv)
		if lhsStart < 0 || lvChainForward(s, lhsStart) != lv {
			continue
		}

		// Right side must repeat the identical chain.
		rv := sigNext(s, i)
		if rv < 0 || s.At(rv).Kind != token.Variable {
			continue
		}
		rhsEnd := lvChainForward(s, rv)
		if rhsEnd < 0 || !sameSigRange(s, lhsStart, lv, rv, rhsEnd) {
			continue
		}

		// Then a compound-capable operator.
		opIdx := sigNext(s, rhsEnd)
		if opIdx < 0 || s.At(opIdx).Kind != token.Punct || !shortOp(s.At(opIdx).Value) {
			continue
		}

		// Then a single operand and a terminating ';'.
		operandIdx := sigNext(s, opIdx)
		if operandIdx < 0 || !shortOperand(s.At(operandIdx)) {
			continue
		}
		endIdx := sigNext(s, operandIdx)
		if endIdx < 0 || s.At(endIdx).Value != ";" {
			continue
		}

		// Refuse if a comment sits inside the rewritten span.
		if !spanClean(s, i, opIdx) {
			continue
		}

		// `= $a <op>` becomes `<op>=`: retag the `=` and drop the duplicate
		// operand and operator.
		s.SetValue(i, s.At(opIdx).Value+"=")
		for k := opIdx; k >= i+1; k-- {
			s.RemoveAt(k)
		}
		changed = true
	}
	return changed
}
