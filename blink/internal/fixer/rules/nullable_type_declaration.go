package rules

import (
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/LanguageConstruct/NullableTypeDeclarationFixer.php
//
// NullableTypeDeclaration standardises a nullable single type to the question
// mark syntax (the fixer default): "int|null" and "null|int" become "?int". It
// fires only for a two-member union whose sole other member is a single type
// name, in a confirmed type position (parameter, return or typed property).
// Unions of three or more members, intersections and DNF types are left alone.
type NullableTypeDeclaration struct{}

func (NullableTypeDeclaration) Name() string {
	return `PhpCsFixer\Fixer\LanguageConstruct\NullableTypeDeclarationFixer`
}

func (NullableTypeDeclaration) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/LanguageConstruct/NullableTypeDeclarationFixer.php"
}

func (NullableTypeDeclaration) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind != token.Punct || t.Value != "|" {
			continue
		}
		if !isTypeUnionOperator(s, i) {
			continue
		}
		// significant token indices of the whole type run around this "|"
		start := typeRunBoundaryPrev(s, i)
		end := typeRunBoundaryNext(s, i)
		if end < 0 {
			continue
		}
		var parts []int
		for j := start + 1; j < end; j++ {
			if k := s.At(j).Kind; k == token.Whitespace || k == token.Comment || k == token.DocComment {
				continue
			}
			parts = append(parts, j)
		}
		if len(parts) == 0 {
			continue
		}
		// exactly one "|" and no "&"/"?" -> a plain two-member union
		pipes := 0
		bad := false
		for _, p := range parts {
			v := s.At(p).Value
			if s.At(p).Kind == token.Punct {
				switch v {
				case "|":
					pipes++
				case "&", "?":
					bad = true
				}
			}
		}
		if bad || pipes != 1 {
			continue
		}
		// split members on the single "|"
		var left, right []int
		seen := false
		for _, p := range parts {
			if !seen && s.At(p).Kind == token.Punct && s.At(p).Value == "|" {
				seen = true
				continue
			}
			if seen {
				right = append(right, p)
			} else {
				left = append(left, p)
			}
		}
		// one side must be exactly the single token "null"; keep the other as T
		var typ []int
		switch {
		case isNullMember(s, right):
			typ = left
		case isNullMember(s, left):
			typ = right
		default:
			continue
		}
		if len(typ) == 0 {
			continue
		}
		// rebuild the run as "?T", dropping the null member, the "|" and any
		// internal whitespace; ReplaceRange spans the first..last run token
		repl := []token.Token{{Kind: token.Punct, Value: "?"}}
		for _, p := range typ {
			repl = append(repl, s.At(p))
		}
		s.ReplaceRange(parts[0], parts[len(parts)-1], repl)
		changed = true
		i = parts[0] // continue past the rewritten run
	}
	return changed
}

// isNullMember reports whether idxs is a single token spelling "null".
func isNullMember(s *tokens.Stream, idxs []int) bool {
	return len(idxs) == 1 && strings.EqualFold(s.At(idxs[0]).Value, "null")
}
