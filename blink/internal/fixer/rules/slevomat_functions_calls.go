package rules

import (
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/Functions/RequireSingleLineCallSniff.php
//
// SlevomatRequireSingleLineCall joins a multi-line call onto one line when it
// fits maxLineLength (0: always). Calls with comments, heredocs or multi-line
// strings are left alone, and with ignoreWithComplexParameter also calls
// holding a closure, an arrow function, a short array or another call.
type SlevomatRequireSingleLineCall struct {
	maxLineLength              *int
	keepComplexParameterFormat bool // ignoreWithComplexParameter false
}

func (SlevomatRequireSingleLineCall) Name() string {
	return `SlevomatCodingStandard\Sniffs\Functions\RequireSingleLineCallSniff`
}

func (SlevomatRequireSingleLineCall) SourceURL() string {
	return fixer.SourceURLFor(SlevomatRequireSingleLineCall{}.Name())
}

func (r SlevomatRequireSingleLineCall) WithConfig(config map[string]any) fixer.Fixer {
	if _, ok := config["maxLineLength"]; ok {
		r.maxLineLength = new(configInt(config, "maxLineLength", 120))
	}
	if v, ok := config["ignoreWithComplexParameter"].(bool); ok {
		r.keepComplexParameterFormat = !v
	}
	return r
}

func (r SlevomatRequireSingleLineCall) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		f.eachToken(f.lineCallRegistered, func(i int) int {
			r.process(f, i)
			return 0
		})
	})
}

func (r SlevomatRequireSingleLineCall) maxLength() int {
	if r.maxLineLength == nil {
		return 120
	}
	return *r.maxLineLength
}

func (r SlevomatRequireSingleLineCall) process(f *slvFile, name int) {
	if !f.lineCallIsCall(name) {
		return
	}
	open := f.nextEffective(name+1, len(f.toks))
	closeIdx := f.toks[open].closer
	if closeIdx < 0 || f.toks[open].line == f.toks[closeIdx].line {
		return
	}
	if f.lineCallSingleSkipped(name) {
		return
	}
	for i := open + 1; i < closeIdx; i++ {
		t := f.toks[i]
		if t.kind == token.Comment || t.heredoc {
			return
		}
	}
	for i := open + 1; i < closeIdx; i++ {
		if f.toks[i].kind == token.String && strings.Contains(f.toks[i].content, f.eol) {
			return
		}
	}
	if !r.keepComplexParameterFormat {
		for i := open + 1; i < closeIdx; i++ {
			if f.isKeyword(i, "fn") || f.isKeyword(i, "function") && f.isClosure(i) || f.isShortArrayOpen(i) {
				return
			}
		}
		for i := open + 1; i < closeIdx; i++ {
			if !f.lineCallIsName(i) {
				continue
			}
			if after := f.nextEffective(i+1, len(f.toks)); f.is(after, "(") {
				return
			}
		}
	}
	lineStart := f.lineCallLineStart(open)
	call := f.lineCallCall(open, closeIdx)
	lineEnd := f.lineCallLineEnd(closeIdx)
	if limit := r.maxLength(); limit != 0 && len(lineStart+call+lineEnd) > limit {
		return
	}
	f.beginChangeset()
	f.add(open, call)
	for i := open + 1; i < closeIdx; i++ {
		f.replaceToken(i, "")
	}
	f.endChangeset()
}

