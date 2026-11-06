package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/AttributeNotation/GeneralAttributeRemoveFixer.php
//
// GeneralAttributeRemove removes the attributes whose fully-qualified name is in
// the configured "attributes" list. Its default is an empty list, so the default
// behaviour is a no-op. Names are resolved to their FQN against the surrounding
// namespace and use imports, mirroring AttributeAnalyzer::determineAttributeFullyQualifiedName.
type GeneralAttributeRemove struct {
	attributes []string
}

func (GeneralAttributeRemove) Name() string {
	return `PhpCsFixer\Fixer\AttributeNotation\GeneralAttributeRemoveFixer`
}

func (GeneralAttributeRemove) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/AttributeNotation/GeneralAttributeRemoveFixer.php"
}

func (f GeneralAttributeRemove) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["attributes"].([]any); ok {
		f.attributes = nil
		for _, a := range v {
			if name, ok := a.(string); ok {
				f.attributes = append(f.attributes, name)
			}
		}
	}
	return f
}

func (f GeneralAttributeRemove) Fix(s *tokens.Stream) bool {
	if len(f.attributes) == 0 {
		return false
	}
	wanted := make(map[string]bool, len(f.attributes))
	for _, a := range f.attributes {
		wanted[a] = true
	}
	changed := false
	index := -1
	for {
		gi := attrNextStart(s, index)
		if gi < 0 {
			break
		}
		g := collectOneAttrGroup(s, gi)
		endIndex := g.end
		removed := 0
		for _, el := range g.attrs {
			if !wanted[grafFullyQualifiedName(s, el.name, el.nameStart)] {
				continue
			}
			grafClearRange(s, el.start, el.end)
			removed++
			changed = true
			if sib := nonEmptySibling(s, el.end, 1); sib >= 0 && isPunctVal(s, sib, ",") {
				clearTokenAt(s, sib)
			}
		}
		if removed > 0 && removed == len(g.attrs) {
			grafClearRange(s, g.start, g.end)
		}
		if ti := sigPrev(s, g.closing); removed > 0 && ti >= 0 && isPunctVal(s, ti, ",") {
			clearTokenAt(s, ti)
		}
		index = endIndex
	}
	return changed
}

// grafClearRange empties every token in [a,b], like Tokens::clearRange.
func grafClearRange(s *tokens.Stream, a, b int) {
	for j := a; j <= b && j < s.Len(); j++ {
		clearTokenAt(s, j)
	}
}

// grafFullyQualifiedName ports AttributeAnalyzer::determineAttributeFullyQualifiedName:
// it resolves an attribute name written at nameStart to its FQN.
func grafFullyQualifiedName(s *tokens.Stream, name string, nameStart int) string {
	if strings.HasPrefix(name, `\`) {
		return name
	}
	nsName, scopeStart, scopeEnd := grafNamespaceInfo(s, nameStart)
	useMap := grafUseMap(s, scopeStart, scopeEnd)
	first, rest := name, ""
	if i := strings.IndexByte(name, '\\'); i >= 0 {
		first, rest = name[:i], name[i:]
	}
	if fqn, ok := useMap[first]; ok {
		if name == first {
			return fqn
		}
		return fqn + rest
	}
	return nsName + `\` + name
}

// grafNamespaceInfo returns the name of the namespace enclosing idx ("" for the
// global namespace) and the token range of its scope.
func grafNamespaceInfo(s *tokens.Stream, idx int) (string, int, int) {
	n := s.Len()
	for i := 1; i < n; i++ {
		if !aliasIsNamespaceDeclaration(s, i) {
			continue
		}
		declEnd := -1
		for j := i + 1; j < n; j++ {
			if isPunctVal(s, j, ";") || isPunctVal(s, j, "{") {
				declEnd = j
				break
			}
		}
		if declEnd < 0 {
			break
		}
		var name strings.Builder
		for j := i + 1; j < declEnd; j++ {
			if s.At(j).Kind != token.Whitespace {
				name.WriteString(s.At(j).Value)
			}
		}
		scopeStart := declEnd
		scopeEnd := n - 1
		if isPunctVal(s, declEnd, "{") {
			if e := s.MatchForward(declEnd); e >= 0 {
				scopeEnd = e
			}
		} else {
			for j := declEnd + 1; j < n; j++ {
				if aliasIsNamespaceDeclaration(s, j) {
					scopeEnd = j - 1
					break
				}
			}
		}
		if idx >= scopeStart && idx <= scopeEnd {
			return name.String(), scopeStart, scopeEnd
		}
	}
	return "", 0, n - 1
}

// grafUseMap collects the use imports in [lo,hi], mapping each alias (or last
// segment) to its fully-qualified name.
func grafUseMap(s *tokens.Stream, lo, hi int) map[string]string {
	m := map[string]string{}
	for i := lo; i <= hi && i < s.Len(); i++ {
		if !kwIs(s.At(i), "use") {
			continue
		}
		n := sigNext(s, i)
		if n < 0 || kwIs(s.At(n), "function") || kwIs(s.At(n), "const") {
			continue
		}
		var sb strings.Builder
		alias := ""
		for j := n; j < s.Len(); j++ {
			t := s.At(j)
			if isPunctVal(s, j, ";") || isPunctVal(s, j, "{") || isPunctVal(s, j, "(") {
				break
			}
			if (t.Kind == token.Keyword || t.Kind == token.Ident) && strings.EqualFold(t.Value, "as") {
				if a := sigNext(s, j); a >= 0 {
					alias = s.At(a).Value
				}
				break
			}
			if t.Kind == token.Ident || (t.Kind == token.Punct && t.Value == `\`) {
				sb.WriteString(t.Value)
			}
		}
		fqn := strings.TrimPrefix(sb.String(), `\`)
		if fqn == "" {
			continue
		}
		if alias == "" {
			if k := strings.LastIndex(fqn, `\`); k >= 0 {
				alias = fqn[k+1:]
			} else {
				alias = fqn
			}
		}
		m[alias] = fqn
	}
	return m
}
