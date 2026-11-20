package rules

import (
	"errors"
	"regexp"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/ControlStructures/BlockControlStructureSpacingSniff.php
//
// SlevomatBlockControlStructureSpacing sets the blank lines around block
// control structures (if, do, while, for, foreach, switch, try, case, default).
type SlevomatBlockControlStructureSpacing struct {
	LinesCountBefore, LinesCountBeforeFirst, LinesCountAfter, LinesCountAfterLast int
	ControlStructures                                                             []string
}

func (SlevomatBlockControlStructureSpacing) Name() string {
	return `SlevomatCodingStandard\Sniffs\ControlStructures\BlockControlStructureSpacingSniff`
}

func (SlevomatBlockControlStructureSpacing) SourceURL() string {
	return fixer.SourceURLFor(SlevomatBlockControlStructureSpacing{}.Name())
}

func (r SlevomatBlockControlStructureSpacing) WithConfig(config map[string]any) fixer.Fixer {
	r.LinesCountBefore = configInt(config, "linesCountBefore", r.LinesCountBefore)
	r.LinesCountBeforeFirst = configInt(config, "linesCountBeforeFirst", r.LinesCountBeforeFirst)
	r.LinesCountAfter = configInt(config, "linesCountAfter", r.LinesCountAfter)
	r.LinesCountAfterLast = configInt(config, "linesCountAfterLast", r.LinesCountAfterLast)
	r.ControlStructures = slvcssKeywords(config, "controlStructures", r.ControlStructures)
	return r
}

func (r SlevomatBlockControlStructureSpacing) Fix(s *tokens.Stream) bool {
	keywords := slvcssCheck(r.ControlStructures, "if", "do", "while", "for", "foreach", "switch", "try", "case", "default")
	return slvRun(s, func(f *slvFile) {
		c := newSlvcss(f)
		c.before, c.after = r.LinesCountBefore, r.LinesCountAfter
		c.beforeFirst = func(int) int { return r.LinesCountBeforeFirst }
		c.afterLast = func(int, int) int { return r.LinesCountAfterLast }
		f.eachToken(func(i int) bool { return keywords[c.keyword(i)] }, func(i int) int {
			if c.isWhilePartOfDo(i) {
				return 0
			}
			c.checkLinesBefore(i)
			c.checkLinesAfter(i)
			return 0
		})
	})
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/ControlStructures/JumpStatementsSpacingSniff.php
//
// SlevomatJumpStatementsSpacing sets the blank lines around jump statements
// (goto, break, continue, return, throw, yield, yield from). The *InCaseOrDefault
// counts are -1 when unset.
type SlevomatJumpStatementsSpacing struct {
	LinesCountBefore, LinesCountBeforeFirst, LinesCountAfter, LinesCountAfterLast int
	LinesCountBeforeWhenFirstInCaseOrDefault                                      int
	LinesCountAfterWhenLastInCaseOrDefault                                        int
	LinesCountAfterWhenLastInLastCaseOrDefault                                    int
	AllowSingleLineYieldStacking                                                  bool
	JumpStatements                                                                []string
}

func (SlevomatJumpStatementsSpacing) Name() string {
	return `SlevomatCodingStandard\Sniffs\ControlStructures\JumpStatementsSpacingSniff`
}

func (SlevomatJumpStatementsSpacing) SourceURL() string {
	return fixer.SourceURLFor(SlevomatJumpStatementsSpacing{}.Name())
}

func (r SlevomatJumpStatementsSpacing) WithConfig(config map[string]any) fixer.Fixer {
	r.LinesCountBefore = configInt(config, "linesCountBefore", r.LinesCountBefore)
	r.LinesCountBeforeFirst = configInt(config, "linesCountBeforeFirst", r.LinesCountBeforeFirst)
	r.LinesCountAfter = configInt(config, "linesCountAfter", r.LinesCountAfter)
	r.LinesCountAfterLast = configInt(config, "linesCountAfterLast", r.LinesCountAfterLast)
	r.LinesCountBeforeWhenFirstInCaseOrDefault = configInt(config, "linesCountBeforeWhenFirstInCaseOrDefault", r.LinesCountBeforeWhenFirstInCaseOrDefault)
	r.LinesCountAfterWhenLastInCaseOrDefault = configInt(config, "linesCountAfterWhenLastInCaseOrDefault", r.LinesCountAfterWhenLastInCaseOrDefault)
	r.LinesCountAfterWhenLastInLastCaseOrDefault = configInt(config, "linesCountAfterWhenLastInLastCaseOrDefault", r.LinesCountAfterWhenLastInLastCaseOrDefault)
	if v, ok := config["allowSingleLineYieldStacking"].(bool); ok {
		r.AllowSingleLineYieldStacking = v
	}
	r.JumpStatements = slvcssKeywords(config, "jumpStatements", r.JumpStatements)
	return r
}

func (r SlevomatJumpStatementsSpacing) Fix(s *tokens.Stream) bool {
	keywords := slvcssCheck(r.JumpStatements, "goto", "break", "continue", "return", "throw", "yield", "yield_from")
	return slvRun(s, func(f *slvFile) {
		c := newSlvcss(f)
		c.before, c.after = r.LinesCountBefore, r.LinesCountAfter
		c.beforeFirst = func(i int) int {
			if r.LinesCountBeforeWhenFirstInCaseOrDefault >= 0 && c.isFirstInCaseOrDefault(i) {
				return r.LinesCountBeforeWhenFirstInCaseOrDefault
			}
			return r.LinesCountBeforeFirst
		}
		c.afterLast = func(_, end int) int {
			if r.LinesCountAfterWhenLastInLastCaseOrDefault >= 0 && c.isLastInLastCaseOrDefault(end) {
				return r.LinesCountAfterWhenLastInLastCaseOrDefault
			}
			if r.LinesCountAfterWhenLastInCaseOrDefault >= 0 && c.isLastInCaseOrDefault(end) {
				return r.LinesCountAfterWhenLastInCaseOrDefault
			}
			return r.LinesCountAfterLast
		}
		f.eachToken(func(i int) bool { return keywords[c.keyword(i)] }, func(i int) int {
			if c.isOneOfYieldSpecialCases(i) {
				return 0
			}
			if (!r.AllowSingleLineYieldStacking || !c.isStackedSingleLineYield(i, true)) && !c.isThrowExpression(i) {
				c.checkLinesBefore(i)
			}
			if (!r.AllowSingleLineYieldStacking || !c.isStackedSingleLineYield(i, false)) && !c.isThrowExpression(i) {
				c.checkLinesAfter(i)
			}
			return 0
		})
	})
}

// slvcssKeywords reads a keyword list option.
func slvcssKeywords(config map[string]any, key string, def []string) []string {
	list, ok := config[key].([]any)
	if !ok {
		return def
	}
	var out []string
	for _, v := range list {
		if s, ok := v.(string); ok {
			out = append(out, strings.TrimSpace(s))
		}
	}
	return out
}

// slvcssCheck is getTokensToCheck: the configured keywords, or every
// supported one when none are configured.
func slvcssCheck(configured []string, supported ...string) map[string]bool {
	if len(configured) == 0 {
		configured = supported
	}
	return phpcsSet(configured...)
}

// slvcss is AbstractControlStructureSpacing over one pass of a file.
type slvcss struct {
	f           *slvFile
	sc          *phpcsScope
	levels      []int
	before      int
	after       int
	beforeFirst func(i int) int
	afterLast   func(i, end int) int
}

var errSlvcssUnsupported = errors.New("unsupported control structure")

func newSlvcss(f *slvFile) *slvcss {
	sc := newPhpcsScope(f)
	_, levels := sc.levelMap()
	return &slvcss{f: f, sc: sc, levels: levels}
}

// code is the PHP_CodeSniffer token code of i, "" out of range.
func (c *slvcss) code(i int) string {
	if i < 0 || i >= len(c.f.toks) {
		return ""
	}
	return c.sc.code[i]
}

// keyword maps i to the sniff's keyword name, "" when it is none.
func (c *slvcss) keyword(i int) string {
	code := c.code(i)
	if strings.HasPrefix(code, "yield") && len(code) > len("yield") {
		return "yield_from"
	}
	return code
}

func (c *slvcss) isCaseOrDefault(i int) bool {
	code := c.code(i)
	return code == "case" || code == "default"
}

// isColon is T_COLON: a ":" that is neither a ternary's (T_INLINE_ELSE) nor
// the one after a goto label (T_GOTO_COLON).
func (c *slvcss) isColon(i int) bool {
	if c.code(i) != ":" {
		return false
	}
	label := c.f.prevEffective(i - 1)
	if label < 0 || c.f.toks[label].kind != token.Ident {
		return true
	}
	switch c.code(c.f.prevEffective(label - 1)) {
	case ";", "{", "}", "open_tag", "":
		return false
	}
	return true
}

func (c *slvcss) line(i int) int { return c.f.toks[i].line }

func (c *slvcss) isComment(i int) bool {
	k := c.f.toks[i].kind
	return k == token.Comment || k == token.DocComment
}

// isInlineComment is TokenHelper::INLINE_COMMENT_TOKEN_CODES: T_COMMENT and
// the phpcs annotations.
func (c *slvcss) isInlineComment(i int) bool {
	return i >= 0 && c.f.toks[i].kind == token.Comment
}

func (c *slvcss) findNext(match func(int) bool, start, end int) int {
	end = min(end, len(c.f.toks))
	for i := max(start, 0); i < end; i++ {
		if match(i) {
			return i
		}
	}
	return -1
}

func (c *slvcss) findPrevious(match func(int) bool, start int) int {
	for i := min(start, len(c.f.toks)-1); i >= 0; i-- {
		if match(i) {
			return i
		}
	}
	return -1
}

func (c *slvcss) checkLinesBefore(i int) {
	f := c.f
	if c.isCaseOrDefault(i) && c.isColon(f.prevEffective(i-1)) {
		return
	}
	start := i
	nonWhitespaceBefore := f.prevNonWhitespace(start - 1)
	if nonWhitespaceBefore < 0 {
		return
	}
	pointerBefore := nonWhitespaceBefore
	pointerToCheckFirst := pointerBefore
	if c.isComment(nonWhitespaceBefore) {
		effectiveBefore := f.prevEffective(pointerBefore - 1)
		if effectiveBefore < 0 {
			return
		}
		if c.line(effectiveBefore) == c.line(nonWhitespaceBefore) {
			pointerToCheckFirst = effectiveBefore
		} else if c.line(nonWhitespaceBefore)+1 == c.line(i) {
			if c.line(effectiveBefore) != c.line(nonWhitespaceBefore) {
				if f.isDocClose(nonWhitespaceBefore) {
					start = f.docBlockOpener(nonWhitespaceBefore)
				} else {
					start = c.multilineCommentStart(nonWhitespaceBefore)
				}
				pointerBefore = f.prevNonWhitespace(start - 1)
			}
			pointerToCheckFirst = pointerBefore
		}
	}

	isFirst := c.code(pointerToCheckFirst) == "{" || c.isColon(pointerToCheckFirst)
	whitespaceBefore := ""
	isOpenTag := f.toks[pointerBefore].kind == token.OpenTag
	if isOpenTag {
		whitespaceBefore += f.toks[pointerBefore].content[min(len("<?php"), len(f.toks[pointerBefore].content)):]
	}
	hasCommentWithLineEndBefore := c.isInlineComment(pointerBefore) && strings.HasSuffix(f.toks[pointerBefore].content, f.eol)
	if hasCommentWithLineEndBefore {
		whitespaceBefore += f.eol
	}
	if pointerBefore+1 != i {
		whitespaceBefore += f.content(pointerBefore+1, start-1)
	}

	required := c.before
	if isFirst {
		required = c.beforeFirst(i)
	}
	if required == strings.Count(whitespaceBefore, f.eol)-1 {
		return
	}

	endOfLineBefore := c.findPrevious(func(j int) bool {
		return f.toks[j].kind == token.Whitespace && f.toks[j].content == f.eol
	}, start-1)

	f.beginChangeset()
	if isOpenTag {
		f.replace(pointerBefore, "<?php")
	}
	if endOfLineBefore >= 0 {
		f.removeBetweenIncluding(pointerBefore+1, endOfLineBefore)
	}
	if c.line(pointerBefore) == c.line(start) {
		f.removeBetweenIncluding(pointerBefore+1, start-1)
	}
	linesToAdd := required
	if hasCommentWithLineEndBefore {
		linesToAdd--
	}
	for range linesToAdd + 1 {
		f.addContent(pointerBefore, f.eol)
	}
	f.endChangeset()
}

func (c *slvcss) checkLinesAfter(i int) {
	f := c.f
	if c.isCaseOrDefault(i) {
		colon := c.findNext(func(j int) bool { return c.code(j) == ":" }, i+1, len(f.toks))
		if colon < 0 {
			return
		}
		if c.isCaseOrDefault(f.nextEffective(colon+1, len(f.toks))) {
			return
		}
	}

	end, err := c.findControlStructureEnd(i)
	if err != nil {
		return
	}
	if after := f.nextEffective(end+1, len(f.toks)); after >= 0 && c.code(after) == ";" {
		end = after
	}
	notWhitespaceAfter := f.nextNonWhitespace(end+1, len(f.toks))
	if notWhitespaceAfter < 0 {
		return
	}

	isEndAfter := func(p int) bool {
		if c.isCaseOrDefault(i) {
			return c.code(p) == "}"
		}
		return c.code(p) == "}" || c.isCaseOrDefault(p)
	}

	isCommentAfterOnSameLine := false
	pointerAfter := notWhitespaceAfter
	if c.isComment(notWhitespaceAfter) {
		if c.line(notWhitespaceAfter) == c.line(end)+1 {
			commentEnd := c.commentEnd(notWhitespaceAfter)
			afterComment := f.nextNonWhitespace(max(commentEnd, 0)+1, len(f.toks))
			if isEndAfter(afterComment) {
				end = commentEnd
				pointerAfter = afterComment
			}
		} else if c.line(notWhitespaceAfter) == c.line(end) {
			isCommentAfterOnSameLine = true
			pointerAfter = f.nextNonWhitespace(notWhitespaceAfter+1, len(f.toks))
		}
	}
	if pointerAfter < 0 || end < 0 {
		return
	}

	required := c.after
	if isEndAfter(pointerAfter) {
		required = c.afterLast(i, end)
	}
	if required == c.line(pointerAfter)-c.line(end)-1 {
		return
	}

	replaceStart := end
	if isCommentAfterOnSameLine {
		replaceStart = notWhitespaceAfter
	}
	lastOnPreviousLine := pointerAfter - 1
	for lastOnPreviousLine > 0 && c.line(lastOnPreviousLine) == c.line(pointerAfter) {
		lastOnPreviousLine--
	}

	f.beginChangeset()
	f.removeBetweenIncluding(replaceStart+1, lastOnPreviousLine)
	if isCommentAfterOnSameLine {
		for range required {
			f.addContent(notWhitespaceAfter, f.eol)
		}
	} else {
		linesToAdd := required
		if strings.HasSuffix(f.toks[end].content, f.eol) {
			linesToAdd--
		}
		for range linesToAdd + 1 {
			f.addContent(end, f.eol)
		}
	}
	f.endChangeset()
}

// multilineCommentStart is CommentHelper::getMultilineCommentStartPointer.
func (c *slvcss) multilineCommentStart(end int) int {
	start := end
	for {
		before := c.findPrevious(c.isInlineComment, start-1)
		if before < 0 || c.line(before)+1 != c.line(start) {
			return start
		}
		start = before
	}
}

var slvcssLineComment = regexp.MustCompile(`^(?://|#(?:[^\[]|$))`)

// commentEnd is CommentHelper::getCommentEndPointer, -1 for null.
func (c *slvcss) commentEnd(start int) int {
	f := c.f
	t := f.toks[start]
	if t.kind == token.DocComment {
		if t.doc != slvDocOpen {
			return -1
		}
		for j := start; j < len(f.toks); j++ {
			if f.toks[j].doc == slvDocClose {
				return j
			}
		}
		return -1
	}
	if slvcssLineComment.MatchString(t.content) {
		return start
	}
	if !strings.HasPrefix(t.content, "/*") {
		return -1
	}
	end := start
	for j := start + 1; j < len(f.toks) && f.toks[j].kind == token.Comment; j++ {
		end = j
	}
	return end
}

func (c *slvcss) closer(i int) (int, error) {
	if i < 0 || c.sc.closer[i] < 0 {
		return 0, errSlvcssUnsupported
	}
	return c.sc.closer[i], nil
}

// findControlStructureEnd ports the method of the same name; an error stands
// for the exceptions (and PHP warnings) process() swallows.
func (c *slvcss) findControlStructureEnd(i int) (int, error) {
	f := c.f
	n := len(f.toks)
	switch code := c.code(i); code {
	case "if":
		end, err := c.closer(i)
		if err != nil {
			return 0, err
		}
		if c.sc.pCloser[i] < 0 {
			return 0, errSlvcssUnsupported
		}
		if p := f.nextEffective(c.sc.pCloser[i]+1, n); p >= 0 && c.code(p) == ":" {
			return 0, errSlvcssUnsupported
		}
		for {
			next := f.nextEffective(end+1, n)
			if next < 0 {
				return end, nil
			}
			switch c.code(next) {
			case "else":
				return c.closer(next)
			case "elseif":
				if end, err = c.closer(next); err != nil {
					return 0, err
				}
			default:
				return end, nil
			}
		}
	case "do":
		end, err := c.closer(i)
		if err != nil {
			return 0, err
		}
		while := c.findNext(func(j int) bool { return c.code(j) == "while" }, end+1, n)
		if while < 0 || c.sc.pCloser[while] < 0 {
			return 0, errSlvcssUnsupported
		}
		return max(c.findNext(func(j int) bool { return c.code(j) == ";" }, c.sc.pCloser[while]+1, n), 0), nil
	case "try":
		end, err := c.closer(i)
		if err != nil {
			return 0, err
		}
		for {
			next := f.nextEffective(end+1, n)
			if next < 0 || c.code(next) != "catch" && c.code(next) != "finally" {
				return end, nil
			}
			if end, err = c.closer(next); err != nil {
				return 0, err
			}
		}
	case "while", "for", "foreach", "switch":
		return c.closer(i)
	case "case", "default":
		isSwitch := func(j int) bool { return c.code(j) == "switch" }
		sw := c.findPrevious(isSwitch, i-1)
		swEnd, err := c.closer(sw)
		if err != nil {
			return 0, err
		}
		for p := i + 1; p < swEnd; p++ {
			if !c.isCaseOrDefault(p) || c.findPrevious(isSwitch, p-1) != sw {
				continue
			}
			before := f.prevNonWhitespace(p - 1)
			if c.isComment(before) && c.line(before)+1 == c.line(p) {
				before = f.prevNonWhitespace(before - 1)
			}
			return before, nil
		}
		return f.prevNonWhitespace(swEnd - 1), nil
	}

	isSemicolon := func(j int) bool { return c.code(j) == ";" }
	next := c.findNext(func(j int) bool {
		return isSemicolon(j) || c.code(j) == "class" && f.isAnonClass(j) || f.closureIsClosure(j) || c.code(j) == "fn" || f.isShortArrayOpen(j)
	}, i+1, n)
	if next < 0 {
		return 0, errSlvcssUnsupported
	}
	if isSemicolon(next) {
		return next, nil
	}
	scopeCloser := f.toks[next].closer
	if !f.isShortArrayOpen(next) {
		var err error
		if scopeCloser, err = c.closer(next); err != nil {
			return 0, err
		}
	}
	if isSemicolon(scopeCloser) {
		return scopeCloser, nil
	}
	next = c.findNext(isSemicolon, scopeCloser+1, n)
	for next >= 0 && c.levels[i] != c.levels[next] {
		next = c.findNext(isSemicolon, next+1, n)
	}
	if next < 0 {
		return 0, errSlvcssUnsupported
	}
	return next, nil
}

func (c *slvcss) isWhilePartOfDo(i int) bool {
	before := c.f.prevEffective(i - 1)
	return c.code(i) == "while" && c.code(before) == "}" && c.sc.cond[before] >= 0 && c.code(c.sc.cond[before]) == "do"
}

var slvcssAssignment = phpcsSet("=", "&=", "|=", ".=", "/=", "-=", "**=", "%=", "*=", "+=", "^=", "=>", "<<=", ">>=", "??=")

func (c *slvcss) isYield(i int) bool {
	k := c.keyword(i)
	return k == "yield" || k == "yield_from"
}

func (c *slvcss) isOneOfYieldSpecialCases(i int) bool {
	if !c.isYield(i) {
		return false
	}
	if c.sc.nested[i] >= 0 {
		return true
	}
	before := c.f.prevEffective(i - 1)
	return slvcssAssignment[c.code(before)] || c.code(before) == "return"
}

func (c *slvcss) isStackedSingleLineYield(i int, previous bool) bool {
	if !c.isYield(i) {
		return false
	}
	adjoining := c.findNext(c.isYield, i+1, len(c.f.toks))
	if previous {
		adjoining = c.findPrevious(c.isYield, i-1)
	}
	if adjoining < 0 {
		return false
	}
	d := c.line(adjoining) - c.line(i)
	return d == 1 || d == -1
}

func (c *slvcss) isThrowExpression(i int) bool {
	if c.code(i) != "throw" {
		return false
	}
	switch c.code(c.f.prevEffective(i - 1)) {
	case ";", ":", "{", "}", "open_tag":
		return false
	}
	return true
}

func (c *slvcss) isFirstInCaseOrDefault(i int) bool {
	previous := c.f.prevEffective(i - 1)
	if !c.isColon(previous) {
		return false
	}
	return c.isCaseOrDefault(c.f.firstNonWhitespaceOnLine(previous))
}

func (c *slvcss) isLastInCaseOrDefault(end int) bool {
	next := c.f.nextEffective(end+1, len(c.f.toks))
	if c.isCaseOrDefault(next) {
		return true
	}
	return c.code(next) == "}" && c.sc.cond[next] >= 0 && c.code(c.sc.cond[next]) == "switch"
}

func (c *slvcss) isLastInLastCaseOrDefault(end int) bool {
	return c.isLastInCaseOrDefault(end) && !c.isCaseOrDefault(c.f.nextEffective(end+1, len(c.f.toks)))
}
