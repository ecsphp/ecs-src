package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PEAR/Sniffs/Classes/ClassDeclarationSniff.php
//
// PEARClassDeclaration puts the opening brace of a class, interface, trait or
// enum alone on the line after its declaration, at the declaration's indent.
type PEARClassDeclaration struct{}

func (PEARClassDeclaration) Name() string {
	return `PHP_CodeSniffer\Standards\PEAR\Sniffs\Classes\ClassDeclarationSniff`
}

func (PEARClassDeclaration) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PEAR/Sniffs/Classes/ClassDeclarationSniff.php"
}

func (PEARClassDeclaration) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		for i := range f.toks {
			if open := phpcsClassyOpener(f, i); open >= 0 {
				pearClassDeclaration(f, i, open)
			}
		}
	})
}

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PSR2/Sniffs/Classes/ClassDeclarationSniff.php
//
// PSR2ClassDeclaration adds to the PEAR checks the spacing of the declaration's
// keywords and names, the layout of a multi-line implements list and the
// placement of the closing brace.
type PSR2ClassDeclaration struct {
	Indent int
}

func (PSR2ClassDeclaration) Name() string {
	return `PHP_CodeSniffer\Standards\PSR2\Sniffs\Classes\ClassDeclarationSniff`
}

func (PSR2ClassDeclaration) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PSR2/Sniffs/Classes/ClassDeclarationSniff.php"
}

func (r PSR2ClassDeclaration) WithConfig(config map[string]any) fixer.Fixer {
	r.Indent = configInt(config, "indent", r.Indent)
	return r
}

func (r PSR2ClassDeclaration) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		for i := range f.toks {
			if open := phpcsClassyOpener(f, i); open >= 0 {
				pearClassDeclaration(f, i, open)
				d := classDecl{keyword: i, open: open, parenOpen: -1, indent: r.Indent}
				d.processOpen(f)
				d.processClose(f)
			}
		}
	})
}

func pearClassDeclaration(f *slvFile, stackPtr, curly int) {
	lastContent := f.prevNonWhitespace(curly - 1)
	classLine, braceLine := f.toks[lastContent].line, f.toks[curly].line
	if braceLine == classLine {
		f.beginChangeset()
		if f.toks[curly-1].kind == token.Whitespace {
			f.replaceToken(curly-1, "")
		}
		f.replaceToken(curly, f.eol+f.tokenContent(curly))
		f.endChangeset()
		return
	}
	if braceLine > classLine+1 {
		f.beginChangeset()
		for i := curly - 1; i > lastContent; i-- {
			if f.toks[i].line == f.toks[curly].line+1 {
				break
			}
			f.replaceToken(i, "")
		}
		f.endChangeset()
		return
	}
	if curly+1 < len(f.toks) && f.toks[curly+1].content != f.eol {
		next := f.nextNonWhitespace(curly+1, len(f.toks))
		if next < 0 || !phpcsIsIgnoreAnnotation(f, next) {
			f.replaceToken(curly, f.tokenContent(curly)+f.eol)
		}
	}
	if f.toks[curly-1].kind != token.Whitespace {
		return
	}
	spaces := f.toks[curly-1].length
	if f.toks[curly-1].content == f.eol {
		spaces = 0
	}
	expected := f.toks[f.firstNonWhitespaceOnLine(stackPtr)].column - 1
	if spaces == expected {
		return
	}
	indent := strings.Repeat(" ", expected)
	if spaces == 0 {
		f.replaceToken(curly, indent+f.tokenContent(curly))
	} else {
		f.replaceToken(curly-1, indent)
	}
}

// phpcsIsIgnoreAnnotation reports whether PHP_CodeSniffer tokenizes the
// comment at i as T_PHPCS_IGNORE.
func phpcsIsIgnoreAnnotation(f *slvFile, i int) bool {
	if !phpcsIsAnnotation(f, i) {
		return false
	}
	text := strings.ToLower(strings.TrimPrefix(strings.TrimLeft(f.toks[i].content, " \t/*#"), "@"))
	return strings.HasPrefix(text, "phpcs:ignore") && !strings.HasPrefix(text, "phpcs:ignorefile")
}

// classDecl is a class-like declaration as PSR2's ClassDeclarationSniff sees it:
// its keyword, its scope opener and, for an anonymous class, its arguments.
type classDecl struct {
	keyword, open         int
	anon                  bool
	parenOpen, parenClose int
	indent                int
}

