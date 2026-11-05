package rules

import (
	"slices"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
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
		s.ReplaceRange(i, numIdx, nil)
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
// IncrementStyle moves an increment/decrement operator to the configured side
// of its operand. With the default `pre` style a post-increment (`$i++`,
// `$a->b[0]--`, `Foo::$n++`, ...) becomes a pre-increment; with `post` the
// reverse. Like upstream, only an operation whose value is discarded is moved:
// the operand starts a statement (after `;`, `{`, `}`, `)` or the open tag) and
// is followed by `;` or `)` (a `for` step, the body of a brace-less `if`). The
// operand is found by upstream's walk (variables, `$$`/`${}`, property and
// static chains, offsets); the whitespace and comments around the operator stay
// where they were.
type IncrementStyle struct {
	// Post selects the `post` style (`++$i` -> `$i++`); the default is `pre`.
	Post bool
}

func (IncrementStyle) Name() string {
	return `PhpCsFixer\Fixer\Operator\IncrementStyleFixer`
}

func (IncrementStyle) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Operator/IncrementStyleFixer.php"
}

// WithConfig reads `style` (`pre` or `post`).
func (f IncrementStyle) WithConfig(config map[string]any) fixer.Fixer {
	if style, ok := config["style"].(string); ok {
		f.Post = style == "post"
	}
	return f
}

func (f IncrementStyle) Fix(s *tokens.Stream) bool {
	changed := false
	for i := s.Len() - 1; i >= 0; i-- {
		t := s.At(i)
		if t.Kind != token.Punct || (t.Value != "++" && t.Value != "--") {
			continue
		}
		successor := incIsSuccessor(s, i)
		if !f.Post && successor {
			next := sigNext(s, i)
			if next < 0 || !punctIn(s.At(next), ";", ")") {
				continue
			}
			start := incFindStart(s, i)
			if start < 0 || !incStatementBoundary(s, sigPrev(s, start)) {
				continue
			}
			removeMergingWhitespace(s, i)
			s.InsertAt(start, t)
			changed = true
		} else if f.Post && !successor {
			if !incStatementBoundary(s, sigPrev(s, i)) {
				continue
			}
			end := incFindEnd(s, i)
			if end < 0 {
				continue
			}
			if next := sigNext(s, end); next < 0 || !punctIn(s.At(next), ";", ")") {
				continue
			}
			at := end + 1
			for at < s.Len() && s.At(at).Kind == token.Whitespace {
				at++
			}
			if at == i+1 {
				continue // no operand (`foo()++`): the operator would stay put
			}
			s.InsertAt(at, t)
			removeMergingWhitespace(s, i)
			changed = true
		}
	}
	return changed
}

// removeMergingWhitespace removes the token at i and joins the whitespace
// tokens that end up adjacent, so the stream keeps one token per gap.
func removeMergingWhitespace(s *tokens.Stream, i int) {
	s.RemoveAt(i)
	if i > 0 && i < s.Len() && s.At(i-1).Kind == token.Whitespace && s.At(i).Kind == token.Whitespace {
		s.SetValue(i-1, s.At(i-1).Value+s.At(i).Value)
		s.RemoveAt(i)
	}
}

// incStatementBoundary reports whether the token at p may precede a moved
// increment: `;`, `{`, `}`, `)` or the open tag (upstream's list).
func incStatementBoundary(s *tokens.Stream, p int) bool {
	if p < 0 {
		return false
	}
	t := s.At(p)
	return t.Kind == token.OpenTag || punctIn(t, ";", "{", "}", ")")
}

// incIsSuccessor ports TokensAnalyzer::isUnarySuccessorOperator: the operator
// at i follows its operand (an offset `]`, a name, a variable, or the `}` of a
// dynamic `${…}`/`->{…}` access).
func incIsSuccessor(s *tokens.Stream, i int) bool {
	p := sigPrev(s, i)
	if p < 0 {
		return false
	}
	t := s.At(p)
	switch {
	case t.Kind == token.Ident, t.Kind == token.Variable:
		return true
	case t.Kind != token.Punct:
		return false
	case t.Value == "]":
		return true
	case t.Value == "}":
		return dynamicBraceClose(s, p)
	}
	return false
}

