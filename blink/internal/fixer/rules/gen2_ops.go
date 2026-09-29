package rules

import (
	"blink/internal/token"
	"blink/internal/tokens"
)

// This file implements a PHP-CS-Fixer Operator rule on the flat token stream,
// following the same conservative approach as gen_ops_inc.go: only the
// token-adjacency case that is provably semantics-preserving is rewritten,
// everything else is left untouched. Helpers (sigPrev/sigNext/spanClean/
// lvaluePrefix/shortOperand) are reused from gen_ops_inc.go.

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Operator/AssignNullCoalescingToCoalesceEqualFixer.php
//
// AssignNullCoalescingToCoalesceEqual rewrites `$a = $a ?? $b;` to `$a ??= $b;`.
// Only the single-variable, single-operand, semicolon-terminated form is
// handled; a lone right operand plus the `;` terminator removes any
// operator-precedence ambiguity. Anything with a chained or complex right side
// is left to the upstream fixer. The lexer emits `??` and `??=` each as a single
// Punct token.
type AssignNullCoalescingToCoalesceEqual struct{}

func (AssignNullCoalescingToCoalesceEqual) Name() string {
	return `PhpCsFixer\Fixer\Operator\AssignNullCoalescingToCoalesceEqualFixer`
}

func (AssignNullCoalescingToCoalesceEqual) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Operator/AssignNullCoalescingToCoalesceEqualFixer.php"
}

func (AssignNullCoalescingToCoalesceEqual) Fix(s *tokens.Stream) bool {
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

		// Then the null-coalescing operator (single Punct token).
		opIdx := sigNext(s, rv)
		if opIdx < 0 || s.At(opIdx).Kind != token.Punct || s.At(opIdx).Value != "??" {
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

		// `= $a ??` becomes `??=`: retag the `=` and drop the duplicate operand
		// and operator.
		s.SetValue(i, "??=")
		for k := opIdx; k >= i+1; k-- {
			s.RemoveAt(k)
		}
		changed = true
	}
	return changed
}