func (d classDecl) processOpen(f *slvFile) {
	stackPtr := d.keyword

	prevNonSpace := f.prevNonWhitespace(stackPtr - 1)
	prevNonEmpty := f.prevEffective(stackPtr - 1)
	if prevNonEmpty >= 0 && (f.isKeyword(prevNonEmpty, "abstract") || f.isKeyword(prevNonEmpty, "final") || f.isKeyword(prevNonEmpty, "readonly")) {
		spaces, newline := 0, false
		if f.toks[prevNonEmpty].line != f.toks[stackPtr].line {
			newline = true
		} else if f.toks[stackPtr-1].kind == token.Whitespace {
			spaces = f.toks[stackPtr-1].length
		}
		if (newline || spaces != 1) && prevNonSpace == prevNonEmpty {
			if !newline && spaces == 0 {
				f.replaceToken(stackPtr, " "+f.tokenContent(stackPtr))
			} else {
				f.beginChangeset()
				f.replaceToken(stackPtr-1, " ")
				for i := stackPtr - 2; i > prevNonSpace; i-- {
					f.replaceToken(i, " ")
				}
				f.endChangeset()
			}
		}
	}

	classIndent := 0
	for i := stackPtr - 1; i > 0; i-- {
		if f.toks[i].line == f.toks[stackPtr].line {
			continue
		}
		if f.toks[i+1].kind == token.Whitespace {
			classIndent = f.toks[i+1].length
		}
		break
	}

	className := -1
	checkSpacing := true
	if !d.anon {
		for i := stackPtr; i < len(f.toks); i++ {
			if f.toks[i].kind == token.Ident && !strings.Contains(f.toks[i].orig, "\\") {
				className = i
				break
			}
		}
	} else if next := f.nextNonWhitespace(stackPtr+1, len(f.toks)); next == d.open && f.toks[next].line > f.toks[stackPtr].line {
		checkSpacing = false
	}

	if checkSpacing {
		gap, newline := 0, false
		switch {
		case f.toks[stackPtr+1].kind != token.Whitespace:
		case f.toks[stackPtr+2].line != f.toks[stackPtr].line:
			newline = true
		default:
			gap = f.toks[stackPtr+1].length
		}
		if newline || gap != 1 {
			if !newline && gap == 0 {
				f.replaceToken(stackPtr, f.tokenContent(stackPtr)+" ")
			} else {
				f.replaceToken(stackPtr+1, " ")
			}
		}
	}

	if className >= 0 && className+2 < len(f.toks) && f.toks[className+2].line == f.toks[className].line {
		if len(f.toks[className+1].content) != 1 {
			f.replaceToken(className+1, " ")
		}
	}

	openingBrace := d.open
	compareToken := stackPtr
	if d.anon && d.parenOpen >= 0 {
		compareToken = d.parenClose
	}

	for _, keywordType := range []string{"extends", "implements"} {
		keyword := -1
		for i := compareToken + 1; i < openingBrace; i++ {
			if f.isKeyword(i, keywordType) {
				keyword = i
				break
			}
		}
		if keyword < 0 {
			continue
		}
		if f.toks[keyword].line != f.toks[compareToken].line {
			f.beginChangeset()
			var comments []string
			for i := compareToken + 1; i < keyword; i++ {
				isComment := f.toks[i].kind == token.Comment && !phpcsIsAnnotation(f, i)
				if isComment {
					comments = append(comments, strings.TrimSpace(f.toks[i].content))
				}
				if f.toks[i].kind == token.Whitespace || isComment {
					f.replaceToken(i, " ")
				}
			}
			f.replaceToken(compareToken, f.tokenContent(compareToken)+" ")
			if len(comments) > 0 {
				i := keyword
				for f.toks[i+1].line == f.toks[keyword].line {
					i++
				}
				f.replaceToken(i, " "+strings.Join(comments, " ")+f.tokenContent(i))
			}
			f.endChangeset()
		} else if f.toks[keyword-1].length != 1 {
			f.replaceToken(keyword-1, " ")
		}
	}

	keywordType := "implements"
	if f.isKeyword(stackPtr, "interface") {
		keywordType = "extends"
	}
	implements := -1
	for i := stackPtr + 1; i < openingBrace; i++ {
		if f.isKeyword(i, keywordType) {
			implements = i
			break
		}
	}
	multiLineImplements := false
	if implements >= 0 {
		prev := openingBrace - 1
		for prev >= implements && f.isEmpty(prev) {
			prev--
		}
		if f.toks[prev].line != f.toks[implements].line {
			multiLineImplements = true
		}
	}

	start := stackPtr
	if className >= 0 {
		start = className
	} else if d.parenOpen >= 0 {
		start = d.parenClose
	}
	var classNames []int
	for i := start + 2; i < openingBrace-1; i++ {
		if f.toks[i].kind == token.Ident || f.isKeyword(i, keywordType) {
			classNames = append(classNames, i)
		}
	}

	checkingImplements := false
	implementsToken := -1
	nextComma := -1
	for n, className := range classNames {
		if f.isKeyword(className, keywordType) {
			checkingImplements = true
			implementsToken = className
			continue
		}
		if checkingImplements && multiLineImplements {
			prev := className - 1
			for prev >= implements && f.toks[prev].kind == token.Whitespace {
				prev--
			}
			prevIsComment := f.toks[prev].kind == token.Comment || f.toks[prev].kind == token.DocComment
			switch {
			case prev == implementsToken && f.toks[className].line != f.toks[prev].line+1,
				prev != implementsToken && (!prevIsComment && f.toks[prev].line != f.toks[className].line-1 || f.toks[prev].line == f.toks[className].line):
				f.beginChangeset()
				for i := prev + 1; i < className; i++ {
					if f.toks[i].kind != token.Whitespace {
						break
					}
					f.replaceToken(i, "")
				}
				f.replaceToken(prev, f.tokenContent(prev)+f.eol)
				f.endChangeset()
			default:
				prev := className - 1
				for prev >= implements && f.toks[prev].kind != token.Whitespace {
					prev--
				}
				found := 0
				if f.toks[prev].line == f.toks[className].line {
					found = f.toks[prev].length
				}
				expected := classIndent + d.indent
				if found != expected {
					padding := strings.Repeat(" ", expected)
					if found == 0 {
						f.replaceToken(prev, f.tokenContent(prev)+padding)
					} else {
						f.replaceToken(prev, padding)
					}
				}
			}
		} else if f.is(className-1, ",") {
			if nextComma >= 0 {
				f.replaceToken(nextComma+1, " "+f.tokenContent(nextComma+1))
			}
		} else {
			prev := className - 1
			last := f.prevNonWhitespace(prev)
			if f.content(last+1, prev) != " " {
				if f.toks[prev].kind == token.Whitespace {
					f.beginChangeset()
					f.replaceToken(prev, " ")
					for prev--; f.toks[prev].kind == token.Whitespace; prev-- {
						f.replaceToken(prev, " ")
					}
					f.endChangeset()
				} else {
					f.replaceToken(prev, f.tokenContent(prev)+" ")
				}
			}
		}

		if checkingImplements && !f.is(className+1, ",") {
			if n != len(classNames)-1 && !f.isKeyword(className+2, keywordType) {
				f.replaceToken(className+1, "")
			}
			nextComma = -1
			for i := className; i < len(f.toks); i++ {
				if f.is(i, ",") {
					nextComma = i
					break
				}
			}
		} else {
			nextComma = className + 1
		}
	}
}

