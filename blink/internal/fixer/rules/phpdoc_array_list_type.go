package rules

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// phpTypeNode is a type node passed to a walkPhpTypes callback, bottom-up. value
// is the node text after its inner types have been mapped. For a composite
// (union/intersection) node, glue is '|' or '&' and members holds the member
// texts; for a leaf, glue is 0 and members is nil.
type phpTypeNode struct {
	value   string
	glue    byte
	members []string
}

// walkPhpTypes ports TypeExpression::mapTypes: cb is called for every type node,
// innermost first, and its result replaces the node. A value that does not parse
// is returned unchanged.
func walkPhpTypes(s string, cb func(phpTypeNode) string) (string, bool) {
	ts, ok := parsePHPTypeValue(s)
	if !ok {
		return s, false
	}
	spans, glue := ts.composite()
	composite := glue != 0
	var inners []phpTypeSpan
	if composite {
		inners = spans
	} else {
		t, _ := parsePHPType(s, 0)
		switch {
		case t.prefixLen > 0 && t.nullableLen > 0:
			return s, false
		case t.nullableLen > 0:
			inners = []phpTypeSpan{{t.nullableLen, len(s)}}
		case t.arrayStart < t.end:
			inners = []phpTypeSpan{{0, t.arrayStart}}
		case t.prefixLen > 0:
			// the generic, callable, shape and parenthesized branches only
			// register inner types when they start the value
		default:
			inners = t.inners
		}
	}
	var b strings.Builder
	last := 0
	for _, sp := range inners {
		mapped, ok := walkPhpTypes(s[sp.start:sp.end], cb)
		if !ok {
			return s, false
		}
		b.WriteString(s[last:sp.start])
		b.WriteString(mapped)
		last = sp.end
	}
	b.WriteString(s[last:])
	value := b.String()
	node := phpTypeNode{value: value}
	if composite {
		node.members, node.glue = phpTypeMembers(value)
	}
	return cb(node), true
}

// fixPhpdocTypeNodes applies transform to the REGEX_TYPES span of every typed
// annotation line of d, mirroring AbstractPhpdocTypesFixer. Using the type
// parser for the span keeps spaced array suffixes like "int []" in the type.
func fixPhpdocTypeNodes(d *docblock, transform func(string) string) bool {
	changed := false
	for i, l := range d.inner {
		trimmed := strings.TrimLeft(l.content, " ")
		loc := phpdocTypedTagRe.FindStringIndex(trimmed)
		if loc == nil {
			continue
		}
		rest := trimmed[loc[1]:]
		ts, ok := parsePHPTypes(rest, 0)
		if !ok {
			continue
		}
		typ := rest[:ts.end]
		newType := transform(typ)
		if newType == typ {
			continue
		}
		lead := l.content[:len(l.content)-len(trimmed)]
		d.inner[i].content = lead + trimmed[:loc[1]] + newType + rest[ts.end:]
		changed = true
	}
	return changed
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/PhpdocArrayTypeFixer.php
//
// PhpdocArrayType rewrites a phpdoc "T[]" type to "array<T>" (risky: "T[]" in a
// union means something else). The rewrite runs on every type node, so nested
// and union members are converted independently.
type PhpdocArrayType struct{}

func (PhpdocArrayType) Name() string {
	return `PhpCsFixer\Fixer\Phpdoc\PhpdocArrayTypeFixer`
}

func (PhpdocArrayType) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/PhpdocArrayTypeFixer.php"
}

func (PhpdocArrayType) Fix(s *tokens.Stream) bool {
	return applyToDocblocks(s, func(d *docblock) bool {
		return fixPhpdocTypeNodes(d, func(typ string) string {
			got, ok := walkPhpTypes(typ, arrayTypeNode)
			if !ok {
				return typ
			}
			return got
		})
	})
}

func arrayTypeNode(n phpTypeNode) string {
	if n.glue != 0 {
		return n.value
	}
	return arrayTypeNormalize(n.value)
}

var (
	arrayTypeLeadingQuoteRe = regexp.MustCompile(`^\??\s*['"]`)
	arrayTypeSuffixRe       = regexp.MustCompile(`^(.+?)((?:[ \t]*\[[ \t]*\])+)$`)
)

