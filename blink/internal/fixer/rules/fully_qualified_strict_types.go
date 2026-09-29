package rules

import (
	"slices"
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Import/FullyQualifiedStrictTypesFixer.php
//
// FullyQualifiedStrictTypes shortens class references to their imported /
// namespace-relative short name (default config: import_symbols=false,
// leading_backslash_in_global_namespace=false, the default phpdoc_tags list).
//
// Residuals (byte-safe: valid PHP, never corrupts, go==rust holds):
//   - Attribute-internal class names (`#[\Foo\Bar]`) are not shortened - the
//     flat lexer folds an attribute into one opaque comment token, so ECS's
//     T_ATTRIBUTE handling has no token equivalent.
//   - PHPDoc types are shortened by an atom scanner (guarding variables, keys,
//     `::` members, generic/callable bases and wildcard-generic `<*>` types)
//     rather than a full recursive TypeExpression parser; it matches ECS on the
//     verified corpora, but a sufficiently exotic unparseable type construct
//     could still differ.
type FullyQualifiedStrictTypes struct{}

func (FullyQualifiedStrictTypes) Name() string {
	return `PhpCsFixer\Fixer\Import\FullyQualifiedStrictTypesFixer`
}

func (FullyQualifiedStrictTypes) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Import/FullyQualifiedStrictTypesFixer.php"
}

var fqReservedTypes = map[string]bool{
	"array": true, "bool": true, "callable": true, "false": true, "float": true,
	"int": true, "iterable": true, "list": true, "mixed": true, "never": true,
	"null": true, "object": true, "parent": true, "resource": true, "self": true,
	"static": true, "string": true, "true": true, "void": true,
}

// fqUses holds the class-import map for one namespace. Like ECS's $uses it is
// keyed by full name, so importing the same class twice keeps only the last
// alias; the caches are then derived from that (build).
type fqUses struct {
	longs             []string // full names in first-occurrence order
	longToShort       map[string]string
	nameByShortLower  map[string]string // lower(short) -> long
	shortByName       map[string]string // long -> short
	shortByNormalized map[string]string // normalizeFqcn(long) -> short
}

func fqNewUses() *fqUses {
	return &fqUses{
		longToShort:       map[string]string{},
		nameByShortLower:  map[string]string{},
		shortByName:       map[string]string{},
		shortByNormalized: map[string]string{},
	}
}

func (u *fqUses) add(long, short string) {
	if _, ok := u.longToShort[long]; !ok {
		u.longs = append(u.longs, long)
	}
	u.longToShort[long] = short
}

// build derives the caches from the (full-name-keyed) import map.
func (u *fqUses) build() {
	for _, long := range u.longs {
		short := u.longToShort[long]
		u.nameByShortLower[strings.ToLower(short)] = long
		u.shortByName[long] = short
		u.shortByNormalized[fqNormalize(long)] = short
	}
}

func fqNormalize(input string) string {
	bs := strings.LastIndexByte(input, '\\')
	if bs < 0 {
		return strings.ToLower(input)
	}
	return input[:bs+1] + strings.ToLower(input[bs+1:])
}

func fqIsReserved(symbol string, reserved map[string]bool) bool {
	if strings.Contains(symbol, "\\") {
		return false
	}
	if fqReservedTypes[strings.ToLower(symbol)] {
		return true
	}
	return reserved[symbol]
}

// resolveSymbol resolves an absolute or relative symbol to a normalized FQCN.
func fqResolveSymbol(symbol string, u *fqUses, ns string, reserved map[string]bool) string {
	if strings.HasPrefix(symbol, "\\") {
		return symbol[1:]
	}
	if fqIsReserved(symbol, reserved) {
		return symbol
	}
	first, rest, hasRest := strings.Cut(symbol, "\\")
	if long, ok := u.nameByShortLower[strings.ToLower(first)]; ok {
		if hasRest {
			return long + "\\" + rest
		}
		return long
	}
	if ns != "" {
		return ns + "\\" + symbol
	}
	return symbol
}

// shortenSymbol shortens a normalized FQCN as much as the namespace + uses allow.
func fqShortenSymbol(fqcn string, u *fqUses, ns string, reserved map[string]bool) string {
	if fqIsReserved(fqcn, reserved) {
		return fqcn
	}
	res := ""
	found := false
	iMin := 0

	if ns != "" && strings.HasPrefix(fqcn, ns+"\\") {
		tmpRes := fqcn[len(ns)+1:]
		firstSeg, _, _ := strings.Cut(tmpRes, "\\")
		if _, collide := u.nameByShortLower[strings.ToLower(firstSeg)]; !collide && !fqIsReserved(tmpRes, reserved) {
			res = tmpRes
			found = true
			iMin = strings.Count(ns, "\\") + 1
		}
	}

	tmp := fqcn
	for i := strings.Count(fqcn, "\\"); i >= iMin; i-- {
		if short, ok := u.shortByName[tmp]; ok {
			tmpRes := short + fqcn[len(tmp):]
			if !fqIsReserved(tmpRes, reserved) {
				res = tmpRes
				found = true
				break
			}
		}
		if i > 0 {
			if bs := strings.LastIndexByte(tmp, '\\'); bs >= 0 {
				tmp = tmp[:bs]
			}
		}
	}

	if !found {
		if short, ok := u.shortByNormalized[fqNormalize(fqcn)]; ok && !fqIsReserved(short, reserved) {
			res = short
			found = true
		}
	}

	if !found {
		res = fqcn
		firstSeg, _, _ := strings.Cut(res, "\\")
		_, collide := u.nameByShortLower[strings.ToLower(firstSeg)]
		if ns != "" || collide {
			res = "\\" + res
		}
	}
	return res
}

