package rules

import (
	"cmp"
	"errors"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/PhpdocTypesOrderFixer.php
//
// PhpdocTypesOrder sorts the members of phpdoc type unions and intersections
// at every nesting level. Zero option values mean the PHP defaults.
type PhpdocTypesOrder struct {
	sortNone      bool
	nullAdjust    string
	caseSensitive bool
}

func (PhpdocTypesOrder) Name() string {
	return `PhpCsFixer\Fixer\Phpdoc\PhpdocTypesOrderFixer`
}

func (PhpdocTypesOrder) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/PhpdocTypesOrderFixer.php"
}

// WithConfig keeps PHP's defaults for absent or invalid options.
func (f PhpdocTypesOrder) WithConfig(config map[string]any) fixer.Fixer {
	switch config["sort_algorithm"] {
	case "alpha":
		f.sortNone = false
	case "none":
		f.sortNone = true
	}
	switch v := config["null_adjustment"]; v {
	case "always_first", "always_last", "none":
		f.nullAdjust = v.(string)
	}
	if v, ok := config["case_sensitive"].(bool); ok {
		f.caseSensitive = v
	}
	return f
}

// phpdocTagsWithTypes is Annotation::TAGS_WITH_TYPES.
var phpdocTagsWithTypes = []string{
	"extends", "implements", "method", "param", "param-out", "phpstan-import-type",
	"phpstan-type", "phpstan-var", "property", "property-read", "property-write",
	"psalm-import-type", "psalm-type", "psalm-var", "return", "throws", "type", "var",
}

// compare ports sortTypes' comparison callback.
func (f PhpdocTypesOrder) compare(a, b string) int {
	a, b = phpdocTypesOrderNormalize(a), phpdocTypesOrderNormalize(b)
	lowerA, lowerB := asciiLower(a), asciiLower(b)
	if aNull := lowerA == "null"; f.nullAdjust != "none" && aNull != (lowerB == "null") {
		if aNull == (f.nullAdjust == "always_last") {
			return 1
		}
		return -1
	}
	if f.sortNone {
		return 0
	}
	if f.caseSensitive {
		return phpSpaceshipStrings(a, b)
	}
	return strings.Compare(lowerA, lowerB)
}

// phpNumericStringRe is PHP 8's numeric string.
var phpNumericStringRe = regexp.MustCompile(`^[ \t\n\r\v\f]*[+-]?(?:[0-9]+(?:\.[0-9]*)?|\.[0-9]+)(?:[eE][+-]?[0-9]+)?[ \t\n\r\v\f]*$`)

// phpSpaceshipStrings ports PHP 8's `$a <=> $b` on two strings
// (zendi_smart_strcmp).
func phpSpaceshipStrings(a, b string) int {
	if !phpNumericStringRe.MatchString(a) || !phpNumericStringRe.MatchString(b) {
		return strings.Compare(a, b)
	}
	a, b = strings.Trim(a, " \t\n\r\v\f"), strings.Trim(b, " \t\n\r\v\f")
	la, errA := strconv.ParseInt(a, 10, 64)
	lb, errB := strconv.ParseInt(b, 10, 64)
	if errA == nil && errB == nil {
		return cmp.Compare(la, lb)
	}
	// an integer string out of int64's range is a float that overflowed
	oflow := func(s string, err error) int {
		if errors.Is(err, strconv.ErrRange) {
			if strings.HasPrefix(s, "-") {
				return -1
			}
			return 1
		}
		return 0
	}
	oa, ob := oflow(a, errA), oflow(b, errB)
	da, _ := strconv.ParseFloat(a, 64)
	db, _ := strconv.ParseFloat(b, 64)
	switch {
	case oa != 0 && oa == ob && da == db:
		// both overflowed to the same side
		return strings.Compare(a, b)
	case errA == nil && ob != 0:
		return -ob
	case errB == nil && oa != 0:
		return oa
	case da == db && math.IsInf(da, 0):
		return strings.Compare(a, b)
	}
	return cmp.Compare(da, db)
}

// phpdocTypesOrderNormalize is /^\(*\??\\?/ removed: leading parentheses, a
// nullable mark and a leading backslash do not count when comparing.
func phpdocTypesOrderNormalize(t string) string {
	t = strings.TrimLeft(t, "(")
	t = strings.TrimPrefix(t, "?")
	return strings.TrimPrefix(t, "\\")
}

func (f PhpdocTypesOrder) Fix(s *tokens.Stream) bool {
	changed := false
	for i := range s.Len() {
		t := s.At(i)
		if t.Kind != token.DocComment {
			continue
		}
		doc := newPHPDocBlock(t.Value)
		annotations := doc.annotationsOfType(phpdocTagsWithTypes)
		if len(annotations) == 0 {
			continue
		}
		for _, a := range annotations {
			f.sortAnnotationTypes(a)
			if a.tag == "method" {
				line := doc.lines[a.start]
				line.content = f.sortMethodSignature(line.content)
			}
		}
		if content := doc.content(); content != t.Value {
			s.SetValue(i, content)
			changed = true
		}
	}
	return changed
}

// sortAnnotationTypes ports the main-types half of applyFix: getTypesContent,
// sortTypes, then setTypes, which always re-joins a composite's members with
// a bare glue and replaces the first occurrence of the old types on the line.
func (f PhpdocTypesOrder) sortAnnotationTypes(a *phpDocAnnotation) {
	types, ok := phpdocAnnotationTypesContent(a)
	if !ok {
		return
	}
	sorted, ok := sortPHPTypeValue(types, f.compare)
	if !ok {
		return
	}
	_, glue := phpTypeMembers(types)
	if glue == 0 {
		glue = '|'
	}
	members, _ := phpTypeMembers(sorted)
	newTypes := strings.Join(members, string(glue))
	if newTypes == types {
		return
	}
	// line by line, as the types may span several
	oldLines, newLines := splitPHPDocLines(types), splitPHPDocLines(newTypes)
	if len(oldLines) != len(newLines) || len(oldLines) > len(a.lines) {
		return
	}
	for k, l := range newLines {
		a.lines[k].content = strings.Replace(a.lines[k].content, oldLines[k].content, l.content, 1)
	}
}