// arrayTypeNormalize ports PhpdocArrayTypeFixer::normalize: a trailing run of
// "[]" (with optional spaces) becomes that many "array<...>" wrappers.
func arrayTypeNormalize(typ string) string {
	if arrayTypeLeadingQuoteRe.MatchString(typ) {
		return typ
	}
	prefix := ""
	if strings.HasPrefix(typ, "?") {
		prefix = "?"
		typ = typ[1:]
	}
	loc := arrayTypeSuffixRe.FindStringSubmatchIndex(typ)
	if loc == nil {
		return prefix + typ
	}
	inner := typ[loc[2]:loc[3]]
	level := strings.Count(typ[loc[4]:loc[5]], "[")
	if strings.HasPrefix(inner, "(") && strings.HasSuffix(inner, ")") {
		inner = inner[1 : len(inner)-1]
	}
	return prefix + strings.Repeat("array<", level) + inner + strings.Repeat(">", level)
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/PhpdocListTypeFixer.php
//
// PhpdocListType rewrites a phpdoc keyless "array<T>" type to "list<T>" (risky:
// the array may actually need a key). A keyed "array<K, V>" is left as is.
type PhpdocListType struct{}

func (PhpdocListType) Name() string {
	return `PhpCsFixer\Fixer\Phpdoc\PhpdocListTypeFixer`
}

func (PhpdocListType) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/PhpdocListTypeFixer.php"
}

func (PhpdocListType) Fix(s *tokens.Stream) bool {
	return applyToDocblocks(s, func(d *docblock) bool {
		return fixPhpdocTypeNodes(d, func(typ string) string {
			got, ok := walkPhpTypes(typ, listTypeNode)
			if !ok {
				return typ
			}
			return got
		})
	})
}

func listTypeNode(n phpTypeNode) string {
	if n.glue != 0 {
		return n.value
	}
	return listTypeNormalize(n.value)
}

// listTypeLookaheadRe is PhpdocListTypeFixer's lookahead
// <(?:[^,<]|<[^>]+>)+(>|{|\() as a plain anchored pattern.
var listTypeLookaheadRe = regexp.MustCompile(`^<(?:[^,<]|<[^>]+>)+(>|\{|\()`)

// listTypeNormalize ports PhpdocListTypeFixer::normalize: the regex
// /\barray(?=<(?:[^,<]|<[^>]+>)+(>|{|\())/i, replacing each matched "array"
// with "list". The word boundary uses Unicode word characters (PhpCsFixer\Preg
// runs patterns with UCP), so "\My_Array" and "\KůňArray" are left alone.
func listTypeNormalize(typ string) string {
	var b strings.Builder
	i := 0
	for i < len(typ) {
		if i+5 <= len(typ) && strings.EqualFold(typ[i:i+5], "array") && listTypeWordBoundaryBefore(typ, i) && listTypeLookaheadRe.MatchString(typ[i+5:]) {
			b.WriteString("list")
			i += 5
			continue
		}
		_, size := utf8.DecodeRuneInString(typ[i:])
		b.WriteString(typ[i : i+size])
		i += size
	}
	return b.String()
}

// listTypeWordBoundaryBefore reports whether position i begins a word boundary:
// the rune before it is not a Unicode word character (or i is at the start).
func listTypeWordBoundaryBefore(typ string, i int) bool {
	if i == 0 {
		return true
	}
	r, _ := utf8.DecodeLastRuneInString(typ[:i])
	return r != '_' && !unicode.IsLetter(r) && !unicode.IsDigit(r)
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/PhpdocTypesNoDuplicatesFixer.php
//
// PhpdocTypesNoDuplicates removes duplicate members from phpdoc type unions. It
// is a deprecated proxy for PhpdocNoDuplicateTypesFixer and matches its output.
type PhpdocTypesNoDuplicates struct{}

func (PhpdocTypesNoDuplicates) Name() string {
	return `PhpCsFixer\Fixer\Phpdoc\PhpdocTypesNoDuplicatesFixer`
}

func (PhpdocTypesNoDuplicates) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/PhpdocTypesNoDuplicatesFixer.php"
}

func (PhpdocTypesNoDuplicates) Fix(s *tokens.Stream) bool {
	return applyToDocblocks(s, func(d *docblock) bool {
		return fixPhpdocTypeNodes(d, func(typ string) string {
			got, ok := walkPhpTypes(typ, removeDuplicatePhpTypesNode)
			if !ok {
				return typ
			}
			return got
		})
	})
}

// removeDuplicatePhpTypesNode ports TypeExpression::removeDuplicateTypes: at each
// composite node, members are deduplicated by their sorted string form (first
// spelling kept) and rejoined with the bare glue.
func removeDuplicatePhpTypesNode(n phpTypeNode) string {
	if n.glue == 0 {
		return n.value
	}
	seen := map[string]bool{}
	unique := make([]string, 0, len(n.members))
	for _, m := range n.members {
		key, ok := sortPHPTypeValue(m, strings.Compare)
		if !ok {
			key = m
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		unique = append(unique, m)
	}
	return strings.Join(unique, string(n.glue))
}
