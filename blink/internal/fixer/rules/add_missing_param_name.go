package rules

import (
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// Symplify: https://github.com/symplify/coding-standard/blob/main/src/Fixer/Commenting/AddMissingParamNameFixer.php
//
// AddMissingParamName adds the parameter variable name to a "@param Type" tag
// that lacks one, taken positionally from the documented function's signature.
type AddMissingParamName struct{}

func (AddMissingParamName) Name() string {
	return `Symplify\CodingStandard\Fixer\Commenting\AddMissingParamNameFixer`
}

func (AddMissingParamName) SourceURL() string {
	return "https://github.com/symplify/coding-standard/blob/main/src/Fixer/Commenting/AddMissingParamNameFixer.php"
}

func (AddMissingParamName) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		if s.At(i).Kind != token.DocComment {
			continue
		}
		names := docBlockParamNames(s, i)
		if len(names) == 0 {
			continue
		}
		d, ok := parseDoc(s.At(i).Value)
		if !ok {
			continue
		}
		if addParamNames(&d, names) {
			s.SetValue(i, d.render())
			changed = true
		}
	}
	return changed
}

// docBlockParamNames returns the ordered parameter variable names ("$foo") of the
// function the docblock at index i documents, or nil when it documents none.
func docBlockParamNames(s *tokens.Stream, i int) []string {
	j := i + 1
	for j < s.Len() {
		t := s.At(j)
		switch t.Kind {
		case token.Whitespace, token.Comment:
			j++
			continue
		case token.Keyword:
			if strings.EqualFold(t.Value, "function") {
				return paramNamesAfter(s, j)
			}
			if isFuncModifier(strings.ToLower(t.Value)) || strings.EqualFold(t.Value, "static") || strings.EqualFold(t.Value, "final") || strings.EqualFold(t.Value, "abstract") {
				j++
				continue
			}
			return nil
		default:
			return nil
		}
	}
	return nil
}

func paramNamesAfter(s *tokens.Stream, fn int) []string {
	open := nextSignificantIndex(s, fn)
	for open >= 0 && s.At(open).Kind != token.Punct {
		open = nextSignificantIndex(s, open)
	}
	if open >= 0 && s.At(open).Value == "&" {
		open = nextSignificantIndex(s, open)
	}
	if open < 0 || s.At(open).Value != "(" {
		return nil
	}
	closeIdx := s.MatchForward(open)
	if closeIdx < 0 {
		return nil
	}
	var names []string
	depth := 0
	for k := open + 1; k < closeIdx; k++ {
		if s.At(k).Kind == token.Punct {
			switch s.At(k).Value {
			case "(", "[", "{":
				depth++
			case ")", "]", "}":
				depth--
			}
			continue
		}
		if depth == 0 && s.At(k).Kind == token.Variable {
			names = append(names, s.At(k).Value)
		}
	}
	return names
}

// addParamNames appends the positional parameter name to each "@param <type>"
// line that has no variable name.
func addParamNames(d *docblock, names []string) bool {
	changed := false
	idx := 0
	for i, l := range d.inner {
		trimmed := strings.TrimLeft(l.content, " ")
		if !strings.HasPrefix(strings.ToLower(trimmed), "@param") {
			continue
		}
		rest := trimmed[len("@param"):]
		if rest == "" || (rest[0] != ' ' && rest[0] != '\t') {
			continue // "@paramX"
		}
		fields := strings.Fields(rest)
		pos := idx
		idx++
		if len(fields) != 1 {
			continue // no type, or already has a name / description
		}
		if pos >= len(names) {
			continue
		}
		// A multi-line array-shape type ("@param array{") ends its name on a
		// later line; if that name is already present in the docblock the param
		// is not missing one, so leave it untouched.
		if docBlockContainsName(d, names[pos]) {
			continue
		}
		d.inner[i].content = strings.TrimRight(l.content, " \t") + " " + names[pos]
		changed = true
	}
	return changed
}

// docBlockContainsName reports whether the variable name already appears in any
// docblock line, preceded by whitespace and closed by a non-identifier char.
func docBlockContainsName(d *docblock, name string) bool {
	for _, l := range d.inner {
		s := l.content
		for {
			at := strings.Index(s, name)
			if at < 0 {
				break
			}
			before := at == 0 || s[at-1] == ' ' || s[at-1] == '\t'
			end := at + len(name)
			after := end >= len(s) || !isParamNameChar(s[end])
			if before && after {
				return true
			}
			s = s[at+1:]
		}
	}
	return false
}

func isParamNameChar(b byte) bool {
	return b == '_' || (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9')
}
