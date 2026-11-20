package rules

import (
	"regexp"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/Commenting/RequireOneLineDocCommentSniff.php

// SlevomatRequireOneLineDocComment collapses a doc comment without a
// description holding one line of content to a one-line doc comment.
type SlevomatRequireOneLineDocComment struct{}

func (SlevomatRequireOneLineDocComment) Name() string {
	return `SlevomatCodingStandard\Sniffs\Commenting\RequireOneLineDocCommentSniff`
}

func (SlevomatRequireOneLineDocComment) SourceURL() string {
	return fixer.SourceURLFor(SlevomatRequireOneLineDocComment{}.Name())
}

func (SlevomatRequireOneLineDocComment) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		f.eachToken(f.isDocOpen, func(open int) int {
			f.cmBRequireOneLine(open)
			return 0
		})
	})
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/Commenting/RequireOneLinePropertyDocCommentSniff.php

// SlevomatRequireOneLinePropertyDocComment is RequireOneLineDocComment for
// the doc comments of properties only.
type SlevomatRequireOneLinePropertyDocComment struct{}

func (SlevomatRequireOneLinePropertyDocComment) Name() string {
	return `SlevomatCodingStandard\Sniffs\Commenting\RequireOneLinePropertyDocCommentSniff`
}

func (SlevomatRequireOneLinePropertyDocComment) SourceURL() string {
	return fixer.SourceURLFor(SlevomatRequireOneLinePropertyDocComment{}.Name())
}

func (SlevomatRequireOneLinePropertyDocComment) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		f.eachToken(f.isDocOpen, func(open int) int {
			property := -1
			for i := open + 1; i < len(f.toks); i++ {
				if f.thIsVariable(i) {
					property = i
					break
				}
			}
			if property < 0 || !f.isProperty(property) || f.docOpenBefore(property) != open || f.thHasDescription(open) {
				return 0
			}
			f.cmBRequireOneLine(open)
			return 0
		})
	})
}

// cmBRequireOneLine is AbstractRequireOneLineDocComment::process.
func (f *slvFile) cmBRequireOneLine(open int) {
	if f.thHasDescription(open) {
		return
	}
	closeIdx := f.docCloser(open)
	lineDifference := f.toks[closeIdx].line - f.toks[open].line
	if lineDifference == 0 {
		return
	}
	for current := open; ; {
		current = f.firstOnNextLine(current)
		if current < 0 || current >= closeIdx {
			break
		}
		star := -1
		for i := current; i < closeIdx; i++ {
			if f.toks[i].doc == slvDocStar {
				star = i
				break
			}
		}
		if star < 0 {
			break
		}
		next := star + 1
		for next <= closeIdx && f.toks[next].doc == slvDocWhitespace {
			next++
		}
		if f.toks[current].line == f.toks[next].line {
			continue
		}
		lineDifference--
	}
	// looks like a compound doc comment
	if lineDifference > 2 {
		return
	}
	isPadding := func(i int) bool { return f.toks[i].doc == slvDocWhitespace || f.toks[i].doc == slvDocStar }
	contentStart := -1
	for i := open + 1; i < closeIdx; i++ {
		if !isPadding(i) {
			contentStart = i
			break
		}
	}
	if contentStart < 0 {
		f.removeBetweenIncluding(open+1, closeIdx-1)
		return
	}
	contentEnd := closeIdx - 1
	for contentEnd > open && isPadding(contentEnd) {
		contentEnd--
	}
	f.beginChangeset()
	for i := open + 1; i < closeIdx; i++ {
		if i >= contentStart && i <= contentEnd {
			if i == contentEnd {
				f.replace(i, strings.TrimRight(f.tokenContent(i), " "))
			}
			continue
		}
		f.replace(i, "")
	}
	f.addBefore(contentStart, " ")
	f.addBefore(closeIdx, " ")
	f.endChangeset()
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/Commenting/DisallowOneLinePropertyDocCommentSniff.php

// SlevomatDisallowOneLinePropertyDocComment expands the one-line doc comment
// of a property to a multi-line one.
type SlevomatDisallowOneLinePropertyDocComment struct{}

func (SlevomatDisallowOneLinePropertyDocComment) Name() string {
	return `SlevomatCodingStandard\Sniffs\Commenting\DisallowOneLinePropertyDocCommentSniff`
}

func (SlevomatDisallowOneLinePropertyDocComment) SourceURL() string {
	return fixer.SourceURLFor(SlevomatDisallowOneLinePropertyDocComment{}.Name())
}

func (SlevomatDisallowOneLinePropertyDocComment) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		f.eachToken(f.thIsVariable, func(property int) int {
			if !f.isProperty(property) {
				return 0
			}
			open := f.docOpenBefore(property)
			if open < 0 {
				return 0
			}
			closeIdx := f.docCloser(open)
			if f.toks[closeIdx].line != f.toks[open].line {
				return 0
			}
			indent := " "
			for i := open; i >= 0; i-- {
				if f.toks[i].kind == token.Whitespace {
					indent = f.toks[i].content + " "
					break
				}
			}
			f.beginChangeset()
			f.replaceToken(open, f.tokenContent(open)+f.eol)
			f.add(open, indent)
			f.add(open, "*")
			if closeIdx-1 != open {
				f.replace(closeIdx-1, strings.TrimRight(f.tokenContent(closeIdx-1), " "))
			}
			f.addBefore(closeIdx, indent)
			f.replaceToken(closeIdx, f.eol+f.tokenContent(closeIdx))
			f.endChangeset()
			return 0
		})
	})
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/Commenting/InlineDocCommentDeclarationSniff.php

