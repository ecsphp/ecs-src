package rules

import (
	"github.com/ecsphp/ecs-src/blink/internal/token"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/PHP/DisallowMultipleAssignmentsSniff.php
//
// DisallowMultipleAssignments reports an assignment that is not the first
// code of its line or statement, as in "$a = $b = 1". It fixes nothing.
type DisallowMultipleAssignments struct{}

func (DisallowMultipleAssignments) Name() string {
	return `PHP_CodeSniffer\Standards\Squiz\Sniffs\PHP\DisallowMultipleAssignmentsSniff`
}

func (DisallowMultipleAssignments) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/PHP/DisallowMultipleAssignmentsSniff.php"
}

func (DisallowMultipleAssignments) Fix(*tokens.Stream) bool { return false }

func (r DisallowMultipleAssignments) Errors(s *tokens.Stream) []fixer.CodingStandardError {
	return squizReport(s, r.Name(), r.process)
}

func (DisallowMultipleAssignments) process(f *slvFile, sc *phpcsScope, report func(i int, code, msg string)) {
	for i := range f.toks {
		if !f.is(i, "=") {
			continue
		}
		fn := squizReportPrevLocal(f, sc, i-1, func(j int) bool { return f.isKeyword(j, "function") || f.isKeyword(j, "fn") })
		if fn >= 0 {
			open, closeIdx := squizMultiAssignParams(f, fn)
			if closeIdx < 0 {
				continue
			}
			if open < i && closeIdx > i {
				continue
			}
		}
		nested := squizReportNested(sc, i)
		inWhile := false
		for _, open := range nested {
			inWhile = inWhile || f.isKeyword(squizReportParenOwner(f, sc, open), "while")
		}
		if inWhile {
			continue
		}
		if conds := sc.conditions(i); len(conds) > 0 && squizMultiAssignIsOO(f, conds[len(conds)-1]) {
			continue
		}
		v := i - 1
		for ; v >= 0; v-- {
			if f.is(v, ";") && sc.code[v] == ";" || f.is(v, "{") && sc.code[v] == "{" || f.toks[v].kind == token.CloseTag {
				v = -2
				break
			}
			if f.is(v, ")") && f.toks[v].closer >= 0 && f.toks[v].closer < v {
				v = f.toks[v].closer
				continue
			}
			if (f.is(v, "]") || f.is(v, "}")) && f.toks[v].closer >= 0 {
				v = f.toks[v].closer
				continue
			}
			if f.is(v, "[") && f.toks[v].closer >= 0 {
				continue
			}
			if f.toks[v].kind == token.Variable {
				if p := squizReportPrevNonEmpty(f, v-1); f.is(p, "->") {
					v = p
					continue
				}
				break
			}
		}
		if v <= 0 {
			continue
		}
		start := sc.startOfStatement(v)
		allowed := func(j int) bool {
			return f.isEmpty(j) || squizReportIsPlainName(f, j) || f.is(j, "::") || f.is(j, "@") || f.is(j, "$") ||
				f.toks[j].kind == token.Ident || f.isKeyword(j, "static")
		}
		p := v - 1
		for p >= 0 && allowed(p) {
			p--
		}
		p = max(p, 0)
		if p < start && !f.is(p, "(") && !f.is(p, "[") {
			p = start
		}
		if f.is(p, "(") && f.isKeyword(squizReportParenOwner(f, sc, p), "for") {
			continue
		}
		if f.toks[p].kind == token.Variable || f.toks[p].kind == token.OpenTag || squizReportIsInlineThen(f, sc, p) ||
			sc.code[p] == "inline_else" || f.is(p, ";") || f.is(p, ")") || allowed(p) {
			continue
		}
		code := "Found"
		for _, open := range nested {
			switch o := squizReportParenOwner(f, sc, open); {
			case f.isKeyword(o, "if"), f.isKeyword(o, "elseif"), f.isKeyword(o, "switch"), f.isKeyword(o, "for"), f.isKeyword(o, "match"):
				code = "FoundInControlStructure"
			}
			if code != "Found" {
				break
			}
		}
		report(i, code, "Assignments must be the first block of code on a line")
	}
}

// squizMultiAssignParams is the parameter list of a function, closure or arrow
// function, or -1, -1.
func squizMultiAssignParams(f *slvFile, fn int) (int, int) {
	j := fn + 1
	for j < len(f.toks) && (f.isEmpty(j) || f.toks[j].kind == token.Ident || f.is(j, "&") || f.toks[j].kind == token.Keyword && !f.isKeyword(j, "use")) {
		j++
	}
	if f.is(j, "(") && f.toks[j].closer > j {
		return j, f.toks[j].closer
	}
	return -1, -1
}

// squizMultiAssignIsOO is Tokens::OO_SCOPE_TOKENS.
func squizMultiAssignIsOO(f *slvFile, i int) bool {
	return f.isKeyword(i, "class") || f.isKeyword(i, "interface") || f.isKeyword(i, "trait") || f.isKeyword(i, "enum")
}