func (d classDecl) processClose(f *slvFile) {
	closeBrace := f.toks[d.open].closer
	prevContent := f.prevNonWhitespace(closeBrace - 1)
	if prevContent == d.open || f.toks[prevContent].line == f.toks[closeBrace].line-1 {
		return
	}
	f.beginChangeset()
	for i := prevContent + 1; f.toks[i].line != f.toks[closeBrace].line; i++ {
		f.replaceToken(i, "")
	}
	if !strings.Contains(f.toks[prevContent].content, f.eol) {
		f.replaceToken(prevContent, f.tokenContent(prevContent)+f.eol)
	}
	f.endChangeset()
}

// ECS stops at a fixer conflict between the extends and implements fixes.
func (PSR2ClassDeclaration) phpcsOnePass() {}

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PSR12/Sniffs/Classes/AnonClassDeclarationSniff.php
//
// PSR12AnonClassDeclaration applies PSR2's class declaration checks to an
// anonymous class, lays out its arguments like a function declaration's, one
// per line once they span lines, and keeps its opening brace off a blank line.
type PSR12AnonClassDeclaration struct {
	Indent int
}

func (PSR12AnonClassDeclaration) Name() string {
	return `PHP_CodeSniffer\Standards\PSR12\Sniffs\Classes\AnonClassDeclarationSniff`
}

func (PSR12AnonClassDeclaration) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PSR12/Sniffs/Classes/AnonClassDeclarationSniff.php"
}