// SlevomatInlineDocCommentDeclaration turns a "/* @var" comment into a doc
// comment and swaps "@var $variable type" to "@var type $variable".
type SlevomatInlineDocCommentDeclaration struct {
	allowDocCommentAboveReturn bool
}

func (SlevomatInlineDocCommentDeclaration) Name() string {
	return `SlevomatCodingStandard\Sniffs\Commenting\InlineDocCommentDeclarationSniff`
}

func (SlevomatInlineDocCommentDeclaration) SourceURL() string {
	return fixer.SourceURLFor(SlevomatInlineDocCommentDeclaration{}.Name())
}

func (r SlevomatInlineDocCommentDeclaration) WithConfig(config map[string]any) fixer.Fixer {
	r.allowDocCommentAboveReturn = thBool(config, "allowDocCommentAboveReturn", false)
	return r
}

var (
	cmBCommentVar   = regexp.MustCompile(`^/\*\s*@var\s+`)
	cmBVariableType = regexp.MustCompile(`(?i)(\$\w+)(?:\s+(.+))?$`)
	cmBWhitespace   = regexp.MustCompile(`\s`)
)

func (r SlevomatInlineDocCommentDeclaration) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		d := newSlvDocs(f)
		isComment := func(i int) bool { return f.toks[i].kind == token.Comment && f.toks[i].doc == 0 }
		f.eachToken(func(i int) bool { return f.isDocOpen(i) || isComment(i) }, func(open int) int {
			r.process(d, open, isComment(open))
			return 0
		})
	})
}

func (r SlevomatInlineDocCommentDeclaration) process(d *slvDocs, open int, comment bool) {
	f := d.f
	closeIdx := open
	if !comment {
		closeIdx = f.docCloser(open)
	}
	if after := f.nextEffective(closeIdx+1, len(f.toks)); after >= 0 {
		for after >= 0 && f.isAttributeOpen(after) {
			after = f.nextEffective(f.toks[after].closer+1, len(f.toks))
		}
		if after >= 0 {
			if f.cmBIsScopeModifier(after) || f.isKeyword(after, "readonly") || f.isKeyword(after, "final") || f.isKeyword(after, "const") {
				return
			}
			if f.isKeyword(after, "static") {
				afterStatic := f.nextEffective(after+1, len(f.toks))
				if afterStatic >= 0 && (f.cmBIsScopeModifier(afterStatic) || f.isKeyword(afterStatic, "readonly")) {
					return
				}
				if afterStatic >= 0 && f.thIsVariable(afterStatic) && f.isProperty(afterStatic) {
					return
				}
			}
		}
	}
	if comment {
		r.checkCommentType(f, open)
		return
	}
	annotations := d.annotations(open, "@var")
	if len(annotations) == 0 {
		return
	}
	if r.allowDocCommentAboveReturn {
		if after := f.nextEffective(closeIdx+1, len(f.toks)); after < 0 || f.isKeyword(after, "return") {
			return
		}
	}
	r.checkFormat(d, open, annotations)
}

func (SlevomatInlineDocCommentDeclaration) checkCommentType(f *slvFile, open int) {
	content := f.toks[open].content
	if !cmBCommentVar.MatchString(content) {
		return
	}
	f.beginChangeset()
	f.replace(open, "/**"+content[2:])
	f.endChangeset()
}

func (SlevomatInlineDocCommentDeclaration) checkFormat(d *slvDocs, open int, annotations []slvAnnotation) {
	f := d.f
	p := d.parse(open)
	for _, a := range annotations {
		if !a.isInvalid() && cmBVarTagVariable(p.toks, a.node.value()) != "" {
			continue
		}
		matches := cmBVariableType.FindStringSubmatch(cmBTagValueString(p.toks, a.node.value()))
		if matches == nil || matches[2] == "" || cmBWhitespace.MatchString(matches[2]) {
			continue
		}
		start, end := f.thAnnotationPointers(p, a.node)
		f.beginChangeset()
		f.add(start, " "+matches[2]+" "+matches[1]+" ")
		f.removeBetweenIncluding(start+1, end)
		f.endChangeset()
	}
}

// cmBVarTagVariable is VarTagValueNode::$variableName: the variable right
// after the type.
func cmBVarTagVariable(toks []pdToken, value *pdNode) string {
	if value.kind != "VarTagValueNode" {
		return ""
	}
	for i := value.fields[0].node.end + 1; i < len(toks) && i <= value.end; i++ {
		switch toks[i].t {
		case pdHorizontalWS:
			continue
		case pdVariable, pdThisVariable:
			return toks[i].v
		}
		return ""
	}
	return ""
}

// cmBTagValueString is the __toString of a tag value without a variable: its
// text with the line breaks of a description as plain newlines.
func cmBTagValueString(toks []pdToken, value *pdNode) string {
	var b strings.Builder
	for i := max(value.start, 0); i <= value.end && i < len(toks); i++ {
		if toks[i].t == pdPhpdocEOL {
			b.WriteString("\n")
			continue
		}
		b.WriteString(toks[i].v)
	}
	return strings.Trim(b.String(), " \t\n\r")
}

// cmBIsScopeModifier is Tokens::SCOPE_MODIFIERS.
func (f *slvFile) cmBIsScopeModifier(i int) bool {
	if f.toks[i].kind != token.Keyword && f.toks[i].kind != token.Ident {
		return false
	}
	switch strings.ToLower(f.toks[i].orig) {
	case "public", "protected", "private", "public(set)", "protected(set)", "private(set)":
		return true
	}
	return false
}
