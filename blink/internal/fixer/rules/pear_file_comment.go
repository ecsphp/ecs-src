package rules

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/lexer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PEAR/Sniffs/Commenting/FileCommentSniff.php
//
// PEARFileComment reports a missing or malformed file docblock: its required
// tags, their order and their content. It fixes nothing; its warnings are
// dropped by ECS, so they are not ported.
type PEARFileComment struct{}

func (PEARFileComment) Name() string {
	return `PHP_CodeSniffer\Standards\PEAR\Sniffs\Commenting\FileCommentSniff`
}

func (PEARFileComment) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PEAR/Sniffs/Commenting/FileCommentSniff.php"
}

func (PEARFileComment) Fix(*tokens.Stream) bool { return false }

func (r PEARFileComment) Errors(s *tokens.Stream) []fixer.CodingStandardError {
	m := newSiModel(newSlvFile(lexer.Lex(s.Render())))
	f, toks := m.f, m.f.toks
	e := &pearDocErrors{f: f, checker: r.Name()}
	stackPtr := -1
	for i := range toks {
		if m.code[i] == "open_tag" {
			stackPtr = i
			break
		}
	}
	if stackPtr < 0 {
		return nil
	}
	ws := func(i int) bool { return toks[i].kind == token.Whitespace }
	commentStart := f.nextNot(ws, stackPtr+1, len(toks))
	if commentStart >= 0 && m.code[commentStart] == "declare" {
		semicolon := -1
		for j := commentStart + 1; j < len(toks); j++ {
			if f.is(j, ";") {
				semicolon = j
				break
			}
		}
		commentStart = f.nextNot(ws, semicolon+1, len(toks))
	}
	if commentStart >= 0 && pearIsPlainComment(f, commentStart) && strings.Contains(toks[commentStart].orig, "vim:") {
		commentStart = f.nextNot(ws, commentStart+1, len(toks))
	}
	errorToken := min(stackPtr+1, len(toks)-1)
	switch {
	case commentStart >= 0 && m.code[commentStart] == "close_tag":
		return nil
	case commentStart >= 0 && pearIsPlainComment(f, commentStart):
		e.add(errorToken, "WrongStyle", `You must use "/**" style comments for a file comment`)
		return e.errs
	case commentStart < 0 || toks[commentStart].doc != slvDocOpen:
		e.add(errorToken, "Missing", "Missing file doc comment")
		return e.errs
	}
	commentEnd := pearDocCloser(f, commentStart)
	next := commentEnd + 1
	for ; next < len(toks); next++ {
		if ws(next) {
			continue
		}
		if toks[next].kind == token.AttributeStart && toks[next].closer > next {
			next = toks[next].closer
			continue
		}
		break
	}
	if next == len(toks) {
		next--
	}
	switch c := m.code[next]; c {
	case "class", "interface", "trait", "function", "closure", "public", "private", "protected", "final",
		"static", "abstract", "readonly", "const":
		e.add(stackPtr, "Missing", "Missing file doc comment")
		return e.errs
	case "enum":
		if pearOODeclaration(m, next) {
			e.add(stackPtr, "Missing", "Missing file doc comment")
			return e.errs
		}
	}
	e.processTags(commentStart, commentEnd, "file")
	return e.errs
}

// pearIsPlainComment is PHP_CodeSniffer's T_COMMENT: a comment that is no
// docblock and no phpcs: annotation.
func pearIsPlainComment(f *slvFile, i int) bool {
	t := f.toks[i]
	return t.kind == token.Comment && t.doc == slvDocNone && !isPhpcsAnnotationComment(t.orig)
}

// pearDocCloser is the comment_closer of the docblock opening at open.
func pearDocCloser(f *slvFile, open int) int {
	for j := open; j < len(f.toks); j++ {
		if f.toks[j].doc == slvDocClose {
			return j
		}
	}
	return len(f.toks) - 1
}

// pearDocErrors collects the errors of PEAR FileComment and its ClassComment
// subclass.
type pearDocErrors struct {
	f       *slvFile
	checker string
	errs    []fixer.CodingStandardError
}

func (e *pearDocErrors) add(i int, code, msg string) {
	e.errs = append(e.errs, fixer.CodingStandardError{Line: e.f.toks[i].line, Message: msg, Checker: e.checker + "." + code})
}

// pearDocExpectedTags is FileCommentSniff::EXPECTED_TAGS, in order.
var pearDocExpectedTags = []struct {
	name               string
	required, multiple bool
}{
	{"@category", true, false}, {"@package", true, false}, {"@subpackage", false, false},
	{"@author", true, true}, {"@copyright", false, true}, {"@license", true, false},
	{"@version", false, false}, {"@link", true, true}, {"@see", false, true},
	{"@since", false, false}, {"@deprecated", false, false},
}

func pearUcfirst(v string) string {
	if v == "" {
		return v
	}
	return strings.ToUpper(v[:1]) + v[1:]
}

