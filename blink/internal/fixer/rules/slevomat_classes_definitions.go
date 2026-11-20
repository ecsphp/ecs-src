package rules

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/lexer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// clEMethodPatterns are a method signature sniff's normalized method patterns.
type clEMethodPatterns struct {
	list []string
}

func clEPatterns(config map[string]any, key string) clEMethodPatterns {
	list, _ := slvStringList(config[key])
	return clEMethodPatterns{list: list}
}

// matches is isMethodNameInPatterns; abort is true where upstream throws on an
// invalid pattern.
func (p clEMethodPatterns) matches(name string) (match, abort bool) {
	for _, pattern := range p.list {
		re := slvlCompilePattern(pattern)
		if re == nil {
			return false, true
		}
		if re.MatchString(name) {
			return true, false
		}
	}
	return false, false
}

// clEMethodSkipped applies the included and excluded method patterns.
func clEMethodSkipped(included, excluded clEMethodPatterns, name string) (skip, abort bool) {
	if len(included.list) != 0 {
		match, abort := included.matches(name)
		if abort {
			return true, true
		}
		if !match {
			return true, false
		}
	}
	if len(excluded.list) != 0 {
		match, abort := excluded.matches(name)
		if abort || match {
			return true, abort
		}
	}
	return false, false
}

// clERunAbortable is slvRun for a sniff that can throw: a thrown file is left
// unchanged, as ECS leaves it.
func clERunAbortable(s *tokens.Stream, process func(f *slvFile, abort *bool)) bool {
	original := s.Render()
	aborted := false
	changed := slvRun(s, func(f *slvFile) {
		if aborted {
			return
		}
		process(f, &aborted)
		if aborted {
			f.inConflict = true
		}
	})
	if aborted {
		if changed {
			s.ReplaceRange(0, s.Len()-1, lexer.Lex(original))
		}
		return false
	}
	return changed
}

// clESignaturePointers is AbstractMethodSignature::getSignatureStartAndEndPointers.
func (f *slvFile) clESignaturePointers(fn int) (start, end int, ok bool) {
	start = f.firstOnLine(fn)
	after := -1
	for i := fn + 1; i < len(f.toks); i++ {
		if f.is(i, "{") || f.is(i, ";") {
			after = i
			break
		}
	}
	if after < 0 {
		return 0, 0, false
	}
	if f.is(after, ";") {
		return start, after, true
	}
	return start, f.prevEffective(after - 1), true
}

// clESignature is AbstractMethodSignature::getSignature.
func (f *slvFile) clESignature(start, end int) string {
	signature := f.content(start, end)
	signature = regexp.MustCompile(regexp.QuoteMeta(f.eol)+`[ \t]*`).ReplaceAllString(signature, " ")
	signature = strings.NewReplacer("( ", "(", " )", ")").Replace(signature)
	return strings.TrimRight(signature, " \t\n\r\x00\x0B")
}

// clEParam is the part of a File::getMethodParameters entry the signature
// sniffs read.
type clEParam struct {
	token         int
	hasAttributes bool
	promoted      bool
}

// clEMethodParameters is File::getMethodParameters.
func (f *slvFile) clEMethodParameters(fn int) []clEParam {
	open := f.functionParenOpen(fn)
	if open < 0 {
		return nil
	}
	closeIdx := f.toks[open].closer
	var params []clEParam
	cur := clEParam{token: -1}
	for i := open + 1; i <= closeIdx; i++ {
		t := f.toks[i]
		switch {
		case t.kind == token.AttributeStart && t.closer >= 0:
			cur.hasAttributes = true
			i = t.closer
		case i != closeIdx && (f.is(i, "(") || f.is(i, "[")) && t.closer > i && !f.slvmIsTypeParen(i):
			i = t.closer
		case f.isVariable(i):
			cur.token = i
		case t.kind == token.Keyword && clEIsPromotionModifier(t.orig):
			cur.promoted = true
		case i == closeIdx || f.is(i, ","):
			if cur.token < 0 {
				continue
			}
			params = append(params, cur)
			cur = clEParam{token: -1}
		}
	}
	return params
}

