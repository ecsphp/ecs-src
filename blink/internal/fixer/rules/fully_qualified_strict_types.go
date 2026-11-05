package rules

import (
	"slices"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Import/FullyQualifiedStrictTypesFixer.php
//
// FullyQualifiedStrictTypes shortens class references to their imported /
// namespace-relative short name, in code and in the types of the phpdoc_tags
// tags. It reads all three options: phpdoc_tags (an empty list leaves
// docblocks alone), leading_backslash_in_global_namespace, and import_symbols,
// which first walks the namespace to discover every class reference, imports
// the namespaced ones whose short name is free (`use` statements inserted after
// the last class import, else after the namespace declaration, the open tag or
// a leading declare) and then shortens with those imports.
//
// The walk is a port of applyFix: the same handlers, in the same order, on
// PHP-CS-Fixer's token model (symScan), so the symbols are discovered in the
// order PHP discovers them, duplicates included.
//
// Left out: when a phpdoc type matched by REGEX_TYPES does not parse with the
// TypeExpression port (or would make PHP's TypeExpression misbehave: a
// covariant/contravariant prefix on a nullable type or a class constant), or
// when the imports would go somewhere the port does not model (after a short
// `<?` open tag, or at the end of the file), the file is left untouched rather
// than fixed differently.
type FullyQualifiedStrictTypes struct {
	ImportSymbols                     bool
	LeadingBackslashInGlobalNamespace bool
	// PhpdocTags replaces the default tag list when PhpdocTagsSet.
	PhpdocTags    []string
	PhpdocTagsSet bool
}

func (FullyQualifiedStrictTypes) Name() string {
	return `PhpCsFixer\Fixer\Import\FullyQualifiedStrictTypesFixer`
}

func (FullyQualifiedStrictTypes) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Import/FullyQualifiedStrictTypesFixer.php"
}

// WithConfig reads import_symbols, leading_backslash_in_global_namespace and
// phpdoc_tags; an option that is not set keeps its default.
func (f FullyQualifiedStrictTypes) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["import_symbols"].(bool); ok {
		f.ImportSymbols = v
	}
	if v, ok := config["leading_backslash_in_global_namespace"].(bool); ok {
		f.LeadingBackslashInGlobalNamespace = v
	}
	if tags, ok := phpdocTagsStringList(config["phpdoc_tags"]); ok {
		f.PhpdocTags, f.PhpdocTagsSet = tags, true
	}
	// a PHP list with gaps is dumped as an object
	if tags, ok := config["phpdoc_tags"].(map[string]any); ok {
		f.PhpdocTags, f.PhpdocTagsSet = make([]string, 0, len(tags)), true
		for _, tag := range tags {
			if name, ok := tag.(string); ok {
				f.PhpdocTags = append(f.PhpdocTags, name)
			}
		}
	}
	return f
}

// fqDefaultPhpdocTags is the phpdoc_tags default.
var fqDefaultPhpdocTags = []string{
	"param", "phpstan-param", "phpstan-property", "phpstan-property-read", "phpstan-property-write",
	"phpstan-return", "phpstan-var", "property", "property-read", "property-write", "psalm-param",
	"psalm-property", "psalm-property-read", "psalm-property-write", "psalm-return", "psalm-var",
	"return", "see", "throws", "var",
}

func (f FullyQualifiedStrictTypes) phpdocTags() map[string]bool {
	list := fqDefaultPhpdocTags
	if f.PhpdocTagsSet {
		list = f.PhpdocTags
	}
	out := make(map[string]bool, len(list))
	for _, t := range list {
		out[t] = true
	}
	return out
}

// fqUses is the class part of applyFix's $uses: full name -> short name, in
// insertion order (a PHP array keeps a key's place when it is overwritten),
// with the caches refreshUsesCache derives from it.
type fqUses struct {
	longs             []string
	longToShort       map[string]string
	nameByShortLower  map[string]string // lower(short) -> long
	shortByName       map[string]string // long -> short
	shortByNormalized map[string]string // normalizeFqcn(long) -> short
}