// determineShortType returns the shortened rendering of typeName, or "" + false
// when it is already shortest (no change).
func fqDetermineShort(typeName string, u *fqUses, ns string, reserved map[string]bool) (string, bool) {
	fqcn := fqResolveSymbol(typeName, u, ns, reserved)
	shortened := fqShortenSymbol(fqcn, u, ns, reserved)
	if shortened == typeName {
		return "", false
	}
	return shortened, true
}

type fqReplace struct {
	start, end int          // inclusive token range of the name run
	repl       string       // rendered replacement (name-run splice)
	raw        *token.Token // when set, replace [start,end] with this single token
}

func (FullyQualifiedStrictTypes) Fix(s *tokens.Stream) bool {
	regions := fqNamespaceRegions(s)
	var repls []fqReplace
	for _, reg := range regions {
		repls = append(repls, fqProcessNamespace(s, reg)...)
	}
	if len(repls) == 0 {
		return false
	}
	for _, r := range slices.Backward(repls) {
		fqApply(s, r)
	}
	return true
}

func fqApply(s *tokens.Stream, r fqReplace) {
	if r.raw != nil {
		s.ReplaceRange(r.start, r.end, []token.Token{*r.raw})
		return
	}
	s.ReplaceRange(r.start, r.end, fqStringToTokens(r.repl))
}

// fqStringToTokens mirrors namespacedStringToTokens.
func fqStringToTokens(input string) []token.Token {
	var out []token.Token
	if strings.HasPrefix(input, "\\") {
		out = append(out, token.Token{Kind: token.Punct, Value: "\\"})
		input = input[1:]
	}
	parts := strings.Split(input, "\\")
	for i, p := range parts {
		out = append(out, token.Token{Kind: token.Ident, Value: p})
		if i != len(parts)-1 {
			out = append(out, token.Token{Kind: token.Punct, Value: "\\"})
		}
	}
	return out
}

type fqRegion struct {
	name       string
	start, end int // token range [start, end) of the namespace scope
}

// fqNamespaceRegions returns the namespace scopes, mirroring
// getNamespaceDeclarations: one global region when there is no namespace.
func fqNamespaceRegions(s *tokens.Stream) []fqRegion {
	var decls []int
	for i := 0; i < s.Len(); i++ {
		if kwIs(s.At(i), "namespace") {
			// `namespace\foo(...)` relative call is not a declaration
			n := sigNext(s, i)
			if n >= 0 && s.At(n).Kind == token.Punct && s.At(n).Value == "\\" {
				continue
			}
			decls = append(decls, i)
		}
	}
	if len(decls) == 0 {
		return []fqRegion{{name: "", start: 0, end: s.Len()}}
	}
	var regions []fqRegion
	for k, d := range decls {
		name, bodyStart, semiOrBrace := fqReadNamespaceName(s, d)
		if semiOrBrace < 0 {
			continue
		}
		if s.At(semiOrBrace).Value == "{" {
			end := s.MatchForward(semiOrBrace)
			if end < 0 {
				end = s.Len()
			}
			regions = append(regions, fqRegion{name: name, start: bodyStart, end: end})
		} else {
			end := s.Len()
			if k+1 < len(decls) {
				end = decls[k+1]
			}
			regions = append(regions, fqRegion{name: name, start: bodyStart, end: end})
		}
	}
	return regions
}

// fqReadNamespaceName reads the name after a `namespace` keyword, returning the
// name, the index just after the `;`/`{`, and the index of that `;`/`{`.
func fqReadNamespaceName(s *tokens.Stream, kw int) (name string, bodyStart, term int) {
	var b strings.Builder
	i := sigNext(s, kw)
	for i >= 0 && i < s.Len() {
		t := s.At(i)
		if t.Kind == token.Ident {
			b.WriteString(t.Value)
			i = sigNext(s, i)
			continue
		}
		if t.Kind == token.Punct && t.Value == "\\" {
			b.WriteByte('\\')
			i = sigNext(s, i)
			continue
		}
		if t.Kind == token.Punct && (t.Value == ";" || t.Value == "{") {
			return b.String(), i + 1, i
		}
		break
	}
	return "", -1, -1
}