func (r PSR12AnonClassDeclaration) WithConfig(config map[string]any) fixer.Fixer {
	r.Indent = configInt(config, "indent", r.Indent)
	return r
}

func (r PSR12AnonClassDeclaration) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		for i := range f.toks {
			if d, ok := phpcsAnonClass(f, i); ok {
				d.indent = r.Indent
				r.process(f, d)
			}
		}
	})
}

// phpcsIsAnonClassKeyword reports whether PHP_CodeSniffer retokenizes the
// "class" at i as T_ANON_CLASS: the keyword is followed by "(", "{", extends or
// implements rather than a name.
func phpcsIsAnonClassKeyword(f *slvFile, i int) bool {
	if !f.isKeyword(i, "class") {
		return false
	}
	next := f.nextEffective(i+1, len(f.toks))
	return f.is(next, "(") || f.is(next, "{") || f.isKeyword(next, "extends") || f.isKeyword(next, "implements")
}

// phpcsAnonClass returns the anonymous class whose "class" keyword is at i.
func phpcsAnonClass(f *slvFile, i int) (classDecl, bool) {
	if !phpcsIsAnonClassKeyword(f, i) {
		return classDecl{}, false
	}
	d := classDecl{keyword: i, anon: true, parenOpen: -1, parenClose: -1, open: -1}
	if n := f.nextEffective(i+1, len(f.toks)); f.is(n, "(") && f.toks[n].closer >= 0 {
		d.parenOpen, d.parenClose = n, f.toks[n].closer
	}
	if sc := f.scopeCloser(i); sc >= 0 {
		d.open = f.toks[sc].closer
	}
	return d, d.open >= 0
}

func (r PSR12AnonClassDeclaration) process(f *slvFile, d classDecl) {
	stackPtr := d.keyword
	d.processOpen(f)
	d.processClose(f)

	if d.parenOpen >= 0 {
		if anonMultiLineArguments(f, stackPtr, d.parenOpen) {
			anonProcessBracket(f, d.parenOpen)
			anonProcessArgumentList(f, d, r.Indent)
		} else {
			anonSingleLineArguments(f, d)
		}
		anonArgumentSpacing(f, d.parenOpen)
	}

	opener := d.open
	if f.toks[opener].line == f.toks[stackPtr].line {
		return
	}
	prev := f.prevNonWhitespace(opener - 1)
	implements := -1
	for i := opener - 1; i >= stackPtr; i-- {
		if f.isKeyword(i, "implements") {
			implements = i
			break
		}
	}
	if implements >= 0 && f.toks[opener].line != f.toks[implements].line && f.toks[opener].line == f.toks[prev].line {
		indent := strings.Repeat(" ", f.toks[f.firstNonWhitespaceOnLine(stackPtr)].column-1)
		f.beginChangeset()
		if f.toks[prev+1].kind == token.Whitespace {
			f.replaceToken(prev+1, "")
		}
		f.replaceToken(prev, f.tokenContent(prev)+f.eol)
		f.replaceToken(opener, indent+f.tokenContent(opener))
		f.endChangeset()
	}
	if f.toks[opener].line > f.toks[prev].line+1 {
		f.beginChangeset()
		for x := prev + 1; x < opener; x++ {
			if f.toks[x].line == f.toks[prev].line {
				continue
			}
			if f.toks[x].line == f.toks[opener].line {
				break
			}
			f.replaceToken(x, "")
		}
		f.endChangeset()
	}
}

func anonSingleLineArguments(f *slvFile, d classDecl) {
	openBracket, closeBracket := d.parenOpen, d.parenClose
	if openBracket == closeBracket-1 {
		return
	}
	if f.toks[openBracket+1].kind == token.Whitespace {
		f.replaceToken(openBracket+1, "")
	}
	prev := f.prevNonWhitespace(closeBracket - 1)
	if slvHeredocEnd(f, prev) {
		return
	}
	spaces, newline := 0, false
	if f.toks[prev].line != f.toks[closeBracket].line {
		newline = true
	} else if f.toks[closeBracket-1].kind == token.Whitespace {
		spaces = f.toks[closeBracket-1].length
	}
	if !newline && spaces == 0 {
		return
	}
	if !newline {
		f.replaceToken(closeBracket-1, "")
		return
	}
	f.beginChangeset()
	closing := ")"
	if next := f.nextNonWhitespace(closeBracket+1, len(f.toks)); f.is(next, ";") {
		closing += ";"
		for i := closeBracket + 1; i <= next; i++ {
			f.replaceToken(i, "")
		}
	}
	p := closeBracket - 1
	for f.isEmpty(p) {
		if f.toks[p].kind == token.Comment && strings.Contains(f.toks[p].content, "*/") {
			break
		}
		p--
	}
	f.replaceToken(p, f.tokenContent(p)+closing)
	for i := f.prevNonWhitespace(closeBracket-1) + 1; i <= closeBracket; i++ {
		f.replaceToken(i, "")
	}
	f.endChangeset()
}