func clEIsPromotionModifier(v string) bool {
	switch strings.ToLower(v) {
	case "public", "protected", "private", "readonly", "public(set)", "protected(set)", "private(set)":
		return true
	}
	return false
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/Classes/RequireMultiLineMethodSignatureSniff.php
//
// SlevomatRequireMultiLineMethodSignature puts every parameter of a one-line
// method signature on its own line when the signature reaches minLineLength
// (121 by default, 0: always) or the method has minParametersCount parameters.
type SlevomatRequireMultiLineMethodSignature struct {
	minLineLength                *int
	minParametersCount           *int
	withPromotedProperties       bool
	withParametersWithAttributes bool
	included, excluded           clEMethodPatterns
}

func (SlevomatRequireMultiLineMethodSignature) Name() string {
	return `SlevomatCodingStandard\Sniffs\Classes\RequireMultiLineMethodSignatureSniff`
}

func (SlevomatRequireMultiLineMethodSignature) SourceURL() string {
	return fixer.SourceURLFor(SlevomatRequireMultiLineMethodSignature{}.Name())
}

func (r SlevomatRequireMultiLineMethodSignature) WithConfig(config map[string]any) fixer.Fixer {
	if config["minLineLength"] != nil {
		r.minLineLength = new(configInt(config, "minLineLength", 0))
	}
	if config["minParametersCount"] != nil {
		r.minParametersCount = new(configInt(config, "minParametersCount", 0))
	}
	r.withPromotedProperties = thBool(config, "withPromotedProperties", false)
	r.withParametersWithAttributes = thBool(config, "withParametersWithAttributes", false)
	r.included = clEPatterns(config, "includedMethodPatterns")
	r.excluded = clEPatterns(config, "excludedMethodPatterns")
	return r
}

func (r SlevomatRequireMultiLineMethodSignature) Fix(s *tokens.Stream) bool {
	// upstream throws when both are set, so ECS fixes nothing
	if r.minLineLength != nil && r.minParametersCount != nil {
		return false
	}
	if r.minLineLength == nil && r.minParametersCount == nil {
		r.minLineLength = new(121)
	}
	return clERunAbortable(s, func(f *slvFile, abort *bool) {
		f.eachToken(func(i int) bool { return f.isKeyword(i, "function") && !f.isClosure(i) }, func(i int) int {
			if !*abort {
				r.process(f, i, abort)
			}
			return 0
		})
	})
}

func (r SlevomatRequireMultiLineMethodSignature) process(f *slvFile, fn int, abort *bool) {
	if !f.thdIsMethod(fn) {
		return
	}
	start, end, ok := f.clESignaturePointers(fn)
	if !ok || f.toks[start].line < f.toks[end].line {
		return
	}
	params := f.clEMethodParameters(fn)
	if len(params) == 0 {
		return
	}
	signature := f.clESignature(start, end)
	skip, aborted := clEMethodSkipped(r.included, r.excluded, f.thdFunctionName(fn))
	if aborted {
		*abort = true
		return
	}
	if skip {
		return
	}
	forceSplit := r.minLineLength != nil && *r.minLineLength == 0
	for _, p := range params {
		if r.withPromotedProperties && p.promoted || r.withParametersWithAttributes && p.hasAttributes {
			forceSplit = true
			break
		}
	}
	if !forceSplit {
		if r.minLineLength != nil && len(signature) < *r.minLineLength {
			return
		}
		if r.minParametersCount != nil && len(params) < *r.minParametersCount {
			return
		}
	}
	open := f.functionParenOpen(fn)
	indentation := f.toks[start].content
	f.beginChangeset()
	for _, p := range params {
		before := open
		for i := p.token - 1; i > open; i-- {
			if f.is(i, ",") {
				before = i
				break
			}
		}
		f.add(before, f.eol+indentation+strings.Repeat(" ", slvTabWidth))
		f.slvlcRemoveWhitespaceAfter(before)
	}
	f.addBefore(f.toks[open].closer, f.eol+indentation)
	f.endChangeset()
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/Classes/RequireSingleLineMethodSignatureSniff.php
//
// SlevomatRequireSingleLineMethodSignature joins a multi-line method signature
// onto one line when it fits maxLineLength (120 by default, 0: always).
type SlevomatRequireSingleLineMethodSignature struct {
	maxLineLength      *int
	included, excluded clEMethodPatterns
}

func (SlevomatRequireSingleLineMethodSignature) Name() string {
	return `SlevomatCodingStandard\Sniffs\Classes\RequireSingleLineMethodSignatureSniff`
}

func (SlevomatRequireSingleLineMethodSignature) SourceURL() string {
	return fixer.SourceURLFor(SlevomatRequireSingleLineMethodSignature{}.Name())
}

func (r SlevomatRequireSingleLineMethodSignature) WithConfig(config map[string]any) fixer.Fixer {
	if _, ok := config["maxLineLength"]; ok {
		r.maxLineLength = new(configInt(config, "maxLineLength", 120))
	}
	r.included = clEPatterns(config, "includedMethodPatterns")
	r.excluded = clEPatterns(config, "excludedMethodPatterns")
	return r
}

func (r SlevomatRequireSingleLineMethodSignature) Fix(s *tokens.Stream) bool {
	maxLength := 120
	if r.maxLineLength != nil {
		maxLength = *r.maxLineLength
	}
	return clERunAbortable(s, func(f *slvFile, abort *bool) {
		f.eachToken(func(i int) bool { return f.isKeyword(i, "function") && !f.isClosure(i) }, func(fn int) int {
			if *abort || !f.thdIsMethod(fn) {
				return 0
			}
			start, end, ok := f.clESignaturePointers(fn)
			if !ok || f.toks[start].line == f.toks[end].line {
				return 0
			}
			signature := f.clESignature(start, end)
			skip, aborted := clEMethodSkipped(r.included, r.excluded, f.thdFunctionName(fn))
			if aborted {
				*abort = true
				return 0
			}
			if skip || maxLength != 0 && len(signature) > maxLength {
				return 0
			}
			f.beginChangeset()
			f.change(start, end, signature)
			f.endChangeset()
			return 0
		})
	})
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/Classes/ModernClassNameReferenceSniff.php
//
// SlevomatModernClassNameReference replaces __CLASS__, get_class(),
// get_parent_class() and get_called_class() with a "::class" reference.
// enableOnObjects (on by default, as ECS runs PHP 8) also turns
// get_class($object) into $object::class.
type SlevomatModernClassNameReference struct {
	disableOnObjects bool
}

func (SlevomatModernClassNameReference) Name() string {
	return `SlevomatCodingStandard\Sniffs\Classes\ModernClassNameReferenceSniff`
}

func (SlevomatModernClassNameReference) SourceURL() string {
	return fixer.SourceURLFor(SlevomatModernClassNameReference{}.Name())
}

func (r SlevomatModernClassNameReference) WithConfig(config map[string]any) fixer.Fixer {
	r.disableOnObjects = !thBool(config, "enableOnObjects", true)
	return r
}

// clEIsClassC tells T_CLASS_C, the __CLASS__ magic constant.
func (f *slvFile) clEIsClassC(i int) bool {
	return f.toks[i].kind == token.Ident && strings.EqualFold(f.toks[i].orig, "__CLASS__") && !f.isName(i)
}

func (r SlevomatModernClassNameReference) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		f.eachToken(func(i int) bool { return f.clEIsClassC(i) || f.isName(i) }, func(i int) int {
			if f.clEIsClassC(i) {
				f.beginChangeset()
				f.replace(i, "self::class")
				f.endChangeset()
				return 0
			}
			r.functionCall(f, i)
			return 0
		})
	})
}

