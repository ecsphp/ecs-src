package rules

import (
	"sort"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Import/GroupImportFixer.php
//
// GroupImport merges single imports that share a namespace into a group import:
// "use A\B; use A\C;" -> "use A\{B, C};". Imports are grouped per type (classy,
// functions, constants); group_types selects which types are grouped (all by
// default). Each type's whole block is emitted at the position of its first
// import in sort order (by namespace length, then full name); single imports
// and disabled types keep their place.
type GroupImport struct {
	// groupTypes is the enabled set; nil means every type (the default).
	groupTypes map[string]bool
}

const (
	groupImportClassy    = "classy"
	groupImportFunctions = "functions"
	groupImportConstants = "constants"
)

func (GroupImport) Name() string {
	return `PhpCsFixer\Fixer\Import\GroupImportFixer`
}

func (GroupImport) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Import/GroupImportFixer.php"
}

func (GroupImport) IsCandidate(s *tokens.Stream) bool {
	return s.Index().HasWord("use")
}

func (f GroupImport) WithConfig(config map[string]any) fixer.Fixer {
	if raw, ok := config["group_types"]; ok {
		types := map[string]bool{}
		switch list := raw.(type) {
		case []any:
			for _, v := range list {
				if sv, ok := v.(string); ok {
					types[sv] = true
				}
			}
		case []string:
			for _, sv := range list {
				types[sv] = true
			}
		}
		f.groupTypes = types
	}
	return f
}

func (f GroupImport) enabled(typ string) bool {
	if f.groupTypes == nil {
		return true
	}
	return f.groupTypes[typ]
}

// groupImportDecl is one parsed single-import statement.
type groupImportDecl struct {
	start      int    // index of the "use" keyword
	consumeEnd int    // last token index removed with the statement (";" or its trailing whitespace)
	typ        string // classy, functions or constants
	fullName   string // the imported name as written, e.g. "A\B"
	realLast   string // the last segment of fullName
	alias      string // the alias, or "" when not aliased
}

func (f GroupImport) Fix(s *tokens.Stream) bool {
	decls := groupImportParse(s)
	if len(decls) == 0 {
		return false
	}

	counts := map[string]int{}
	for _, d := range decls {
		counts[groupImportKey(d)]++
	}

	byType := map[string][]groupImportDecl{}
	for _, d := range decls {
		if counts[groupImportKey(d)] > 1 && f.enabled(d.typ) {
			byType[d.typ] = append(byType[d.typ], d)
		}
	}
	if len(byType) == 0 {
		return false
	}

	consumeEnd := map[int]int{}
	blockAt := map[int][]token.Token{}
	for _, list := range byType {
		sort.SliceStable(list, func(a, b int) bool {
			na := groupImportNamespace(list[a].fullName)
			nb := groupImportNamespace(list[b].fullName)
			if len(na) != len(nb) {
				return len(na) < len(nb)
			}
			return list[a].fullName < list[b].fullName
		})
		for _, d := range list {
			consumeEnd[d.start] = d.consumeEnd
		}
		blockAt[list[0].start] = groupImportBlock(list)
	}

	out := make([]token.Token, 0, s.Len())
	for i := 0; i < s.Len(); {
		if blk, ok := blockAt[i]; ok {
			out = append(out, blk...)
			i = consumeEnd[i] + 1
			continue
		}
		if end, ok := consumeEnd[i]; ok {
			i = end + 1
			continue
		}
		out = append(out, s.At(i))
		i++
	}

	s.ReplaceRange(0, s.Len()-1, out)
	return true
}