// slvHeredocEnd reports whether the token at i closes a heredoc or nowdoc.
func slvHeredocEnd(f *slvFile, i int) bool {
	return f.toks[i].heredoc && !strings.Contains(f.toks[i].orig, "\n")
}

// anonMultiLineArguments is MultiLineFunctionDeclarationSniff::isMultiLineDeclaration.
func anonMultiLineArguments(f *slvFile, stackPtr, openBracket int) bool {
	next := f.nextEffective(openBracket+1, len(f.toks))
	if f.toks[next].line != f.toks[stackPtr].line {
		return true
	}
	closeBracket := f.toks[openBracket].closer
	end := slvEndOfStatement(f, openBracket+1)
	for f.is(end, ",") {
		next := f.nextEffective(end+1, closeBracket)
		if next < 0 {
			return false
		}
		if f.toks[next].line != f.toks[end].line {
			return true
		}
		end = slvEndOfStatement(f, next)
	}
	next = f.nextEffective(end+1, closeBracket)
	return next >= 0 && f.toks[next].line != f.toks[end].line
}

// slvEndOfStatement is File::findEndOfStatement, without its match arm handling.
func slvEndOfStatement(f *slvFile, start int) int {
	lastNotEmpty := start
	ternaries := 0
	for i := start; i < len(f.toks); i++ {
		t := f.toks[i]
		if i != start {
			switch {
			case f.is(i, "?"):
				ternaries++
			case f.is(i, ":") && ternaries > 0:
				ternaries--
			case f.is(i, ",") || f.is(i, ";") || f.is(i, ":") || f.is(i, "=>") && f.isDoubleArrow(i):
				return i
			case f.is(i, ")") || f.is(i, "]") || f.is(i, "}") || t.kind == token.OpenTag || t.kind == token.CloseTag:
				return lastNotEmpty
			}
		}
		if sc := f.scopeCloser(i); sc >= 0 {
			if f.isKeyword(i, "fn") {
				lastNotEmpty = sc
				i = sc - 1
				continue
			}
			if i == start && t.kind == token.Keyword {
				return sc
			}
			i = sc
		} else if (f.is(i, "(") || f.is(i, "[")) && t.closer >= 0 {
			i = t.closer
		}
		if !f.isEmpty(i) {
			lastNotEmpty = i
		}
	}
	return len(f.toks) - 1
}

// anonProcessBracket is MultiLineFunctionDeclarationSniff::processBracket.
func anonProcessBracket(f *slvFile, openBracket int) {
	closeBracket := f.toks[openBracket].closer
	if f.toks[openBracket].line != f.toks[closeBracket].line {
		if next := f.nextEffective(openBracket+1, len(f.toks)); f.toks[next].line == f.toks[openBracket].line {
			f.replaceToken(openBracket, f.tokenContent(openBracket)+f.eol)
		}
	}
	for i := openBracket + 1; i < closeBracket; i++ {
		t := f.toks[i]
		if (f.is(i, "(") || f.is(i, "[") || t.kind == token.AttributeStart) && t.closer >= 0 {
			i = t.closer
			continue
		}
		if !f.is(i, ",") {
			continue
		}
		if next := f.nextEffective(i+1, len(f.toks)); f.toks[next].line == t.line {
			f.replaceToken(i, f.tokenContent(i)+f.eol)
		}
	}
}