// dynamicBraceClose reports whether the "}" at i closes a dynamic variable or
// property brace (`${…}`, `->{…}`, `?->{…}`), which PHP-CS-Fixer retags.
func dynamicBraceClose(s *tokens.Stream, i int) bool {
	open := s.MatchBackward(i)
	return open >= 0 && dynamicBraceOpen(s, open)
}

// dynamicBraceOpen reports whether the "{" at i opens a dynamic variable or
// property brace.
func dynamicBraceOpen(s *tokens.Stream, i int) bool {
	p := sigPrev(s, i)
	if p < 0 {
		return false
	}
	t := s.At(p)
	return t.Kind == token.Variable && t.Value == "$" || punctIn(t, "->", "?->")
}

// incFindStart ports AbstractIncrementOperatorFixer::findStart: the first token
// of the operand ending before the post-increment at i, or -1.
func incFindStart(s *tokens.Stream, i int) int {
	for {
		i = sigPrev(s, i)
		if i < 0 {
			return -1
		}
		if t := s.At(i); punctIn(t, ")", "]", "}") {
			if i = s.MatchBackward(i); i < 0 {
				return -1
			}
		}
		if s.At(i).Kind == token.Variable {
			break
		}
	}
	p := sigPrev(s, i)
	if p < 0 {
		return i
	}
	pt := s.At(p)
	switch {
	case pt.Kind == token.Variable && pt.Value == "$":
		return incFindStart(s, i)
	case punctIn(pt, "->", "?->"):
		return incFindStart(s, p)
	case pt.Kind == token.Punct && pt.Value == "::":
		pp := sigPrev(s, p)
		if pp < 0 || !incClassNamePart(s.At(pp)) {
			return incFindStart(s, p)
		}
		// the class name: back over its parts (whitespace stops the walk)
		j := p - 1
		for j >= 0 && (incClassNamePart(s.At(j)) || s.At(j).Kind == token.Punct && s.At(j).Value == `\`) {
			j--
		}
		return sigNext(s, j)
	}
	return i
}

// incClassNamePart reports whether t is a T_STRING or T_STATIC token.
func incClassNamePart(t token.Token) bool {
	return t.Kind == token.Ident || t.Kind == token.Keyword && t.Value == "static"
}

// incFindEnd ports IncrementStyleFixer::findEnd: the last token of the operand
// following the pre-increment at i, or -1.
func incFindEnd(s *tokens.Stream, i int) int {
	next := sigNext(s, i)
	for next >= 0 {
		t := s.At(next)
		if punctIn(t, "(", "[") || t.Kind == token.Punct && t.Value == "{" && dynamicBraceOpen(s, next) {
			if next = s.MatchForward(next); next < 0 {
				return -1
			}
		} else if t.Kind != token.Variable && !incClassNamePart(t) && (t.Kind != token.Punct || t.Value != `\`) {
			break
		}
		i = next
		next = sigNext(s, next)
	}
	if next < 0 {
		return i
	}
	switch t := s.At(next); {
	case punctIn(t, "->", "?->"):
		return incFindEnd(s, next)
	case t.Kind == token.Punct && t.Value == "::":
		return incFindEnd(s, sigNext(s, next))
	}
	return i
}

// punctIn reports whether t is a punctuation token with one of the values vs.
func punctIn(t token.Token, vs ...string) bool {
	return t.Kind == token.Punct && slices.Contains(vs, t.Value)
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
		// an interpolated string or backtick literal is one operand, up to its Tail
		if s.At(operandIdx).Part == token.Head {
			chunks := literalChunks(s, operandIdx)
			if chunks == nil {
				continue
			}
			operandIdx = chunks[len(chunks)-1]
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
		s.ReplaceRange(i+1, opIdx, nil)
		changed = true
	}
	return changed
}