// groupImportParse collects the top-level single-import statements. A grouped
// ("use A\{...}"), multi ("use A, B") or closure/trait use is left untouched.
func groupImportParse(s *tokens.Stream) []groupImportDecl {
	var decls []groupImportDecl
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind != token.Keyword || !strings.EqualFold(t.Value, "use") {
			continue
		}
		if inClassLikeBody(s, i) {
			continue // trait use
		}
		j := sigNext(s, i)
		if j < 0 || isPunctVal(s, j, "(") {
			continue // closure use
		}

		typ := groupImportClassy
		if s.At(j).Kind == token.Keyword {
			switch strings.ToLower(s.At(j).Value) {
			case "function":
				typ = groupImportFunctions
				j = sigNext(s, j)
			case "const":
				typ = groupImportConstants
				j = sigNext(s, j)
			}
		}

		var full strings.Builder
		alias := ""
		aliased := false
		complex := false
		semi := -1
		for k := j; k >= 0 && k < s.Len(); k = sigNext(s, k) {
			tk := s.At(k)
			switch {
			case tk.Kind == token.Punct && tk.Value == ";":
				semi = k
			case tk.Kind == token.Punct && tk.Value == `\`:
				full.WriteString(`\`)
			case tk.Kind == token.Punct && (tk.Value == "{" || tk.Value == ","):
				complex = true
			case tk.Kind == token.Ident:
				if aliased {
					alias = tk.Value
				} else {
					full.WriteString(tk.Value)
				}
			case tk.Kind == token.Keyword && strings.EqualFold(tk.Value, "as"):
				aliased = true
			}
			if semi >= 0 || complex {
				break
			}
		}
		if complex || semi < 0 {
			continue
		}

		end := semi
		if semi+1 < s.Len() && s.At(semi+1).Kind == token.Whitespace {
			end = semi + 1
		}
		fullName := full.String()
		realLast := fullName
		if idx := strings.LastIndexByte(realLast, '\\'); idx >= 0 {
			realLast = realLast[idx+1:]
		}
		decls = append(decls, groupImportDecl{
			start:      i,
			consumeEnd: end,
			typ:        typ,
			fullName:   fullName,
			realLast:   realLast,
			alias:      alias,
		})
	}
	return decls
}

func groupImportKey(d groupImportDecl) string {
	return groupImportNamespace(d.fullName) + "\x00" + d.typ
}

// groupImportNamespace returns the namespace prefix (with trailing "\") of a
// full import name, or the whole name when it has no usable "\" separator.
func groupImportNamespace(full string) string {
	i := strings.LastIndexByte(full, '\\')
	if i <= 0 {
		return full
	}
	return full[:i+1]
}

// groupImportBlock renders the group-use lines for one type, already sorted.
// Consecutive imports with the same namespace form one "use Prefix\{...};" line.
func groupImportBlock(sorted []groupImportDecl) []token.Token {
	var out []token.Token
	for i := 0; i < len(sorted); {
		ns := groupImportNamespace(sorted[i].fullName)
		j := i
		for j < len(sorted) && groupImportNamespace(sorted[j].fullName) == ns {
			j++
		}
		out = append(out, groupImportGroupTokens(sorted[i].typ, ns, sorted[i:j])...)
		i = j
	}
	return out
}

func groupImportGroupTokens(typ, ns string, members []groupImportDecl) []token.Token {
	out := []token.Token{
		{Kind: token.Keyword, Value: "use"},
		{Kind: token.Whitespace, Value: " "},
	}
	switch typ {
	case groupImportFunctions:
		out = append(out, token.Token{Kind: token.Keyword, Value: "function"}, token.Token{Kind: token.Whitespace, Value: " "})
	case groupImportConstants:
		out = append(out, token.Token{Kind: token.Keyword, Value: "const"}, token.Token{Kind: token.Whitespace, Value: " "})
	}

	for part := range strings.SplitSeq(strings.TrimRight(ns, `\`), `\`) {
		out = append(out, token.Token{Kind: token.Ident, Value: part}, token.Token{Kind: token.Punct, Value: `\`})
	}
	out = append(out, token.Token{Kind: token.Punct, Value: "{"})

	for idx, m := range members {
		if idx > 0 {
			out = append(out, token.Token{Kind: token.Punct, Value: ","}, token.Token{Kind: token.Whitespace, Value: " "})
		}
		out = append(out, token.Token{Kind: token.Ident, Value: m.realLast})
		if m.alias != "" {
			out = append(out,
				token.Token{Kind: token.Whitespace, Value: " "},
				token.Token{Kind: token.Keyword, Value: "as"},
				token.Token{Kind: token.Whitespace, Value: " "},
				token.Token{Kind: token.Ident, Value: m.alias},
			)
		}
	}

	out = append(out,
		token.Token{Kind: token.Punct, Value: "}"},
		token.Token{Kind: token.Punct, Value: ";"},
		token.Token{Kind: token.Whitespace, Value: "\n"},
	)
	return out
}