func (r SlevomatModernClassNameReference) functionCall(f *slvFile, fn int) {
	name := strings.TrimLeft(strings.ToLower(f.toks[fn].content), `\`)
	if name != "get_class" && name != "get_parent_class" && name != "get_called_class" {
		return
	}
	open := f.nextEffective(fn+1, len(f.toks))
	if !f.is(open, "(") || f.toks[open].closer < 0 {
		return
	}
	closeIdx := f.toks[open].closer
	if prev := f.prevEffective(fn - 1); f.is(prev, "->") || f.is(prev, "::") || f.isKeyword(prev, "function") {
		return
	}
	param := f.nextEffective(open+1, closeIdx)
	isObject := func() bool {
		return param >= 0 && f.isVariable(param) && f.nextEffective(param+1, len(f.toks)) == closeIdx
	}
	isThis := func() bool {
		return isObject() && strings.ToLower(f.toks[param].content) == "$this"
	}
	var fixed string
	switch name {
	case "get_class":
		switch {
		case param < 0:
			fixed = "self::class"
		case isThis():
			fixed = "static::class"
		case !r.disableOnObjects && isObject():
			fixed = f.toks[param].content + "::class"
		default:
			return
		}
	case "get_parent_class":
		if param >= 0 {
			if !isThis() {
				return
			}
			class := f.clEClassPointer(fn)
			if class < 0 || !f.isKeyword(f.prevEffective(class-1), "final") {
				return
			}
		}
		fixed = "parent::class"
	default:
		fixed = "static::class"
	}
	f.beginChangeset()
	f.change(fn, closeIdx, fixed)
	f.endChangeset()
}

// clEClassPointer is FunctionHelper::findClassPointer: the innermost class-like
// scope holding i, or -1.
func (f *slvFile) clEClassPointer(i int) int {
	for o := f.enclosingScope(i); o >= 0; o = f.enclosingScope(o) {
		if owner := f.braceOwner(o); owner >= 0 && f.isOOScopeOwner(owner) {
			return owner
		}
	}
	return -1
}

// clEIsConst tells T_CONST: a "const" keyword that is not part of "use const"
// or a member name.
func (f *slvFile) clEIsConst(i int) bool {
	if !f.isKeyword(i, "const") {
		return false
	}
	p := f.prevEffective(i - 1)
	return !f.isKeyword(p, "use") && !f.isMemberOperator(p) && !f.is(p, "?->") && !f.isKeyword(p, "function")
}

// clEOrigContent is File::getTokensAsString with origContent: the source text
// of the tokens start..end inclusive.
func (f *slvFile) clEOrigContent(start, end int) string {
	var b strings.Builder
	for i := start; i <= end && i < len(f.toks); i++ {
		b.WriteString(f.toks[i].orig)
	}
	return b.String()
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/Classes/DisallowMultiConstantDefinitionSniff.php
//
// SlevomatDisallowMultiConstantDefinition splits "const A = 1, B = 2;" into
// one definition per constant, repeating the modifiers, type and doc comment.
type SlevomatDisallowMultiConstantDefinition struct{}

func (SlevomatDisallowMultiConstantDefinition) Name() string {
	return `SlevomatCodingStandard\Sniffs\Classes\DisallowMultiConstantDefinitionSniff`
}

func (SlevomatDisallowMultiConstantDefinition) SourceURL() string {
	return fixer.SourceURLFor(SlevomatDisallowMultiConstantDefinition{}.Name())
}

func (SlevomatDisallowMultiConstantDefinition) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		f.eachToken(f.clEIsConst, func(i int) int {
			f.clEMultiConstant(i)
			return 0
		})
	})
}

func (f *slvFile) clEFindNext(start, end int, match func(i int) bool) int {
	end = min(end, len(f.toks))
	for i := start; i < end; i++ {
		if match(i) {
			return i
		}
	}
	return -1
}

func (f *slvFile) clEMultiConstant(constant int) {
	semicolon := f.clEFindNext(constant+1, len(f.toks), func(i int) bool { return f.is(i, ";") })
	if semicolon < 0 {
		return
	}
	var commas []int
	for next := constant; ; {
		next = f.clEFindNext(next+1, semicolon, func(i int) bool {
			return f.is(i, ",") || f.isShortArrayOpen(i) || f.is(i, "(")
		})
		if next < 0 {
			break
		}
		if !f.is(next, ",") {
			if f.toks[next].closer < 0 {
				return
			}
			next = f.toks[next].closer
			continue
		}
		commas = append(commas, next)
	}
	if len(commas) == 0 {
		return
	}
	firstModifier := constant
	modifiers := ""
	for p := f.prevEffective(constant - 1); p >= 0 && (f.isKeyword(p, "final") || f.isKeyword(p, "public") || f.isKeyword(p, "protected") || f.isKeyword(p, "private")); p = f.prevEffective(p - 1) {
		firstModifier = p
		modifiers = fmt.Sprintf("%s %s", f.toks[p].content, modifiers)
	}
	afterConst := f.nextEffective(constant+1, len(f.toks))
	beforeSemicolon := f.prevEffective(semicolon - 1)
	equal := f.clEFindNext(afterConst+1, semicolon, func(i int) bool { return f.is(i, "=") })
	if equal < 0 {
		return
	}
	name := f.prevEffective(equal - 1)
	typeHint := ""
	if name != afterConst {
		typeHint = f.content(afterConst, f.prevEffective(name-1)) + " "
	}
	first := f.firstOnLine(firstModifier)
	indentation := f.clEOrigContent(first, firstModifier-1)
	docComment := ""
	hasDoc := false
	if doc := f.docOpenBefore(constant); doc >= 0 {
		hasDoc = true
		docComment = strings.Trim(f.clEOrigContent(doc, f.docCloser(doc)), " \t\n\r\x00\x0B")
	}
	type around struct{ before, after int }
	data := map[int]around{}
	for _, c := range commas {
		data[c] = around{f.prevEffective(c - 1), f.nextEffective(c+1, len(f.toks))}
	}
	f.beginChangeset()
	f.add(constant, " ")
	f.slvlcRemoveBetween(constant, afterConst)
	for _, c := range commas {
		f.slvlcRemoveBetween(data[c].before, c)
		doc := ""
		if hasDoc {
			doc = indentation + docComment + f.eol
		}
		f.replaceToken(c, ";"+f.eol+doc+indentation+modifiers+"const "+typeHint)
		f.slvlcRemoveBetween(c, data[c].after)
	}
	f.slvlcRemoveBetween(beforeSemicolon, semicolon)
	f.endChangeset()
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/Classes/DisallowMultiPropertyDefinitionSniff.php
//
// SlevomatDisallowMultiPropertyDefinition splits "public $a, $b;" into one
// definition per property, repeating the modifiers, type and doc comment.
type SlevomatDisallowMultiPropertyDefinition struct{}

func (SlevomatDisallowMultiPropertyDefinition) Name() string {
	return `SlevomatCodingStandard\Sniffs\Classes\DisallowMultiPropertyDefinitionSniff`
}

func (SlevomatDisallowMultiPropertyDefinition) SourceURL() string {
	return fixer.SourceURLFor(SlevomatDisallowMultiPropertyDefinition{}.Name())
}

func (SlevomatDisallowMultiPropertyDefinition) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		f.eachToken(f.clEIsModifier, func(i int) int {
			f.clEMultiProperty(i)
			return 0
		})
	})
}

// clEIsModifier tells a TokenHelper::PROPERTY_MODIFIERS_TOKEN_CODES token.
func (f *slvFile) clEIsModifier(i int) bool {
	if i < 0 || f.toks[i].kind != token.Keyword {
		return false
	}
	switch strings.ToLower(f.toks[i].orig) {
	case "final", "abstract", "var", "public", "protected", "private", "readonly", "static",
		"public(set)", "protected(set)", "private(set)":
	default:
		return false
	}
	p := f.prevEffective(i - 1)
	return !f.isMemberOperator(p) && !f.isKeyword(p, "function") && !f.isKeyword(p, "const")
}

// clEIsMemberStart tells T_VARIABLE, T_CONST, T_FUNCTION and T_CLASS.
func (f *slvFile) clEIsMemberStart(i int) bool {
	switch {
	case f.isVariable(i):
		return true
	case f.isKeyword(i, "function"):
		return !f.isClosure(i)
	case f.isKeyword(i, "class"):
		p := f.prevEffective(i - 1)
		return !f.isKeyword(p, "new") && !f.isMemberOperator(p)
	}
	return f.clEIsConst(i)
}

// clEPropertyStart is PropertyHelper::getStartPointer.
func (f *slvFile) clEPropertyStart(property int) int {
	codeEnd := 0
	for i := property - 1; i >= 0; i-- {
		if f.is(i, ";") || f.is(i, "}") || f.is(i, "{") || f.is(i, "(") || f.is(i, ",") {
			codeEnd = i
			break
		}
	}
	start := f.prevEffective(property - 1)
	if start < codeEnd {
		start = -1
	}
	for {
		possible := -1
		for i := start - 1; i >= codeEnd && i >= 0; i-- {
			if f.clEIsModifier(i) {
				possible = i
				break
			}
		}
		if possible < 0 {
			return start
		}
		start = possible
	}
}

func (f *slvFile) clEMultiProperty(modifier int) {
	if f.isKeyword(f.prevEffective(modifier-1), "as") {
		return
	}
	if f.clEIsModifier(f.nextEffective(modifier+1, len(f.toks))) {
		return
	}
	property := f.clEFindNext(modifier+1, len(f.toks), f.clEIsMemberStart)
	if property < 0 || !f.isVariable(property) || !f.isProperty(property) {
		return
	}
	end := f.clEFindNext(property+1, len(f.toks), func(i int) bool { return f.is(i, ";") || f.is(i, "{") })
	if end < 0 || f.is(end, "{") {
		return
	}
	semicolon := end
	var commas []int
	for next := property; ; {
		next = f.clEFindNext(next+1, semicolon, func(i int) bool {
			return f.is(i, ",") || f.isShortArrayOpen(i) || f.isKeyword(i, "array")
		})
		if next < 0 {
			break
		}
		if f.isShortArrayOpen(next) {
			next = f.toks[next].closer
			continue
		}
		if f.isKeyword(next, "array") {
			if o := f.nextEffective(next+1, len(f.toks)); f.is(o, "(") && f.toks[o].closer >= 0 {
				next = f.toks[o].closer
			}
			continue
		}
		commas = append(commas, next)
	}
	if len(commas) == 0 {
		return
	}
	start := f.clEPropertyStart(property)
	if start < 0 {
		return
	}
	beforeProperty := f.prevEffective(property - 1)
	beforeSemicolon := f.prevEffective(semicolon - 1)
	indentation := f.indentation(start)
	docComment := ""
	hasDoc := false
	if doc := f.docOpenBefore(property); doc >= 0 {
		hasDoc = true
		docComment = strings.Trim(f.content(doc, f.docCloser(doc)), " \t\n\r\x00\x0B")
	}
	type around struct{ before, after int }
	data := map[int]around{}
	for _, c := range commas {
		data[c] = around{f.prevEffective(c - 1), f.nextEffective(c+1, len(f.toks))}
	}
	propertyContent := f.content(start, beforeProperty)
	f.beginChangeset()
	f.change(beforeProperty+1, property-1, " ")
	for _, c := range commas {
		f.slvlcRemoveBetween(data[c].before, c)
		doc := ""
		if hasDoc {
			doc = indentation + docComment + f.eol
		}
		f.replace(c, ";"+f.eol+doc+indentation+propertyContent+" ")
		f.slvlcRemoveBetween(c, data[c].after)
	}
	f.slvlcRemoveBetween(beforeSemicolon, semicolon)
	f.endChangeset()
}
