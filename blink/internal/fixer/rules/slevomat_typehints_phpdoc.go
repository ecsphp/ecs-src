package rules

import (
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/lexer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// slvRunDocs runs process on every doc comment opener of each pass; process
// is false where upstream throws, which leaves the file unchanged in ECS.
func slvRunDocs(s *tokens.Stream, process func(d *slvDocs, open int) bool) bool {
	aborted := false
	src := s.Render()
	changed := slvRun(s, func(f *slvFile) {
		if aborted {
			return
		}
		d := newSlvDocs(f)
		f.eachToken(f.isDocOpen, func(open int) int {
			if !aborted && !process(d, open) {
				aborted = true
			}
			return 0
		})
	})
	if aborted && changed {
		s.ReplaceRange(0, s.Len()-1, lexer.Lex(src))
		return false
	}
	return changed
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/TypeHints/DisallowArrayTypeHintSyntaxSniff.php

// SlevomatDisallowArrayTypeHintSyntax rewrites "Foo[]" in annotations as a
// generic type, "array<Foo>" or the native type hint or traversable type
// it is a union with.
type SlevomatDisallowArrayTypeHintSyntax struct {
	traversableTypeHints []string
}

func (SlevomatDisallowArrayTypeHintSyntax) Name() string {
	return `SlevomatCodingStandard\Sniffs\TypeHints\DisallowArrayTypeHintSyntaxSniff`
}

func (SlevomatDisallowArrayTypeHintSyntax) SourceURL() string {
	return fixer.SourceURLFor(SlevomatDisallowArrayTypeHintSyntax{}.Name())
}

func (r SlevomatDisallowArrayTypeHintSyntax) WithConfig(config map[string]any) fixer.Fixer {
	r.traversableTypeHints = nil
	list, _ := slvStringList(config["traversableTypeHints"])
	for _, t := range list {
		if !strings.HasPrefix(t, `\`) {
			t = `\` + t
		}
		r.traversableTypeHints = append(r.traversableTypeHints, t)
	}
	return r
}

func (r SlevomatDisallowArrayTypeHintSyntax) Fix(s *tokens.Stream) bool {
	return slvRunDocs(s, r.process)
}

func (r SlevomatDisallowArrayTypeHintSyntax) process(d *slvDocs, open int) bool {
	for _, a := range d.annotations(open, "") {
		for _, arr := range slvArrayTypeNodes(a.node.value()) {
			var union *pdNode
			for _, u := range pdNodesByType(a.node, "UnionTypeNode") {
				if slices.Contains(u.fields[0].list, arr) {
					union = u
					break
				}
			}
			target := arr
			identifier := "array"
			if union != nil {
				if generic, ok := r.genericIdentifier(d, open, union, a.node.value()); ok {
					target, identifier = union, generic
				}
			} else if generic, ok := r.genericIdentifier(d, open, arr, a.node.value()); ok {
				identifier = generic
			}
			fixed := pdNew("GenericTypeNode", pdF(pdIdent(identifier)), pdL([]*pdNode{slvFixArrayNode(arr.fields[0].node)}))
			if !d.fixNode(open, target, fixed) {
				return false
			}
		}
	}
	return true
}

// slvArrayTypeNodes are the array types of a node, but not those nested
// right in another one.
func slvArrayTypeNodes(n *pdNode) []*pdNode {
	var out []*pdNode
	var walk func(n *pdNode)
	walk = func(n *pdNode) {
		if n == nil {
			return
		}
		if n.kind == "ArrayTypeNode" {
			out = append(out, n)
			if n.fields[0].node.kind == "ArrayTypeNode" {
				return
			}
		}
		for _, f := range n.fields {
			if f.isList {
				for _, c := range f.list {
					walk(c)
				}
			} else {
				walk(f.node)
			}
		}
	}
	walk(n)
	return out
}

func slvFixArrayNode(n *pdNode) *pdNode {
	if n.kind != "ArrayTypeNode" {
		return n
	}
	return pdNew("GenericTypeNode", pdF(pdIdent("array")), pdL([]*pdNode{slvFixArrayNode(n.fields[0].node)}))
}

func (r SlevomatDisallowArrayTypeHintSyntax) genericIdentifier(d *slvDocs, open int, n, value *pdNode) (string, bool) {
	if n.kind != "UnionTypeNode" {
		if value.kind != "ParamTagValueNode" && value.kind != "ReturnTagValueNode" {
			return "", false
		}
		f := d.f
		fn := -1
		for i := open + 1; i < len(f.toks); i++ {
			if f.isFunctionToken(i) {
				fn = i
				break
			}
		}
		if fn < 0 || !f.isKeyword(fn, "function") || f.isClosure(fn) {
			return "", false
		}
		if value.kind == "ParamTagValueNode" {
			return slvParameterTypeHint(f, fn, value.text)
		}
		return slvReturnTypeHint(f, fn)
	}
	types := n.fields[0].list
	if len(types) != 2 {
		return "", false
	}
	for i, j := range []int{1, 0} {
		if types[i].kind == "ArrayTypeNode" && types[j].kind == "IdentifierTypeNode" &&
			r.isTraversable(slvFullyQualifiedTypeHint(d, open, types[j].name)) {
			return types[j].name, true
		}
	}
	return "", false
}

func (r SlevomatDisallowArrayTypeHintSyntax) isTraversable(t string) bool {
	return t == "array" || t == "iterable" || slices.Contains(r.traversableTypeHints, t)
}

// slvFullyQualifiedTypeHint is TypeHintHelper::getFullyQualifiedTypeHint.
func slvFullyQualifiedTypeHint(d *slvDocs, pointer int, typeHint string) string {
	if slvTypeHintIsSimple(typeHint) {
		switch typeHint {
		case "integer":
			return "int"
		case "boolean":
			return "bool"
		}
		return typeHint
	}
	return d.useStatements().resolveClassName(d.f, typeHint, pointer)
}

var slvWhitespaceRe = regexp.MustCompile(`\s+`)

// slvParameterTypeHint is the type hint of the parameter name in
// FunctionHelper::getParametersTypeHints, false when it has none.
func slvParameterTypeHint(f *slvFile, fn int, name string) (string, bool) {
	open := f.slvmParenOpener(fn)
	if open < 0 || f.toks[open].closer < 0 {
		return "", false
	}
	var names *slvNames
	hint, found := "", false
	for i := open + 1; i < f.toks[open].closer; i++ {
		if f.toks[i].kind != token.Variable {
			continue
		}
		before := i - 1
		for before >= 0 && (f.isEmpty(before) || f.is(before, "&") || f.is(before, "...")) {
			before--
		}
		if names == nil {
			names = newSlvNames(f)
		}
		if !slvIsTypeHintToken(names, before) {
			if f.toks[i].content == name {
				hint, found = "", false
			}
			continue
		}
		start := before - 1
		for start >= 0 && (f.toks[start].kind == token.Whitespace || slvIsTypeHintToken(names, start)) {
			start--
		}
		start = f.nextNonWhitespace(start+1, len(f.toks))
		if p := f.prevEffective(start - 1); p >= 0 && f.is(p, "?") && names.nullable[p] {
			start = p
		}
		if f.toks[i].content == name {
			hint, found = slvWhitespaceRe.ReplaceAllString(f.content(start, before), ""), true
		}
	}
	return hint, found
}

// slvIsTypeHintToken tells a TokenHelper::TYPE_HINT_TOKEN_CODES token.
func slvIsTypeHintToken(n *slvNames, i int) bool {
	f := n.f
	if i < 0 {
		return false
	}
	for _, k := range []string{"self", "parent", "false", "true", "null"} {
		if f.isKeyword(i, k) {
			return true
		}
	}
	return n.isTypeHintToken(i)
}

// slvReturnTypeHint is FunctionHelper::findReturnTypeHint, false when the
// function has none.
func slvReturnTypeHint(f *slvFile, fn int) (string, bool) {
	start := f.slvmReturnTypeStart(fn)
	if start < 0 {
		return "", false
	}
	after := -1
	for j := start + 1; j < len(f.toks); j++ {
		if f.is(j, "{") || f.is(j, ";") {
			after = j
			break
		}
	}
	if after < 0 {
		return "", false
	}
	end := f.prevEffective(after - 1)
	return slvWhitespaceRe.ReplaceAllString(f.content(start, end), ""), true
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/TypeHints/LongTypeHintsSniff.php

// SlevomatLongTypeHints writes "integer" and "boolean" in annotations as
// "int" and "bool".
type SlevomatLongTypeHints struct{}

func (SlevomatLongTypeHints) Name() string {
	return `SlevomatCodingStandard\Sniffs\TypeHints\LongTypeHintsSniff`
}

func (SlevomatLongTypeHints) SourceURL() string {
	return fixer.SourceURLFor(SlevomatLongTypeHints{}.Name())
}

func (SlevomatLongTypeHints) Fix(s *tokens.Stream) bool {
	return slvRunDocs(s, func(d *slvDocs, open int) bool {
		for _, a := range d.annotations(open, "") {
			for _, n := range pdNodesByType(a.node, "IdentifierTypeNode") {
				var short string
				switch strings.ToLower(n.name) {
				case "integer":
					short = "int"
				case "boolean":
					short = "bool"
				default:
					continue
				}
				if !d.fixNode(open, n, pdIdent(short)) {
					return false
				}
			}
		}
		return true
	})
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/TypeHints/NullTypeHintOnLastPositionSniff.php

// SlevomatNullTypeHintOnLastPosition moves "null" to the end of a union in
// annotations.
type SlevomatNullTypeHintOnLastPosition struct{}

func (SlevomatNullTypeHintOnLastPosition) Name() string {
	return `SlevomatCodingStandard\Sniffs\TypeHints\NullTypeHintOnLastPositionSniff`
}

func (SlevomatNullTypeHintOnLastPosition) SourceURL() string {
	return fixer.SourceURLFor(SlevomatNullTypeHintOnLastPosition{}.Name())
}

func (SlevomatNullTypeHintOnLastPosition) Fix(s *tokens.Stream) bool {
	return slvRunDocs(s, func(d *slvDocs, open int) bool {
		for _, a := range d.annotations(open, "") {
			for _, union := range pdNodesByType(a.node, "UnionTypeNode") {
				types := union.fields[0].list
				pos := slices.IndexFunc(types, func(t *pdNode) bool {
					return t.kind == "IdentifierTypeNode" && strings.ToLower(t.name) == "null"
				})
				if pos < 0 || pos == len(types)-1 {
					continue
				}
				// upstream gives the clone the original children, which the
				// printer then prints anew instead of preserving their format
				fixed := pdClone(union)
				fixed.fields[0].list = append(slices.Delete(slices.Clone(types), pos, pos+1), types[pos])
				if !d.fixNode(open, union, fixed) {
					return false
				}
			}
		}
		return true
	})
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/TypeHints/RequireTrailingCommaInArrayShapeSniff.php

// SlevomatRequireTrailingCommaInArrayShape adds a trailing comma after the
// last item of a multi-line array shape.
type SlevomatRequireTrailingCommaInArrayShape struct{}

func (SlevomatRequireTrailingCommaInArrayShape) Name() string {
	return `SlevomatCodingStandard\Sniffs\TypeHints\RequireTrailingCommaInArrayShapeSniff`
}

func (SlevomatRequireTrailingCommaInArrayShape) SourceURL() string {
	return fixer.SourceURLFor(SlevomatRequireTrailingCommaInArrayShape{}.Name())
}

func (SlevomatRequireTrailingCommaInArrayShape) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		d := newSlvDocs(f)
		f.eachToken(f.isDocOpen, func(open int) int {
			slvTrailingCommaInArrayShapes(d, open)
			return 0
		})
	})
}

func slvTrailingCommaInArrayShapes(d *slvDocs, open int) {
	p := d.parse(open)
	if p == nil {
		return
	}
	var offsets []int
	for _, shape := range pdNodesByType(p.root, "ArrayShapeNode") {
		last := slvShapeLastTokenIndex(p.toks, shape)
		if last < 0 {
			continue
		}
		n := 0
		for _, t := range p.toks[:last+1] {
			n += len(t.v)
		}
		offsets = append(offsets, n)
	}
	if len(offsets) == 0 {
		return
	}
	sort.Sort(sort.Reverse(sort.IntSlice(offsets)))
	f := d.f
	ends := map[int]int{}
	end := 0
	for i := open; i <= p.close; i++ {
		end += len(f.toks[i].content)
		ends[i] = end
	}
	pointer := p.close
	replacements := map[int]string{}
	var order []int
	for _, offset := range offsets {
		for pointer > open && ends[pointer-1] >= offset {
			pointer--
		}
		at := offset - ends[pointer] + len(f.toks[pointer].content)
		content, ok := replacements[pointer]
		if !ok {
			content = f.toks[pointer].content
			order = append(order, pointer)
		}
		replacements[pointer] = content[:at] + "," + content[at:]
	}
	f.beginChangeset()
	for _, i := range order {
		f.replace(i, replacements[i])
	}
	f.endChangeset()
}

// slvShapeLastTokenIndex is the index of the last token of the last item of
// a multi-line array shape without a trailing comma, else -1.
func slvShapeLastTokenIndex(toks []pdToken, shape *pdNode) int {
	start, end := shape.start, shape.end
	for toks[start].t != pdOpenCurlyBracket {
		start++
	}
	depth := 0
	for i := start; i <= end; i++ {
		switch toks[i].t {
		case pdOpenCurlyBracket:
			depth++
		case pdCloseCurlyBracket:
			depth--
		}
		if toks[i].t == pdCloseCurlyBracket && depth == 0 {
			end = i
			break
		}
	}
	if toks[start].line == toks[end].line {
		return -1
	}
	last := end - 1
	for toks[last].t == pdHorizontalWS || toks[last].t == pdPhpdocEOL || toks[last].t == pdComment {
		last--
	}
	if last == start || toks[last].t == pdComma {
		return -1
	}
	return last
}