// phpdocAnnotationTypesContent ports Annotation::getTypesContent:
// {^(?:\h*\*|/\*\*)[\h*]*@NAME\h+TYPES(?:(?:[*\h\v]|\&?[\.\$\s]).*)?\r?$}is.
func phpdocAnnotationTypesContent(a *phpDocAnnotation) (string, bool) {
	c := a.content()
	i := skipPHPHSpace(c, 0)
	switch {
	case i < len(c) && c[i] == '*':
		i++
	case strings.HasPrefix(c, "/**"):
		i = 3
	default:
		return "", false
	}
	i = pcreSkip(c, i, func(r rune) bool { return r == '*' || pcreIsHSpace(r) })
	if n := len(a.tag) + 1; len(c) < i+n || !strings.EqualFold(c[i:i+n], "@"+a.tag) {
		return "", false
	}
	// The pattern is case-insensitive, so a "Covariant " prefix counts here;
	// TypeExpression's case-sensitive parse of such types then fails.
	start, ts, ok := parsePHPAfterHSpaces(phpdocVariancePrefixRe.ReplaceAllStringFunc(c, strings.ToLower), i+len(a.tag)+1, func(c string, p int) (phpTypes, bool) {
		ts, ok := parsePHPTypes(c, p)
		return ts, ok && phpdocTypesTailOK(c[ts.end:])
	})
	if !ok {
		return "", false
	}
	return c[start:ts.end], true
}

var phpdocVariancePrefixRe = regexp.MustCompile(`(?i)co(?:ntra)?variant`)

// phpdocTypesTailOK checks what may follow the types:
// (?:(?:[*\h\v]|\&?[\.\$\s]).*)?\r?$ with the s flag.
func phpdocTypesTailOK(rest string) bool {
	if rest == "" {
		return true
	}
	ok := func(r rune) bool { return r == '.' || r == '$' || pcreIsSpace(r) }
	if r, _ := pcreRuneAt(rest, 0); r == '*' || ok(r) {
		return true
	}
	r, _ := pcreRuneAt(rest, 1)
	return rest[0] == '&' && ok(r)
}

// sortMethodSignature ports the "@method parameters types" half of applyFix:
// /\*\h*@method\h+TYPES\h+\K(?&callable)/, the callable (with an optional
// template) sorted as a type.
func (f PhpdocTypesOrder) sortMethodSignature(line string) string {
	for from := 0; from < len(line); from++ {
		star := strings.IndexByte(line[from:], '*')
		if star < 0 {
			return line
		}
		from += star
		i := skipPHPHSpace(line, from+1)
		if !strings.HasPrefix(line[i:], "@method") {
			continue
		}
		// TYPES\h+ then the callable's name, template, arguments and return type
		_, call, ok := parsePHPAfterHSpaces(line, i+len("@method"), func(line string, p int) (phpTypeSpan, bool) {
			ts, ok := parsePHPTypes(line, p)
			if !ok {
				return phpTypeSpan{}, false
			}
			_, call, ok := parsePHPAfterHSpaces(line, ts.end, func(line string, q int) (phpTypeSpan, bool) {
				nameEnd, ok := parsePHPTypeName(line, q)
				if !ok {
					return phpTypeSpan{}, false
				}
				var t phpType
				end, ok := parsePHPCallableAfterName(line, nameEnd, &t)
				return phpTypeSpan{q, end}, ok
			})
			return call, ok
		})
		if !ok {
			continue
		}
		callStart, end := call.start, call.end
		sorted, ok := sortPHPTypeValue(line[callStart:end], f.compare)
		if !ok {
			continue
		}
		return line[:callStart] + sorted + line[end:]
	}
	return line
}

// Swaps a `$var`-before-type @var/@type into type-before-`$var`, ported from the
// fixer's Preg replace (\h -> [ \t] for RE2).
var phpdocVarOrderRe = regexp.MustCompile(`(?i)(@(?:type|var)\s*)(\$\S+)([ \t]+)([^$](?:[^<\s]|<[^>]*>)*)(\s|\*)`)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/PhpdocVarAnnotationCorrectOrderFixer.php
//
// PhpdocVarAnnotationCorrectOrder fixes swapped @var/@type order, rewriting
// "@var $x int" to "@var int $x". Already-correct "@var Type $x" is a no-op.
type PhpdocVarAnnotationCorrectOrder struct{}

func (PhpdocVarAnnotationCorrectOrder) Name() string {
	return `PhpCsFixer\Fixer\Phpdoc\PhpdocVarAnnotationCorrectOrderFixer`
}

func (PhpdocVarAnnotationCorrectOrder) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/PhpdocVarAnnotationCorrectOrderFixer.php"
}

func (PhpdocVarAnnotationCorrectOrder) Fix(s *tokens.Stream) bool {
	changed := false
	for i := range s.Len() {
		t := s.At(i)
		if t.Kind != token.DocComment {
			continue
		}
		lower := strings.ToLower(t.Value)
		if !strings.Contains(lower, "@var") && !strings.Contains(lower, "@type") {
			continue
		}
		newVal := phpdocVarOrderRe.ReplaceAllString(t.Value, "${1}${4}${3}${2}${5}")
		if newVal != t.Value {
			s.SetValue(i, newVal)
			changed = true
		}
	}
	return changed
}