func fqNewUses() *fqUses {
	u := &fqUses{longToShort: map[string]string{}}
	u.build()
	return u
}

func (u *fqUses) add(long, short string) {
	if _, ok := u.longToShort[long]; !ok {
		u.longs = append(u.longs, long)
	}
	u.longToShort[long] = short
}

// build derives the caches (refreshUsesCache).
func (u *fqUses) build() {
	u.nameByShortLower = map[string]string{}
	u.shortByName = map[string]string{}
	u.shortByNormalized = map[string]string{}
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

func fqFirstSegment(s string) string {
	first, _, _ := strings.Cut(s, `\`)
	return first
}

func fqLastSegment(s string) string {
	if bs := strings.LastIndexByte(s, '\\'); bs >= 0 {
		return s[bs+1:]
	}
	return s
}

// fqCtx is the fixer's state while it walks one namespace.
type fqCtx struct {
	f    FullyQualifiedStrictTypes
	sc   *symScan
	s    *tokens.Stream
	tags map[string]bool
	ns   string
	uses *fqUses
	// reserved is reservedIdentifiersByLevel: @template/@type names, by the
	// curly-brace level they are visible from.
	reserved map[int]map[string]bool
	// discovering is the symbol-discovery phase of import_symbols; the
	// discovered symbols are collected in discovered.
	discovering bool
	discovered  []string
	edits       map[int]fqEdit
	fail        bool // something the port cannot mirror: leave the file alone
}

// fqEdit replaces the inclusive token range [start, end] (an insertion before
// start when end < start) with toks.
type fqEdit struct {
	start, end int
	toks       []token.Token
}

func (c *fqCtx) addEdit(e fqEdit) {
	if _, seen := c.edits[e.start]; seen {
		return // a later visit of a name PHP already shortened is a no-op
	}
	c.edits[e.start] = e
}

func (c *fqCtx) isReserved(symbol string) bool {
	if strings.Contains(symbol, `\`) {
		return false
	}
	if phpReservedTypes[strings.ToLower(symbol)] {
		return true
	}
	for _, ids := range c.reserved {
		if ids[symbol] {
			return true
		}
	}
	return false
}

// resolveSymbol resolves an absolute or relative symbol to a normalized FQCN.
func (c *fqCtx) resolveSymbol(symbol string, u *fqUses) string {
	if strings.HasPrefix(symbol, `\`) {
		return symbol[1:]
	}
	if c.isReserved(symbol) {
		return symbol
	}
	first, rest, hasRest := strings.Cut(symbol, `\`)
	if long, ok := u.nameByShortLower[strings.ToLower(first)]; ok {
		if hasRest {
			return long + `\` + rest
		}
		return long
	}
	if c.ns != "" {
		return c.ns + `\` + symbol
	}
	return symbol
}

// shortenSymbol shortens a normalized FQCN as much as the namespace and uses allow.
func (c *fqCtx) shortenSymbol(fqcn string, u *fqUses) string {
	if c.isReserved(fqcn) {
		return fqcn
	}
	res, found := "", false
	iMin := 0
	if c.ns != "" && strings.HasPrefix(fqcn, c.ns+`\`) {
		tmpRes := fqcn[len(c.ns)+1:]
		if _, collide := u.nameByShortLower[strings.ToLower(fqFirstSegment(tmpRes))]; !collide && !c.isReserved(tmpRes) {
			res, found = tmpRes, true
			iMin = strings.Count(c.ns, `\`) + 1
		}
	}
	tmp := fqcn
	for i := strings.Count(fqcn, `\`); i >= iMin; i-- {
		if short, ok := u.shortByName[tmp]; ok {
			if tmpRes := short + fqcn[len(tmp):]; !c.isReserved(tmpRes) {
				res, found = tmpRes, true
				break
			}
		}
		if i > 0 {
			tmp = tmp[:strings.LastIndexByte(tmp, '\\')]
		}
	}
	if !found {
		if short, ok := u.shortByNormalized[fqNormalize(fqcn)]; ok && !c.isReserved(short) {
			res, found = short, true
		}
	}
	if !found {
		res = fqcn
		_, collide := u.nameByShortLower[strings.ToLower(fqFirstSegment(res))]
		if c.ns != "" || c.f.LeadingBackslashInGlobalNamespace || collide {
			res = `\` + res
		}
	}
	return res
}

// determineShortType returns the shortened rendering of typeName and whether
// it differs. While discovering, it records typeName instead.
func (c *fqCtx) determineShortType(typeName string) (string, bool) {
	if c.discovering {
		if !c.isReserved(typeName) {
			c.discovered = append(c.discovered, typeName)
		}
		return "", false
	}
	shortened := c.shortenSymbol(c.resolveSymbol(typeName, c.uses), c.uses)
	if shortened == typeName {
		return "", false
	}
	return shortened, true
}

// visit is shortenClassIfPossible for the inclusive token range [start, end].
func (c *fqCtx) visit(start, end int) {
	var b strings.Builder
	for j := start; j <= end; j++ {
		b.WriteString(c.s.At(j).Value)
	}
	if b.Len() == 0 {
		return
	}
	if repl, changed := c.determineShortType(b.String()); changed {
		c.addEdit(fqEdit{start: start, end: end, toks: fqStringToTokens(repl)})
	}
}

// fqOrdered is a PHP array with string keys: insertion order, an overwrite
// keeps the key's place.
type fqOrdered struct {
	keys []string
	vals map[string]string
}

func newFqOrdered() *fqOrdered { return &fqOrdered{vals: map[string]string{}} }

func (o *fqOrdered) set(k, v string) {
	if _, ok := o.vals[k]; !ok {
		o.keys = append(o.keys, k)
	}
	o.vals[k] = v
}

func (o *fqOrdered) unset(k string) {
	if _, ok := o.vals[k]; !ok {
		return
	}
	delete(o.vals, k)
	o.keys = slices.DeleteFunc(o.keys, func(x string) bool { return x == k })
}

// setupUsesFromDiscoveredSymbols adds an import for every discovered symbol
// whose short name is free, and returns the new imports (short -> full).
func (c *fqCtx) setupUsesFromDiscoveredSymbols() *fqOrdered {
	forImport := newFqOrdered()
	if len(c.discovered) == 0 {
		return forImport
	}
	byShort := newFqOrdered() // discoveredFqcnByShortNameLower
	if c.ns == "" {
		for _, symbol := range c.discovered {
			if !strings.HasPrefix(symbol, `\`) {
				start := fqFirstSegment(symbol)
				byShort.set(strings.ToLower(start), c.resolveSymbol(start, c.uses))
			}
		}
	}
	useByShortLower := map[string]bool{}
	for _, long := range c.uses.longs {
		short := c.uses.longToShort[long]
		byShort.set(strings.ToLower(short), long)
		useByShortLower[strings.ToLower(short)] = true
	}
	symbols := slices.Clone(c.discovered)
	slices.SortStableFunc(symbols, func(a, b string) int {
		if ra, rb := strings.HasPrefix(a, `\`), strings.HasPrefix(b, `\`); ra != rb {
			if ra {
				return 1
			}
			return -1
		}
		return strings.Count(a, `\`) - strings.Count(b, `\`)
	})
	for _, symbol := range symbols {
		for {
			shortEndLower := strings.ToLower(fqLastSegment(symbol))
			if _, taken := byShort.vals[shortEndLower]; !taken {
				shortStartLower := strings.ToLower(fqFirstSegment(strings.TrimLeft(symbol, `\`)))
				if strings.HasPrefix(symbol, `\`) || (c.ns == "" && !useByShortLower[shortStartLower]) || !strings.Contains(symbol, `\`) {
					byShort.set(shortEndLower, c.resolveSymbol(symbol, c.uses))
					break
				}
			}
			// else short name collision - keep unimported
			if strings.HasPrefix(symbol, `\`) || c.ns == "" || !strings.Contains(symbol, `\`) {
				break
			}
			symbol = symbol[:strings.LastIndexByte(symbol, '\\')]
		}
	}
	for _, long := range c.uses.longs {
		key := strings.ToLower(c.uses.longToShort[long])
		if v, ok := byShort.vals[key]; ok && strings.EqualFold(v, long) {
			byShort.unset(key)
		}
	}
	empty := fqNewUses()
	for _, key := range slices.Clone(byShort.keys) {
		fqcn := byShort.vals[key]
		shortened := strings.TrimLeft(c.shortenSymbol(fqcn, empty), `\`)
		// prevent importing non-namespaced names in global namespace
		if strings.Contains(shortened, `\`) {
			shortEnd := fqLastSegment(fqcn)
			c.uses.add(fqcn, shortEnd)
			forImport.set(shortEnd, fqcn)
		}
	}
	c.uses.build()
	return forImport
}

func (f FullyQualifiedStrictTypes) Fix(s *tokens.Stream) bool {
	sc := sharedSymScan(s)
	if !fqIsCandidate(sc) {
		return false
	}
	tags := f.phpdocTags()
	var edits []fqEdit
	for _, decl := range sc.nsDecls {
		c := &fqCtx{f: f, sc: sc, s: s, tags: tags, ns: decl.name, edits: map[int]fqEdit{}}
		var lastUseEnd int
		c.uses, lastUseEnd = fqCollectUses(sc, decl)
		var forImport *fqOrdered
		if f.ImportSymbols {
			c.discovering = true
			c.walk(decl)
			c.discovering = false
			forImport = c.setupUsesFromDiscoveredSymbols()
		}
		c.walk(decl)
		if c.fail {
			return false
		}
		if forImport != nil && len(forImport.keys) > 0 {
			ins, ok := fqImportEdit(sc, decl, lastUseEnd, forImport)
			if !ok {
				return false
			}
			edits = append(edits, ins)
		}
		for _, e := range c.edits {
			edits = append(edits, e)
		}
	}
	if len(edits) == 0 {
		return false
	}
	slices.SortStableFunc(edits, func(a, b fqEdit) int {
		if a.start != b.start {
			return a.start - b.start
		}
		return (a.end - a.start) - (b.end - b.start) // an insertion goes before a replacement
	})
	out := make([]token.Token, 0, s.Len()+16)
	next := 0
	for _, e := range edits {
		if e.start < next {
			continue // overlapping: keep the first
		}
		for j := next; j < e.start; j++ {
			out = append(out, s.At(j))
		}
		out = append(out, e.toks...)
		next = max(e.end+1, e.start)
	}
	for j := next; j < s.Len(); j++ {
		out = append(out, s.At(j))
	}
	s.ReplaceRange(0, s.Len()-1, fqMergeWhitespace(out))
	return true
}

// fqIsCandidate mirrors isCandidate.
func fqIsCandidate(sc *symScan) bool {
	for i := range sc.n {
		t := sc.tok(i)
		switch t.Kind {
		case token.AttributeStart, token.DocComment, token.Variable:
			return true
		case token.Punct:
			if t.Value == "::" {
				return true
			}
		case token.Keyword:
			if sc.isKeyword(i, "catch", "extends", "implements", "instanceof", "new") || sc.isFunctionKeyword(i) || sc.isTraitUse(i) {
				return true
			}
		}
	}
	return false
}

// fqMergeWhitespace joins adjacent whitespace tokens, as the lexer gives them.
func fqMergeWhitespace(toks []token.Token) []token.Token {
	out := toks[:0:0]
	for _, t := range toks {
		if n := len(out); n > 0 && t.Kind == token.Whitespace && out[n-1].Kind == token.Whitespace {
			out[n-1].Value += t.Value
			continue
		}
		out = append(out, t)
	}
	return out
}

// fqStringToTokens mirrors namespacedStringToTokens.
func fqStringToTokens(input string) []token.Token {
	var out []token.Token
	if strings.HasPrefix(input, `\`) {
		out = append(out, nsSeparatorToken())
		input = input[1:]
	}
	parts := strings.Split(input, `\`)
	for i, p := range parts {
		out = append(out, token.Token{Kind: token.Ident, Value: p})
		if i != len(parts)-1 {
			out = append(out, nsSeparatorToken())
		}
	}
	return out
}
