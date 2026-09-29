package rules

import (
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

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Operator/StandardizeIncrementFixer.php
//
// StandardizeIncrement rewrites `$i += 1` to `++$i` and `$i -= 1` to `--$i`.
// The pre-increment form is value-equivalent to the compound assignment, so the
// rewrite is safe in any expression position. Only a single plain variable on
// the left is handled; complex lvalues are left to the upstream fixer.
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

		// Left side: exactly one plain variable.
		lv := sigPrev(s, i)
		if lv < 0 || s.At(lv).Kind != token.Variable {
			continue
		}
		if p := sigPrev(s, lv); p >= 0 && lvaluePrefix(s.At(p).Value) {
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
		if !spanClean(s, lv, numIdx) {
			continue
		}

		op := "++"
		if t.Value == "-=" {
			op = "--"
		}

		// Drop `<op> 1` (operator..number), then the inline gap before it, then
		// prepend the increment operator to the variable.
		for k := numIdx; k >= i; k-- {
			s.RemoveAt(k)
		}
		if lv+1 < s.Len() && s.At(lv+1).Kind == token.Whitespace && !hasNewline(s.At(lv+1).Value) {
			s.RemoveAt(lv + 1)
		}
		s.InsertAt(lv, token.Token{Kind: token.Punct, Value: op})
		changed = true
		i = lv
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
// `$a <op>= <operand>;`. Only the single-variable, single-operand,
// semicolon-terminated form is handled; a lone right operand plus the `;`
// terminator removes any operator-precedence ambiguity. Anything with a chained
// or complex right side is left to the upstream fixer.
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

		// Left side: a single plain variable.
		lv := sigPrev(s, i)
		if lv < 0 || s.At(lv).Kind != token.Variable {
			continue
		}
		if p := sigPrev(s, lv); p >= 0 && lvaluePrefix(s.At(p).Value) {
			continue
		}

		// Right side must start with the same variable.
		rv := sigNext(s, i)
		if rv < 0 || s.At(rv).Kind != token.Variable || s.At(rv).Value != s.At(lv).Value {
			continue
		}

		// Then a compound-capable operator.
		opIdx := sigNext(s, rv)
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
