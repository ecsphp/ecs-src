package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/WhiteSpace/FunctionSpacingSniff.php
//
// FunctionSpacing puts Spacing blank lines before and after a function, with
// SpacingBeforeFirst and SpacingAfterLast at the edges of its scope.
type FunctionSpacing struct {
	Spacing            int
	SpacingBeforeFirst int
	SpacingAfterLast   int
}

func (FunctionSpacing) Name() string {
	return `PHP_CodeSniffer\Standards\Squiz\Sniffs\WhiteSpace\FunctionSpacingSniff`
}

func (FunctionSpacing) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/WhiteSpace/FunctionSpacingSniff.php"
}

func (fs FunctionSpacing) WithConfig(config map[string]any) fixer.Fixer {
	fs.Spacing = configInt(config, "spacing", fs.Spacing)
	fs.SpacingBeforeFirst = configInt(config, "spacingBeforeFirst", fs.SpacingBeforeFirst)
	fs.SpacingAfterLast = configInt(config, "spacingAfterLast", fs.SpacingAfterLast)
	return fs
}

func (fs FunctionSpacing) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		f.eachToken(f.isNamedFunction, func(i int) int {
			fs.process(f, i)
			return 0
		})
	})
}

// isFunctionSpacingIgnored is the sniff's $ignore: whitespace and Tokens::METHOD_MODIFIERS.
func (f *slvFile) isFunctionSpacingIgnored(i int) bool {
	if f.toks[i].kind == token.Whitespace {
		return true
	}
	if f.toks[i].kind != token.Keyword {
		return false
	}
	switch strings.ToLower(f.toks[i].orig) {
	case "public", "protected", "private", "static", "abstract", "final":
		return true
	}
	return false
}

// isNamedFunction tells PHP_CodeSniffer's T_FUNCTION: not a closure, and not
// the "function" of "use function", which it retokenizes to T_STRING.
func (f *slvFile) isNamedFunction(i int) bool {
	if !f.isKeyword(i, "function") || f.isClosure(i) || f.isKeyword(f.prevEffective(i-1), "use") {
		return false
	}
	e := f.enclosing(i)
	return !f.is(e, "{") || !f.is(f.prevEffective(e-1), "\\")
}

// phpIndex mirrors PHP reading $tokens[false]: a missing token is token 0.
func phpIndex(i int) int {
	return max(i, 0)
}

