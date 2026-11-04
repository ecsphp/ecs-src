package rules

import (
	"slices"
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Alias/EregToPregFixer.php
//
// EregToPreg replaces ereg, eregi, ereg_replace, eregi_replace, split and spliti
// with their preg_* equivalent when the pattern is a literal string that is a
// valid PCRE.
type EregToPreg struct{}

func (EregToPreg) Name() string {
	return `PhpCsFixer\Fixer\Alias\EregToPregFixer`
}

func (EregToPreg) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Alias/EregToPregFixer.php"
}

var eregFunctions = [][3]string{
	{"ereg", "preg_match", ""},
	{"eregi", "preg_match", "i"},
	{"ereg_replace", "preg_replace", ""},
	{"eregi_replace", "preg_replace", "i"},
	{"split", "preg_split", ""},
	{"spliti", "preg_split", "i"},
}

var eregDelimiters = []string{"/", "#", "!"}

func (EregToPreg) Fix(s *tokens.Stream) bool {
	changed := false
	for _, m := range eregFunctions {
		curr := 0
		for {
			fn, str, ok := eregFindSequence(s, m[0], curr)
			if !ok {
				break
			}
			curr = str
			if !aliasIsGlobalFunctionCall(s, fn) {
				continue
			}
			// the first argument must be just the string
			if next := aliasNextMeaningful(s, str); next < 0 || (!isPunctVal(s, next, ",") && !isPunctVal(s, next, ")")) {
				continue
			}

			prefix, lit := "", s.At(str).Value
			if lit[0] == 'b' || lit[0] == 'B' {
				prefix, lit = lit[:1], lit[1:] // a binary string
			}
			quote := lit[:1]
			pattern := lit[1 : len(lit)-1]
			delim := eregBestDelimiter(pattern)
			preg := delim + strings.ReplaceAll(pattern, delim, `\`+delim) + delim + "D" + m[2]
			if !eregIsValidPreg(preg) {
				continue
			}
			s.SetValue(fn, m[1])
			s.SetValue(str, prefix+quote+preg+quote)
			changed = true
		}
	}
	if changed {
		aliasRelex(s)
	}
	return changed
}

// eregFindSequence finds, from index start on, the name followed by "(" and a
// literal string, returning the name and string indices.
func eregFindSequence(s *tokens.Stream, name string, start int) (fn, str int, ok bool) {
	for i := start; i < s.Len(); i++ {
		if s.At(i).Kind != token.Ident || !strings.EqualFold(s.At(i).Value, name) {
			continue
		}
		open := aliasNextMeaningful(s, i)
		if !isPunctVal(s, open, "(") {
			continue
		}
		j := aliasNextMeaningful(s, open)
		if j < 0 {
			continue
		}
		if aliasIsConstantString(s.At(j)) {
			return i, j, true
		}
	}
	return 0, 0, false
}

// eregBestDelimiter returns the least used delimiter, the earliest on a tie.
func eregBestDelimiter(pattern string) string {
	best, bestCount := "", -1
	for _, d := range eregDelimiters {
		if n := strings.Count(pattern, d); bestCount < 0 || n < bestCount {
			best, bestCount = d, n
		}
	}
	return best
}

// eregIsValidPreg stands in for PHP-CS-Fixer's check that preg_match accepts
// the pattern and runs it on an empty subject. Blink has no PCRE, so it accepts
// only a subset of PCRE2 whose validity it decides exactly: a missed fix is
// harmless, a wrong one would fail at runtime.
func eregIsValidPreg(preg string) bool {
	if strings.IndexByte(preg, 0) >= 0 {
		return false // PHP before 8.2 cuts the pattern at a NUL byte
	}
	// PHP's delimiter scan: the first unescaped delimiter ends the pattern
	delim := preg[0]
	end := -1
	for i := 1; i < len(preg); i++ {
		if preg[i] == '\\' {
			i++
		} else if preg[i] == delim {
			end = i
			break
		}
	}
	if end < 0 || (preg[end+1:] != "D" && preg[end+1:] != "Di") {
		return false
	}
	ps := eregPcreParser{p: preg[1:end]}
	_, _, ok := ps.alternation()
	return ok && ps.i == len(ps.p) && ps.maxRef <= ps.groups
}

const (
	// eregMaxSize bounds the estimated compiled size (a repeated group is
	// copied once per repetition), far below PCRE2's "regular expression is
	// too large"
	eregMaxSize = 10000
	// eregMaxDepth bounds group nesting, far below PCRE2's limit of 250
	eregMaxDepth = 50
	// eregMaxCount is PCRE2's largest {n,m} bound
	eregMaxCount = 65535
	// eregMaxWays and eregMaxCost bound the ways to match the empty subject,
	// which the check runs on, and the cost of trying them, far below PHP's
	// backtrack limit, past which preg_match fails
	eregMaxWays = 64
	eregMaxCost = 10000
)

// eregPcreParser walks a pattern body in the accepted PCRE2 subset:
// literals, ".", "^", "$", escaped punctuation, \d \D \w \W \s \S \h \H \v \V,
// \b \B \A \z \Z \G, \n \r \t \f \e \a, \xh, \xhh, backreferences \1-\9 to
// existing groups, classes (literals, ASCII ranges in order, the class escapes
// above, [:name:] and [:^name:]), groups "(", "(?:", "(?>", lookaheads "(?="
// and "(?!", fixed-length lookbehinds "(?<=" and "(?<!", options "(?i)" and
// "(?i:" (letters imsU), comments "(?#...)", "|", and the quantifiers * + ?
// {n} {n,} {n,m} (optionally lazy or possessive) on a repeatable item, but not
// on a group that can match the empty string.
type eregPcreParser struct {
	p      string
	i      int
	depth  int
	groups int // capturing groups
	maxRef int // highest backreference
	size   int // estimated compiled size
}

// eregAtom describes a parsed item: whether a quantifier may follow it,
// whether it is a group, in how many ways it matches the empty string (0: it
// cannot), and what trying it on the empty string costs.
type eregAtom struct {
	repeatable, group bool
	ways, cost        int
}

// alternation parses alternatives up to a ")" or the end, returning the ways
// to match the empty string and the cost of trying them all.
func (ps *eregPcreParser) alternation() (ways, cost int, ok bool) {
	for {
		w, c, ok := ps.sequence()
		if !ok {
			return 0, 0, false
		}
		ways += w
		cost += c
		if ways > eregMaxWays || cost > eregMaxCost {
			return 0, 0, false
		}
		if ps.i < len(ps.p) && ps.p[ps.i] == '|' {
			ps.i++
			continue
		}
		return ways, cost, true
	}
}

// sequence parses items, each optionally quantified, up to "|", ")" or the
// end, returning the ways to match the empty string and the cost of trying
// them all.
func (ps *eregPcreParser) sequence() (ways, cost int, ok bool) {
	ways = 1
	for ps.i < len(ps.p) && ps.p[ps.i] != '|' && ps.p[ps.i] != ')' {
		before := ps.size
		a, ok := ps.atom()
		if !ok {
			return 0, 0, false
		}
		if ps.atQuantifier() {
			if !a.repeatable || (a.group && a.ways > 0) {
				return 0, 0, false
			}
			lower, upper, ok := ps.quantifier()
			if !ok {
				return 0, 0, false
			}
			if a.group {
				// PCRE2 copies a group once per repetition
				ps.size = before + (ps.size-before)*min(upper+1, eregMaxSize+1)
			}
			if ps.i < len(ps.p) && (ps.p[ps.i] == '?' || ps.p[ps.i] == '+') {
				ps.i++ // lazy or possessive
			}
			if ps.atQuantifier() {
				return 0, 0, false
			}
			if lower == 0 {
				a.ways = 1 // the atom cannot match empty: only no repetition does
			}
		}
		// the matcher tries the item once per way to match what precedes it,
		// and the ways multiply (past an item that cannot match empty, it
		// stops trying)
		cost += ways * max(a.cost, 1)
		ways *= a.ways
		if ps.size > eregMaxSize || ways > eregMaxWays || cost > eregMaxCost {
			return 0, 0, false
		}
	}
	return ways, cost, true
}

// atQuantifier reports whether a quantifier, or something PCRE2 versions
// disagree on reading as one ("{" then a digit, "," or a space), starts here.
func (ps *eregPcreParser) atQuantifier() bool {
	if ps.i >= len(ps.p) {
		return false
	}
	switch ps.p[ps.i] {
	case '*', '+', '?':
		return true
	case '{':
		return ps.i+1 < len(ps.p) && strings.IndexByte("0123456789, \t\n\r\f\v", ps.p[ps.i+1]) >= 0
	}
	return false
}

// quantifier parses * + ? or a strict {n}, {n,} or {n,m}; upper is the
// largest repetition count (the lower one for an open range, 1 for * and +).
func (ps *eregPcreParser) quantifier() (lower, upper int, ok bool) {
	c := ps.p[ps.i]
	ps.i++
	switch c {
	case '*', '?':
		return 0, 1, true
	case '+':
		return 1, 1, true
	}
	lo, ok := ps.number()
	if !ok {
		return 0, 0, false
	}
	hi := lo
	if ps.i < len(ps.p) && ps.p[ps.i] == ',' {
		ps.i++
		if ps.i < len(ps.p) && ps.p[ps.i] != '}' {
			if hi, ok = ps.number(); !ok || hi < lo {
				return 0, 0, false
			}
		}
	}
	if ps.i >= len(ps.p) || ps.p[ps.i] != '}' {
		return 0, 0, false
	}
	ps.i++
	return lo, hi, true
}

// number parses 1 to 5 digits up to eregMaxCount.
func (ps *eregPcreParser) number() (int, bool) {
	start, n := ps.i, 0
	for ps.i < len(ps.p) && ps.p[ps.i] >= '0' && ps.p[ps.i] <= '9' && ps.i-start < 5 {
		n = n*10 + int(ps.p[ps.i]-'0')
		ps.i++
	}
	return n, ps.i > start && n <= eregMaxCount && (ps.i >= len(ps.p) || ps.p[ps.i] < '0' || ps.p[ps.i] > '9')
}

// atom parses one item.
func (ps *eregPcreParser) atom() (eregAtom, bool) {
	char := eregAtom{repeatable: true}
	assertion := eregAtom{ways: 1}
	ps.size += 2
	if ps.atQuantifier() {
		return eregAtom{}, false // nothing to repeat, or an unclear "{"
	}
	c := ps.p[ps.i]
	ps.i++
	switch c {
	case 0:
		return eregAtom{}, false
	case '^', '$':
		return assertion, true
	case '[':
		return char, ps.class()
	case '(':
		return ps.group()
	case '\\':
		if ps.i >= len(ps.p) {
			return eregAtom{}, false
		}
		e := ps.p[ps.i]
		ps.i++
		switch {
		case strings.IndexByte("dDwWsShHvVnrtfea", e) >= 0:
			return char, true
		case strings.IndexByte("bBAzZG", e) >= 0:
			return assertion, true
		case e == 'x':
			return char, ps.hexDigits()
		case e >= '1' && e <= '9':
			if ps.i < len(ps.p) && ps.p[ps.i] >= '0' && ps.p[ps.i] <= '9' {
				return eregAtom{}, false
			}
			ps.maxRef = max(ps.maxRef, int(e-'0'))
			// it matches empty when the group did
			return eregAtom{repeatable: true, ways: 1}, true
		}
		return char, eregIsEscapablePunct(e)
	}
	return char, true
}

// hexDigits parses the one or two hex digits of a \xh or \xhh escape.
func (ps *eregPcreParser) hexDigits() bool {
	isHex := func(k int) bool {
		if k >= len(ps.p) {
			return false
		}
		c := ps.p[k]
		return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
	}
	if !isHex(ps.i) {
		return false
	}
	ps.i++
	if isHex(ps.i) {
		ps.i++
	}
	return true
}

// eregIsEscapablePunct reports whether a backslash before c makes a literal.
func eregIsEscapablePunct(c byte) bool {
	return c >= ' ' && c <= '~' && (c < '0' || c > '9') && (c < 'a' || c > 'z') && (c < 'A' || c > 'Z')
}

// group parses what follows a "(" up to its ")".
func (ps *eregPcreParser) group() (eregAtom, bool) {
	a := eregAtom{repeatable: true, group: true}
	atomic := false
	rest := ps.p[ps.i:]
	switch {
	case strings.HasPrefix(rest, "?#"):
		end := strings.IndexByte(rest, ')')
		if end < 0 {
			return eregAtom{}, false
		}
		ps.i += end + 1
		// a quantifier after a comment applies to the item before it: refuse
		return eregAtom{ways: 1}, true
	case strings.HasPrefix(rest, "?<=") || strings.HasPrefix(rest, "?<!"):
		ps.i += 3
		return eregAtom{ways: 1}, ps.lookbehind()
	case strings.HasPrefix(rest, "?:"):
		ps.i += 2
	case strings.HasPrefix(rest, "?>"):
		ps.i += 2
		atomic = true
	case strings.HasPrefix(rest, "?=") || strings.HasPrefix(rest, "?!"):
		ps.i += 2
		a = eregAtom{}
		atomic = true
	case strings.HasPrefix(rest, "?"):
		// option settings: (?i) alone, or (?i:...) as a group
		flags := rest[1:]
		n := len(flags) - len(strings.TrimLeft(flags, "imsU-"))
		on, off, hasOff := strings.Cut(flags[:n], "-")
		if n == len(flags) || (on == "" && off == "") || (hasOff && off == "") || strings.Contains(off, "-") {
			return eregAtom{}, false
		}
		ps.i += 1 + n + 1
		if flags[n] == ')' {
			return eregAtom{ways: 1}, true
		}
		if flags[n] != ':' {
			return eregAtom{}, false
		}
	default:
		ps.groups++
	}
	if ps.depth++; ps.depth > eregMaxDepth {
		return eregAtom{}, false
	}
	ways, cost, ok := ps.alternation()
	if !ok || ps.i >= len(ps.p) || ps.p[ps.i] != ')' {
		return eregAtom{}, false
	}
	ps.depth--
	ps.i++
	a.ways, a.cost = ways, cost+1
	if atomic {
		// matched at most once, never retried
		a.ways = min(ways, 1)
	}
	if !a.repeatable {
		// a negative lookahead matches empty when its body does not
		a.ways = 1
	}
	return a, true
}

// lookbehind parses the body of a lookbehind up to its ")": alternatives of
// single characters and assertions, with {n} as the only quantifier, so that
// each alternative has a fixed length, as PCRE2 before 10.43 requires.
func (ps *eregPcreParser) lookbehind() bool {
	length := 0
	for ps.i < len(ps.p) {
		switch c := ps.p[ps.i]; {
		case c == ')':
			ps.i++
			return length <= 255
		case c == '|':
			ps.i++
			length = 0
			continue
		case c == '(' && !strings.HasPrefix(ps.p[ps.i:], "(?#"),
			c == '\\' && ps.i+1 < len(ps.p) && ps.p[ps.i+1] >= '0' && ps.p[ps.i+1] <= '9':
			return false
		}
		a, ok := ps.atom()
		if !ok {
			return false
		}
		width := 0
		if a.repeatable {
			width = 1
		}
		if ps.atQuantifier() {
			if !a.repeatable || ps.p[ps.i] != '{' {
				return false
			}
			lower, upper, ok := ps.quantifier()
			if !ok || lower != upper || ps.atQuantifier() || (ps.i < len(ps.p) && ps.p[ps.i] == '+') {
				return false
			}
			width = lower
		}
		length += width
		if length > 255 {
			return false
		}
	}
	return false
}

// eregPosixClasses are the [:name:] classes PCRE2 knows.
var eregPosixClasses = []string{
	"alnum", "alpha", "ascii", "blank", "cntrl", "digit", "graph",
	"lower", "print", "punct", "space", "upper", "word", "xdigit",
}

// class parses a character class after its "[" up to its "]".
func (ps *eregPcreParser) class() bool {
	// PCRE2 refuses "[:alpha:]" and the like outside a class
	if ps.i < len(ps.p) && strings.IndexByte(":.=", ps.p[ps.i]) >= 0 {
		return false
	}
	if ps.i < len(ps.p) && ps.p[ps.i] == '^' {
		ps.i++
	}
	first := true
	for ps.i < len(ps.p) {
		if ps.p[ps.i] == ']' && !first {
			ps.i++
			return true
		}
		first = false
		lo, isChar, ok := ps.classAtom()
		if !ok {
			return false
		}
		// a "-" before the "]" is a literal
		if ps.i+1 >= len(ps.p) || ps.p[ps.i] != '-' || ps.p[ps.i+1] == ']' {
			continue
		}
		ps.i++
		hi, hiIsChar, ok := ps.classAtom()
		if !ok || !isChar || !hiIsChar || lo > hi {
			return false
		}
		// keep "-" right after a range out: PCRE2 versions differ there
		if ps.i+1 < len(ps.p) && ps.p[ps.i] == '-' && ps.p[ps.i+1] != ']' {
			return false
		}
	}
	return false
}

// classAtom parses one class item; isChar says it is a single ASCII character
// (its code in c), which alone may end a range.
func (ps *eregPcreParser) classAtom() (c byte, isChar, ok bool) {
	ps.size += 2
	b := ps.p[ps.i]
	ps.i++
	switch {
	case b == 0:
		return 0, false, false
	case b == '[':
		if ps.i >= len(ps.p) || strings.IndexByte(":.=", ps.p[ps.i]) < 0 {
			return b, true, true
		}
		if ps.p[ps.i] != ':' {
			return 0, false, false // collating elements and equivalence classes
		}
		name, rest, found := strings.Cut(ps.p[ps.i+1:], ":]")
		name = strings.TrimPrefix(name, "^")
		if !found || !slices.Contains(eregPosixClasses, name) {
			return 0, false, false
		}
		ps.i = len(ps.p) - len(rest)
		return 0, false, true
	case b == '\\':
		if ps.i >= len(ps.p) {
			return 0, false, false
		}
		e := ps.p[ps.i]
		ps.i++
		switch {
		case strings.IndexByte("dDwWsShHvVnrtfeab", e) >= 0:
			return 0, false, true
		case e == 'x':
			return 0, false, ps.hexDigits() // a range end only when written plainly
		case eregIsEscapablePunct(e):
			return e, true, true
		}
		return 0, false, false
	case b >= 0x80:
		return 0, false, true // part of a UTF-8 character or a Latin-1 byte
	}
	return b, true, true
}
