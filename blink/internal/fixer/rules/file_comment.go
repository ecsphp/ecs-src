package rules

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/Commenting/FileCommentSniff.php
//
// SquizFileComment rewrites the @author and @copyright tags of the file
// docblock to Squiz's own; a copyright keeps its leading year range, or gets
// the current year.
type SquizFileComment struct{}

func (SquizFileComment) Name() string {
	return `PHP_CodeSniffer\Standards\Squiz\Sniffs\Commenting\FileCommentSniff`
}

func (SquizFileComment) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/Commenting/FileCommentSniff.php"
}

const (
	squizAuthor    = "Squiz Pty Ltd <products@squiz.net>"
	squizCopyright = "Squiz Pty Ltd (ABN 77 084 670 600)"
)

var (
	squizCopyrightOK   = regexp.MustCompile(`^([0-9]{4})(-[0-9]{4})? (Squiz Pty Ltd \(ABN 77 084 670 600\))$`)
	squizCopyrightYear = regexp.MustCompile(`^([0-9]{4}(-[0-9]{4})?)`)
)

// fileCommentNow is the clock the sniff fills a missing copyright year from; tests pin it.
var fileCommentNow = time.Now

var fileCommentIgnore = map[string]bool{
	"class": true, "interface": true, "trait": true, "enum": true, "function": true,
	"public": true, "private": true, "protected": true, "final": true, "static": true,
	"abstract": true, "readonly": true, "const": true, "include": true,
	"include_once": true, "require": true, "require_once": true,
}

func (SquizFileComment) Fix(s *tokens.Stream) bool {
	open := -1
	for i := range s.Len() {
		if t := s.At(i); t.Kind == token.OpenTag && !strings.HasPrefix(t.Value, "<?=") {
			open = i
			break
		}
	}
	if open < 0 {
		return false
	}
	c := open + 1
	for c < s.Len() && s.At(c).Kind == token.Whitespace {
		c++
	}
	if c >= s.Len() || s.At(c).Kind != token.DocComment || !strings.HasSuffix(s.At(c).Value, "*/") {
		return false
	}
	next := c + 1
	for next < s.Len() {
		if s.At(next).Kind == token.Whitespace {
			next++
			continue
		}
		if e := s.AttributeEnd(next); s.At(next).Kind == token.AttributeStart && e >= 0 {
			next = e + 1
			continue
		}
		break
	}
	if next < s.Len() && s.At(next).Kind == token.Keyword && fileCommentIgnore[strings.ToLower(s.At(next).Value)] {
		return false
	}
	doc := s.At(c).Value
	repl := map[int][2]string{}
	for _, tag := range phpcsDocTags(doc, phpcsEolChar(s)) {
		if tag.StrPos < 0 {
			continue
		}
		switch tag.Name {
		case "@author":
			if tag.Str != squizAuthor {
				repl[tag.StrPos] = [2]string{tag.Str, squizAuthor}
			}
		case "@copyright":
			if !squizCopyrightOK.MatchString(tag.Str) {
				year := squizCopyrightYear.FindString(tag.Str)
				if year == "" {
					year = strconv.Itoa(fileCommentNow().Year())
				}
				repl[tag.StrPos] = [2]string{tag.Str, year + " " + squizCopyright}
			}
		}
	}
	if len(repl) == 0 {
		return false
	}
	s.SetValue(c, replaceDocStrings(doc, repl))
	return true
}