// anonProcessArgumentList is PEAR FunctionDeclarationSniff::processArgumentList.
func anonProcessArgumentList(f *slvFile, d classDecl, indent int) {
	stackPtr := d.keyword
	i := stackPtr - 1
	for i >= 0 && f.toks[i].line == f.toks[stackPtr].line {
		i--
	}
	i++
	functionIndent := 0
	if f.toks[i].kind == token.Whitespace {
		functionIndent = f.toks[i].length
	}

	closeBracket := d.parenClose
	prev := f.prevNonWhitespace(closeBracket - 1)
	if f.toks[closeBracket].line != f.toks[d.parenOpen].line && f.toks[prev].line == f.toks[closeBracket].line {
		f.replaceToken(closeBracket, f.eol+f.tokenContent(closeBracket))
	}

	isCommentLike := func(i int) bool { return f.toks[i].kind == token.Comment }
	lastLine := f.toks[d.parenOpen].line
	for i := d.parenOpen + 1; i < closeBracket; i++ {
		t := f.toks[i]
		if t.line != lastLine {
			expected := functionIndent + indent
			if i == closeBracket || t.kind == token.Whitespace && i+1 == closeBracket {
				expected = functionIndent
			}
			found := 0
			switch {
			case t.kind == token.Whitespace && t.line != f.toks[i+1].line:
				f.replaceToken(i, "")
				continue
			case t.kind == token.Whitespace:
				found = t.length
			case t.doc == slvDocWhitespace:
				found = t.length
				expected++
			case isCommentLike(i) && isCommentLike(i-1) && f.toks[i-1].line == lastLine:
				trimmed := strings.TrimLeft(t.content, " \t\n\r\x00\x0B")
				if trimmed == "" {
					lastLine = t.line
					continue
				}
				found = len(t.content) - len(trimmed)
				if trimmed[0] == '*' {
					expected++
				} else if found >= expected {
					lastLine = t.line
					continue
				}
			}
			if expected != found {
				spaces := strings.Repeat(" ", expected)
				switch {
				case found == 0:
					f.replaceToken(i, spaces+f.tokenContent(i))
				case isCommentLike(i):
					f.replaceToken(i, spaces+strings.TrimLeft(t.content, " \t\n\r\x00\x0B"))
				default:
					f.replaceToken(i, spaces)
				}
			}
			lastLine = t.line
		}
		if f.is(i, "(") && t.closer >= 0 {
			if p := f.prevEffective(i - 1); !f.isKeyword(p, "use") {
				i = t.closer
				lastLine = f.toks[i].line
				continue
			}
		}
		if f.isShortArrayOpen(i) || f.isKeyword(i, "array") {
			if f.is(i, "[") {
				i = t.closer
			} else if o := f.nextEffective(i+1, len(f.toks)); f.is(o, "(") && f.toks[o].closer >= 0 {
				i = f.toks[o].closer
			}
			lastLine = f.toks[i].line
			continue
		}
		if t.kind == token.AttributeStart && t.closer >= 0 {
			i = t.closer
			lastLine = f.toks[i].line
			continue
		}
	}
}

// anonArgumentSpacing is FunctionCallArgumentSpacingSniff::checkSpacing.
func anonArgumentSpacing(f *slvFile, openBracket int) {
	closeBracket := f.toks[openBracket].closer
	for sep := openBracket + 1; sep < closeBracket; sep++ {
		switch {
		case f.isKeyword(sep, "fn"):
			if sc := f.scopeCloser(sep); sc >= 0 {
				sep = sc - 1
			}
			continue
		case f.isKeyword(sep, "function") || f.isKeyword(sep, "match"):
			if sc := f.scopeCloser(sep); sc >= 0 {
				sep = sc
			}
			continue
		case f.isKeyword(sep, "class"):
			if d, ok := phpcsAnonClass(f, sep); ok {
				sep = f.toks[d.open].closer
			}
			continue
		case f.isShortArrayOpen(sep):
			sep = f.toks[sep].closer
			continue
		case !f.is(sep, ","):
			continue
		}
		if slvInnermostParen(f, sep) != openBracket {
			continue
		}
		if f.toks[sep-1].kind == token.Whitespace {
			prev := f.prevEffective(sep - 2)
			if !f.toks[prev].heredoc {
				f.beginChangeset()
				if f.toks[prev].line != f.toks[sep].line {
					f.replaceToken(prev, f.tokenContent(prev)+",")
					f.replaceToken(sep, "")
				} else {
					f.replaceToken(sep-1, "")
				}
				f.endChangeset()
			}
		}
		if f.toks[sep+1].kind != token.Whitespace {
			if sep+1 != closeBracket {
				f.replaceToken(sep, f.tokenContent(sep)+" ")
			}
		} else if next := f.nextEffective(sep+1, len(f.toks)); f.toks[next].line == f.toks[sep].line && f.toks[sep+1].length > 1 {
			f.replaceToken(sep+1, " ")
		}
	}
}

// slvInnermostParen returns the innermost "(" holding i, or -1.
func slvInnermostParen(f *slvFile, i int) int {
	for j := f.enclosing(i); j >= 0; j = f.enclosing(j) {
		if f.is(j, "(") {
			return j
		}
	}
	return -1
}