func (e *pearDocErrors) processTags(commentStart, commentEnd int, docBlock string) {
	toks := e.f.toks
	var commentTags []int
	for j := commentStart; j < commentEnd; j++ {
		if toks[j].doc == slvDocTag {
			commentTags = append(commentTags, j)
		}
	}
	expected := map[string]bool{}
	multiple := map[string]bool{}
	for _, t := range pearDocExpectedTags {
		expected[t.name] = true
		multiple[t.name] = t.multiple
	}
	var foundTags []string
	tagTokens := map[string][]int{}
	for _, tag := range commentTags {
		name := toks[tag].orig
		if !expected[name] {
			continue
		}
		if !multiple[name] && len(tagTokens[name]) > 0 {
			e.add(tag, "Duplicate"+pearUcfirst(name[1:])+"Tag", "Only one "+name+" tag is allowed in a "+docBlock+" comment")
		}
		foundTags = append(foundTags, name)
		tagTokens[name] = append(tagTokens[name], tag)
		str := -1
		for j := tag; j < commentEnd; j++ {
			if toks[j].doc == slvDocString {
				str = j
				break
			}
		}
		if str < 0 || toks[str].line != toks[tag].line {
			e.add(tag, "Empty"+pearUcfirst(name[1:])+"Tag", "Content missing for "+name+" tag in "+docBlock+" comment")
		}
	}
	pos := 0
	for _, t := range pearDocExpectedTags {
		if len(tagTokens[t.name]) == 0 {
			if t.required {
				e.add(commentEnd, "Missing"+pearUcfirst(t.name[1:])+"Tag", "Missing "+t.name+" tag in "+docBlock+" comment")
			}
			continue
		}
		e.processTag(t.name, tagTokens[t.name])
		if pos >= len(foundTags) {
			break
		}
		if foundTags[pos] != t.name {
			e.add(commentTags[pos], pearUcfirst(t.name[1:])+"TagOrder",
				"The tag in position "+strconv.Itoa(pos+1)+" should be the "+t.name+" tag")
		}
		pos++
		for pos < len(foundTags) && foundTags[pos] == t.name {
			pos++
		}
	}
}

var (
	pearDocAuthor    = regexp.MustCompile(`^([^<]*)\s+<([\da-zA-Z\-_+]([\da-zA-Z\-_+.\w]*[\da-zA-Z\-_+])*@[\da-zA-Z][-.\w]*[\da-zA-Z]\.[a-zA-Z]{2,})>$`)
	pearDocCopyright = regexp.MustCompile(`^([0-9]{4})((.{1})([0-9]{4}))? (.+)$`)
	pearDocLicense   = regexp.MustCompile(`^([^\s]+)\s+(.*)`)
	pearDocNonAlpha  = regexp.MustCompile(`[^A-Za-z_]`)
)

// pearIsUnderscoreName is Common::isUnderscoreName.
func pearIsUnderscoreName(name string) bool {
	if strings.Contains(name, " ") || name == "" || name[0] < 'A' || name[0] > 'Z' {
		return false
	}
	for bit := range strings.SplitSeq(name, "_") {
		if bit != "" && bit[:1] != strings.ToUpper(bit[:1]) {
			return false
		}
	}
	return true
}

// pearSuggestName joins the "_" separated bits of v, each with an upper-cased
// first letter, as the sniff's suggestions do.
func pearSuggestName(v string) string {
	bits := strings.Split(v, "_")
	var out strings.Builder
	out.WriteString(pearUcfirst(bits[0]) + "_")
	for _, bit := range bits[1:] {
		if bit != "" {
			out.WriteString(pearUcfirst(bit) + "_")
		}
	}
	return strings.Trim(out.String(), "_")
}

// processTag runs the sniff's process<Tag> method for the tags it has one for.
// The @version check only warns, which ECS drops.
func (e *pearDocErrors) processTag(name string, tags []int) {
	toks := e.f.toks
	for _, tag := range tags {
		if tag+2 >= len(toks) || toks[tag+2].doc != slvDocString {
			continue
		}
		content := toks[tag+2].orig
		switch name {
		case "@category":
			if !pearIsUnderscoreName(content) {
				e.add(tag, "InvalidCategory", `Category name "`+content+`" is not valid; consider "`+
					pearSuggestName(strings.ReplaceAll(content, " ", "_"))+`" instead`)
			}
		case "@package":
			if pearIsUnderscoreName(content) {
				continue
			}
			v := pearDocNonAlpha.ReplaceAllString(strings.Trim(strings.ReplaceAll(content, " ", "_"), "_"), "")
			if v == "" {
				e.add(tag, "InvalidPackageValue", `Package name "`+content+`" is not valid`)
				continue
			}
			e.add(tag, "InvalidPackage", `Package name "`+content+`" is not valid; consider "`+pearSuggestName(v)+`" instead`)
		case "@subpackage":
			if !pearIsUnderscoreName(content) {
				e.add(tag, "InvalidSubpackage", `Subpackage name "`+content+`" is not valid; consider "`+
					pearSuggestName(strings.ReplaceAll(content, " ", "_"))+`" instead`)
			}
		case "@author":
			if !pearDocAuthor.MatchString(content) {
				e.add(tag, "InvalidAuthors", `Content of the @author tag must be in the form "Display Name <username@example.com>"`)
			}
		case "@copyright":
			mm := pearDocCopyright.FindStringSubmatch(content)
			if mm == nil {
				e.add(tag, "IncompleteCopyright", "@copyright tag must contain a year and the name of the copyright holder")
			} else if mm[3] != "" && mm[3] != "-" {
				e.add(tag, "CopyrightHyphen", "A hyphen must be used between the earliest and latest year")
			}
		case "@license":
			if !pearDocLicense.MatchString(content) {
				e.add(tag, "IncompleteLicense", "@license tag must contain a URL and a license name")
			}
		}
	}
}