// lineCallSingleSkipped is RequireSingleLineCallSniff::shouldBeSkipped: a call
// inside another call's parentheses (innermost first, up to the enclosing
// named function) is left alone.
func (f *slvFile) lineCallSingleSkipped(name int) bool {
	var pointers []int
	for e := f.enclosing(name); e >= 0; e = f.enclosing(e) {
		if f.is(e, "(") {
			pointers = append(pointers, e)
		}
		if f.is(e, "{") && f.closesScope(f.toks[e].closer) {
			if owner := f.braceOwner(e); owner >= 0 && f.isKeyword(owner, "function") && !f.isClosure(owner) {
				pointers = append(pointers, owner)
			}
		}
	}
	slices.SortFunc(pointers, func(a, b int) int { return b - a })
	for _, p := range pointers {
		if f.isKeyword(p, "function") {
			return false
		}
		before := f.prevEffective(p - 1)
		if before < 0 || !f.lineCallIsPlainString(before) {
			continue
		}
		return true
	}
	return false
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/Functions/RequireMultiLineCallSniff.php
//
// SlevomatRequireMultiLineCall puts every parameter of a call on its own line
// when the call line reaches minLineLength (121 by default, 0: always) or the
// call has minParametersCount parameters. Calls whose name matches an
// excludedCallPatterns regex are skipped.
type SlevomatRequireMultiLineCall struct {
	minLineLength      *int
	minParametersCount *int
	excluded           []*regexp.Regexp
	invalid            bool
}

func (SlevomatRequireMultiLineCall) Name() string {
	return `SlevomatCodingStandard\Sniffs\Functions\RequireMultiLineCallSniff`
}

func (SlevomatRequireMultiLineCall) SourceURL() string {
	return fixer.SourceURLFor(SlevomatRequireMultiLineCall{}.Name())
}

func (r SlevomatRequireMultiLineCall) WithConfig(config map[string]any) fixer.Fixer {
	if _, ok := config["minLineLength"]; ok && config["minLineLength"] != nil {
		r.minLineLength = new(configInt(config, "minLineLength", 0))
	}
	if _, ok := config["minParametersCount"]; ok && config["minParametersCount"] != nil {
		r.minParametersCount = new(configInt(config, "minParametersCount", 0))
	}
	if patterns, ok := config["excludedCallPatterns"].([]any); ok {
		r.excluded = nil
		for _, p := range patterns {
			v, _ := p.(string)
			if v = strings.TrimSpace(v); v == "" {
				continue
			}
			re := slvlCompilePattern(v)
			if re == nil {
				// upstream throws on an invalid pattern, so ECS fixes nothing
				r.invalid = true
				continue
			}
			r.excluded = append(r.excluded, re)
		}
	}
	return r
}

func (r SlevomatRequireMultiLineCall) Fix(s *tokens.Stream) bool {
	// upstream throws when both are set, so ECS fixes nothing
	if r.invalid || r.minLineLength != nil && r.minParametersCount != nil {
		return false
	}
	if r.minLineLength == nil && r.minParametersCount == nil {
		r.minLineLength = new(121)
	}
	return slvRun(s, func(f *slvFile) {
		f.eachToken(f.lineCallRegistered, func(i int) int {
			r.process(f, i)
			return 0
		})
	})
}

func (r SlevomatRequireMultiLineCall) process(f *slvFile, name int) {
	if !f.lineCallIsCall(name) {
		return
	}
	open := f.nextEffective(name+1, len(f.toks))
	closeIdx := f.toks[open].closer
	if closeIdx < 0 {
		return
	}
	first := f.nextEffective(open+1, len(f.toks))
	if first == closeIdx {
		return
	}
	params := []int{first}
	level := 0
	for i := open + 1; i < closeIdx; i++ {
		switch {
		case f.is(i, "(") || f.isShortArrayOpen(i):
			level++
		case f.is(i, ")") || f.is(i, "]") && f.toks[i].closer >= 0 && f.isShortArrayOpen(f.toks[i].closer):
			level--
		case f.is(i, ",") && level == 0:
			if p := f.nextEffective(i+1, closeIdx); p >= 0 {
				params = append(params, p)
			}
		}
	}
	lines := map[int]bool{f.toks[open].line: true, f.toks[closeIdx].line: true}
	for _, p := range params {
		lines[f.toks[p].line] = true
	}
	if len(lines)-2 >= len(params) {
		return
	}
	if f.lineCallMultiSkipped(name, closeIdx) {
		return
	}

	lineStart := f.lineCallLineStart(open)
	var lineEnd string
	var lineLength int
	if f.toks[closeIdx].line == f.toks[name].line {
		call := f.lineCallCall(open, closeIdx)
		lineEnd = f.lineCallLineEnd(closeIdx)
		lineLength = utf8.RuneCountInString(lineStart + call + lineEnd)
	} else {
		lineEnd = f.lineCallLineEnd(open + 1)
		lineLength = utf8.RuneCountInString(lineStart + lineEnd)
	}
	indentation := f.indentation(f.firstNonWhitespaceOnLine(name))
	oneIndentation := strings.Repeat(" ", slvTabWidth)
	if !r.shouldReport(lineLength, lineStart, lineEnd, len(params), len(oneIndentation)) {
		return
	}
	callName := strings.TrimLeft(f.toks[name].content, "\\")
	for _, re := range r.excluded {
		if re.MatchString(callName) {
			return
		}
	}

	paramIndentation := indentation + oneIndentation
	f.beginChangeset()
	for i := open + 1; i < closeIdx; i++ {
		switch {
		case slices.Contains(params, i):
			f.lineCallRemoveWhitespaceBefore(i)
			f.addBefore(i, f.eol+paramIndentation)
		case f.toks[i].content == f.eol:
			f.add(i, oneIndentation)
		default:
			// a no-op change that conflicts with inner calls, so they are fixed in the next pass
			f.replace(i, f.toks[i].content)
		}
	}
	f.addBefore(closeIdx, f.eol+indentation)
	f.endChangeset()
}

func (r SlevomatRequireMultiLineCall) shouldReport(lineLength int, lineStart, lineEnd string, paramsCount, indentationLength int) bool {
	if r.minLineLength != nil && *r.minLineLength == 0 {
		return true
	}
	if r.minLineLength != nil && lineLength < *r.minLineLength {
		return false
	}
	if r.minParametersCount != nil && paramsCount < *r.minParametersCount {
		return false
	}
	if paramsCount > 1 {
		return true
	}
	return len(strings.TrimSpace(lineStart)+strings.TrimSpace(lineEnd)) > indentationLength
}

// lineCallMultiSkipped is RequireMultiLineCallSniff::shouldBeSkipped: a call
// that is a parameter of a call started earlier on its line, or followed on its
// closing line by another call with parameters closing on the call's line.
func (f *slvFile) lineCallMultiSkipped(name, closeIdx int) bool {
	for i := f.firstNonWhitespaceOnLine(name); i < name; i++ {
		if !f.lineCallIsName(i) {
			continue
		}
		after := f.nextEffective(i+1, len(f.toks))
		if f.is(after, "(") && f.toks[after].closer > name {
			return true
		}
	}
	last := f.lastOnLine(closeIdx)
	for i := closeIdx + 1; i <= last; i++ {
		if !f.lineCallIsName(i) {
			continue
		}
		after := f.nextEffective(i+1, len(f.toks))
		if after < 0 || !f.is(after, "(") || f.toks[after].closer < 0 {
			continue
		}
		c := f.toks[after].closer
		if f.toks[c].line == f.toks[name].line && c != f.nextEffective(after+1, len(f.toks)) {
			return true
		}
	}
	return false
}

// lineCallRemoveWhitespaceBefore is FixerHelper::removeWhitespaceBefore.
func (f *slvFile) lineCallRemoveWhitespaceBefore(i int) {
	for j := i - 1; j > 0; j-- {
		v := f.tokenContent(j)
		if v == "" || strings.Trim(v, " \t\n\r\v\f") != "" {
			break
		}
		f.replaceToken(j, "")
	}
}

// slvlNotNames are the identifiers PHP_CodeSniffer gives a token of their own
// rather than T_STRING.
var slvlNotNames = map[string]bool{
	"true": true, "false": true, "null": true, "exit": true, "die": true,
	"eval": true, "__halt_compiler": true, "self": true, "parent": true,
	"__line__": true, "__file__": true, "__dir__": true, "__function__": true,
	"__class__": true, "__trait__": true, "__method__": true, "__namespace__": true,
	"__property__": true,
}

// lineCallIsName tells a TokenHelper::NAME_TOKEN_CODES token (T_STRING or a
// qualified name).
func (f *slvFile) lineCallIsName(i int) bool {
	t := f.toks[i]
	if t.kind != token.Ident {
		return false
	}
	if !slvlNotNames[strings.ToLower(t.orig)] {
		return true
	}
	// a member or declared name is T_STRING whatever it is called ($o->eval())
	p := f.prevEffective(i - 1)
	return f.is(p, "->") || f.is(p, "?->") || f.is(p, "::") || f.isKeyword(p, "function") || f.isKeyword(p, "const")
}

// lineCallIsPlainString tells a T_STRING from a qualified name.
func (f *slvFile) lineCallIsPlainString(i int) bool {
	return f.lineCallIsName(i) && !strings.Contains(f.toks[i].orig, "\\")
}

// lineCallRegistered is AbstractLineCall::register: names plus self, parent
// and static.
func (f *slvFile) lineCallRegistered(i int) bool {
	if f.lineCallIsName(i) || f.isKeyword(i, "static") {
		return true
	}
	t := f.toks[i]
	if t.kind != token.Ident {
		return false
	}
	v := strings.ToLower(t.orig)
	return v == "self" || v == "parent"
}

// lineCallIsCall is AbstractLineCall::isCall.
func (f *slvFile) lineCallIsCall(name int) bool {
	next := f.nextEffective(name+1, len(f.toks))
	if !f.is(next, "(") {
		return false
	}
	prev := f.prevEffective(name - 1)
	if f.is(prev, "&") {
		prev = f.prevEffective(prev - 1)
	}
	return !f.isKeyword(prev, "function")
}

// lineCallLineStart is AbstractLineCall::getLineStart.
func (f *slvFile) lineCallLineStart(i int) string {
	return f.content(f.firstOnLine(i), i)
}

// lineCallLineEnd is AbstractLineCall::getLineEnd.
func (f *slvFile) lineCallLineEnd(i int) string {
	next := f.firstOnNextLine(i)
	if next < 0 {
		return ""
	}
	return strings.TrimRight(f.content(i, next-1), " \t\n\r\x00\x0B")
}

// lineCallCall is AbstractLineCall::getCall: the parameters joined on one line.
func (f *slvFile) lineCallCall(open, closeIdx int) string {
	end := closeIdx
	if before := f.prevEffective(closeIdx - 1); f.is(before, ",") {
		end = before
	}
	var b strings.Builder
	for i := open + 1; i < end; i++ {
		if f.is(i, ",") {
			if next := f.nextEffective(i+1, len(f.toks)); f.is(next, ")") {
				i = next - 1
				continue
			}
		}
		t := f.toks[i]
		if t.kind == token.Whitespace {
			if t.content == f.eol {
				if f.is(i-1, ",") {
					b.WriteString(" ")
				}
				continue
			}
			if t.column == 1 {
				continue
			}
		}
		b.WriteString(t.content)
	}
	return strings.Trim(b.String(), " \t\n\r\x00\x0B")
}

// slvlCompilePattern turns a PHP regex ("/x/i") into a Go one, or nil when it
// has no valid delimiters or uses what RE2 lacks.
func slvlCompilePattern(p string) *regexp.Regexp {
	if len(p) < 2 {
		return nil
	}
	open := p[0]
	closeDelim := open
	switch open {
	case '(':
		closeDelim = ')'
	case '[':
		closeDelim = ']'
	case '{':
		closeDelim = '}'
	case '<':
		closeDelim = '>'
	}
	end := strings.LastIndexByte(p, closeDelim)
	if end <= 0 {
		return nil
	}
	body, flags := p[1:end], p[end+1:]
	var prefix strings.Builder
	for _, c := range flags {
		switch c {
		case 'i', 'm', 's':
			prefix.WriteString(string(c))
		case 'u', 'D':
		default:
			return nil
		}
	}
	if prefix.String() != "" {
		body = "(?" + prefix.String() + ")" + body
	}
	re, err := regexp.Compile(body)
	if err != nil {
		return nil
	}
	return re
}