func (fs FunctionSpacing) process(f *slvFile, fn int) {
	toks := f.toks
	n := len(toks)
	eol := func(times int) string { return strings.Repeat(f.eol, max(times, 0)) }

	if p := f.prevEffective(fn - 1); p >= 0 && toks[p].kind == token.OpenTag && toks[p].line != 1 {
		return
	}
	closer := f.scopeCloser(fn)
	if closer < 0 {
		closer = -1
		for j := fn; j < n; j++ {
			if f.is(j, ";") {
				closer = j
				break
			}
		}
	}
	closer = phpIndex(closer)

	prev := fn - 1
	for prev >= 0 && f.isFunctionSpacingIgnored(prev) {
		prev--
	}
	startOfDeclarationLine := phpIndex(f.nextOfKind(prev+1, token.Whitespace))
	for i := startOfDeclarationLine; i >= 0 && toks[i].line == toks[startOfDeclarationLine].line; i-- {
		startOfDeclarationLine = i
	}

	prev = startOfDeclarationLine
	if startOfDeclarationLine > 0 {
		seenDocblock := false
		for prev = startOfDeclarationLine - 1; prev > 0; prev-- {
			if toks[prev].kind == token.Whitespace {
				continue
			}
			if !seenDocblock && f.isDocClose(prev) {
				prev = f.docOpener(prev)
				seenDocblock = true
				continue
			}
			if toks[prev].kind == token.AttributeEnd && toks[prev].closer >= 0 {
				prev = toks[prev].closer
				continue
			}
			break
		}
	}
	isFirst := f.is(prev, "{")

	nextNotIgnored := func(from int) int {
		for j := from; j < n; j++ {
			if !f.isFunctionSpacingIgnored(j) {
				return j
			}
		}
		return 0
	}
	next := nextNotIgnored(closer + 1)
	if f.isEmpty(next) && toks[next].line == toks[closer].line {
		next = nextNotIgnored(next + 1)
	}
	isLast := f.is(next, "}")

	nextLineToken := closer + 1
	for ; nextLineToken < n; nextLineToken++ {
		if toks[nextLineToken].doc == slvDocOpen || toks[nextLineToken].line != toks[closer].line {
			break
		}
	}
	required := fs.Spacing
	if isLast {
		required = fs.SpacingAfterLast
	}
	foundLines := required
	nextContent := -1
	if nextLineToken != n-1 && nextLineToken < n {
		if nextContent = f.nextOfKind(nextLineToken, token.Whitespace); nextContent >= 0 {
			foundLines = toks[nextContent].line - toks[nextLineToken].line
		}
	}
	if foundLines != required {
		f.beginChangeset()
		for i := nextLineToken; i <= nextContent; i++ {
			if toks[i].line == toks[nextContent].line {
				f.replaceToken(i, eol(required)+f.tokenContent(i))
				break
			}
			f.replaceToken(i, "")
		}
		f.endChangeset()
	}

	startOfPreamble := phpIndex(f.nextOfKind(prev+1, token.Whitespace))
	prevLineToken := -1
	for i := startOfPreamble; i >= 0; i-- {
		if toks[i].line != toks[startOfPreamble].line {
			prevLineToken = i
			break
		}
	}
	var prevContent int
	if prevLineToken < 0 {
		foundLines, prevContent, prevLineToken = 0, 0, 0
	} else {
		firstBefore := phpIndex(f.prevNonWhitespace(startOfDeclarationLine - 1))
		if toks[firstBefore].kind == token.Comment {
			return
		}
		stopAt := 0
		if owner := f.conditionOwner(prevLineToken); owner >= 0 {
			stopAt = owner
		}
		prevContent = prev
		prevLine := toks[prevContent].line - 1
		foundLines = toks[startOfPreamble].line - toks[prevContent].line - 1
		for i := prevContent; i > stopAt; i-- {
			if f.is(i, "}") {
				if owner := f.scopeCondition(i); owner >= 0 && f.isNamedFunction(owner) {
					return
				}
				break
			}
			if f.isNamedFunction(i) {
				return
			}
			if toks[i].line == prevLine {
				break
			}
		}
	}
	required = fs.Spacing
	if isFirst {
		required = fs.SpacingBeforeFirst
	}
	if foundLines == required {
		return
	}
	nextSpace := fn - 1
	for j := prevContent + 1; j < fn; j++ {
		if toks[j].kind == token.Whitespace {
			nextSpace = j
			break
		}
	}
	if foundLines < required {
		f.replaceToken(prevLineToken, f.tokenContent(prevLineToken)+eol(required-foundLines))
		return
	}
	nextContent = phpIndex(f.nextOfKind(nextSpace+1, token.Whitespace))
	f.beginChangeset()
	for i := nextSpace; i < nextContent; i++ {
		if toks[i].line == toks[prevContent].line {
			continue
		}
		if toks[i].line == toks[nextContent].line {
			f.replaceToken(i, eol(required)+f.tokenContent(i))
			break
		}
		f.replaceToken(i, "")
	}
	f.endChangeset()
}

// conditionOwner is the innermost scope condition holding i (the last key of
// its conditions), -1 when it sits in no scope.
func (f *slvFile) conditionOwner(i int) int {
	for open := f.enclosingScope(i); open >= 0; open = f.enclosingScope(open) {
		if owner := f.scopeCondition(open); owner >= 0 {
			return owner
		}
	}
	return -1
}
