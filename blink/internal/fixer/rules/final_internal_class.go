package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ClassNotation/FinalInternalClassFixer.php
//
// FinalInternalClass marks internal classes final (risky). "Internal" is decided
// by class-level annotations and attributes: a class is fixed when an include tag
// is present (or the include list is empty) and no exclude tag is. The defaults
// target "@internal" and skip Doctrine entities/documents.
type FinalInternalClass struct {
	include        map[string]bool // nil => default include ("internal")
	exclude        map[string]bool // nil => default exclude (Doctrine, final)
	considerAbsent bool            // fix classes without any docblock/attribute
	noAttributes   bool            // deprecated annotation_* mode: ignore attributes
}

// ficDefaultInclude/ficDefaultExclude are DEFAULTS normalized (lowercased, "@"
// stripped).
var (
	ficDefaultInclude = map[string]bool{"internal": true}
	ficDefaultExclude = map[string]bool{
		"final": true, "entity": true, "orm\\entity": true, "orm\\mapping\\entity": true,
		"mapping\\entity": true, "document": true, "odm\\document": true,
	}
)

func (FinalInternalClass) Name() string {
	return `PhpCsFixer\Fixer\ClassNotation\FinalInternalClassFixer`
}

func (FinalInternalClass) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ClassNotation/FinalInternalClassFixer.php"
}

func (f FinalInternalClass) WithConfig(config map[string]any) fixer.Fixer {
	if list, ok := generalPhpdocAnnotationStringList(config["annotation_include"]); ok {
		f.include = ficNormalizeTags(list)
		f.noAttributes = true
	} else if list, ok := generalPhpdocAnnotationStringList(config["include"]); ok {
		f.include = ficNormalizeTags(list)
	}
	if list, ok := generalPhpdocAnnotationStringList(config["annotation_exclude"]); ok {
		f.exclude = ficNormalizeTags(list)
		f.noAttributes = true
	} else if list, ok := generalPhpdocAnnotationStringList(config["exclude"]); ok {
		f.exclude = ficNormalizeTags(list)
	}
	if v, ok := config["consider_absent_docblock_as_internal_class"].(bool); ok {
		f.considerAbsent = v
	}
	return f
}

func ficNormalizeTags(list []string) map[string]bool {
	out := map[string]bool{}
	for _, key := range list {
		key = strings.TrimPrefix(key, "@")
		out[strings.ToLower(key)] = true
	}
	return out
}

func (f FinalInternalClass) effInclude() map[string]bool {
	if f.include != nil {
		return f.include
	}
	return ficDefaultInclude
}

func (f FinalInternalClass) effExclude() map[string]bool {
	if f.exclude != nil {
		return f.exclude
	}
	return ficDefaultExclude
}

func (f FinalInternalClass) Fix(s *tokens.Stream) bool {
	changed := false
	for index := s.Len() - 1; index >= 0; index-- {
		if !taIsClassKw(s, index) || !f.isClassCandidate(s, index) {
			continue
		}
		s.InsertSliceAt(index, []token.Token{
			{Kind: token.Keyword, Value: "final"},
			{Kind: token.Whitespace, Value: " "},
		})
		changed = true
	}
	return changed
}

// ficDecision is a tri-state: a docblock/attribute votes yes, no, or abstains.
type ficDecision int

const (
	ficAbstain ficDecision = iota
	ficYes
	ficNo
)

func (f FinalInternalClass) isClassCandidate(s *tokens.Stream, index int) bool {
	if taIsAnonymousClass(s, index) {
		return false
	}
	if hasFinal, hasAbstract := ficClassModifiers(s, index); hasFinal || hasAbstract {
		return false
	}

	var decisions []ficDecision
	cur := index
	for cur >= 0 {
		cur = getPrevNonWhitespace(s, cur)
		if cur < 0 {
			break
		}
		t := s.At(cur)
		accept := t.Kind == token.AttributeEnd || t.Kind == token.DocComment ||
			t.Kind == token.Comment || kwIs(t, "readonly")
		if !accept {
			break
		}
		if !f.noAttributes && t.Kind == token.AttributeEnd {
			start := s.AttributeStartOf(cur)
			if start < 0 {
				break
			}
			decisions = append(decisions, f.attributeDecision(s, start, cur))
			cur = start
		}
		if s.At(cur).Kind == token.DocComment {
			decisions = append(decisions, f.phpDocDecision(s, cur))
		}
	}

	hasYes := false
	for _, d := range decisions {
		if d == ficNo {
			return false
		}
		if d == ficYes {
			hasYes = true
		}
	}
	return hasYes || (len(decisions) == 0 && f.considerAbsent)
}

// ficClassModifiers reports whether the class keyword is preceded by final or
// abstract (in its modifier run of final/abstract/readonly).
func ficClassModifiers(s *tokens.Stream, classKw int) (hasFinal, hasAbstract bool) {
	for j := sigPrev(s, classKw); j >= 0; j = sigPrev(s, j) {
		switch {
		case kwIs(s.At(j), "final"):
			hasFinal = true
		case kwIs(s.At(j), "abstract"):
			hasAbstract = true
		case kwIs(s.At(j), "readonly"):
		default:
			return
		}
	}
	return
}

func (f FinalInternalClass) phpDocDecision(s *tokens.Stream, index int) ficDecision {
	d, ok := parseDoc(s.At(index).Value)
	if !ok {
		return ficAbstain
	}
	tags := map[string]bool{}
	for _, l := range d.inner {
		if tag := ficTagOf(l.content); tag != "" {
			tags[tag] = true
		}
	}
	return f.decide(tags)
}

// ficTagOf extracts the lowercased tag of a docblock line ("@ORM\Entity(x)" ->
// "orm\entity"), or "" when the line is not a tag.
func ficTagOf(content string) string {
	trimmed := strings.TrimLeft(content, " \t")
	if !strings.HasPrefix(trimmed, "@") {
		return ""
	}
	j := 1
	for j < len(trimmed) && trimmed[j] != '(' && trimmed[j] != ' ' && trimmed[j] != '\t' {
		j++
	}
	return strings.ToLower(trimmed[1:j])
}

func (f FinalInternalClass) attributeDecision(s *tokens.Stream, startIndex, endIndex int) ficDecision {
	candidates := map[string]bool{}
	str := ""
	cur := startIndex
	for cur < endIndex {
		cur = sigNext(s, cur)
		if cur < 0 {
			break
		}
		t := s.At(cur)
		if t.Kind == token.Ident || (t.Kind == token.Punct && t.Value == `\`) {
			str += strings.ToLower(t.Value)
		} else if str != "" {
			candidates[str] = true
			str = ""
		}
	}
	if str != "" {
		candidates[str] = true
	}
	return f.decide(candidates)
}

// decide applies the exclude-then-include rule to a tag set.
func (f FinalInternalClass) decide(tags map[string]bool) ficDecision {
	exclude := f.effExclude()
	for tag := range tags {
		if exclude[tag] {
			return ficNo
		}
	}
	if f.isConfiguredAsInclude(tags) {
		return ficYes
	}
	return ficAbstain
}

func (f FinalInternalClass) isConfiguredAsInclude(tags map[string]bool) bool {
	include := f.effInclude()
	if len(include) == 0 {
		return true
	}
	for tag := range tags {
		if include[tag] {
			return true
		}
	}
	return false
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ClassNotation/FinalClassFixer.php
//
// FinalClass marks every non-abstract, non-Doctrine class final (risky). It is a
// proxy for FinalInternalClass configured with an empty include list and absent
// docblocks treated as internal.
type FinalClass struct{}

func (FinalClass) Name() string {
	return `PhpCsFixer\Fixer\ClassNotation\FinalClassFixer`
}

func (FinalClass) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ClassNotation/FinalClassFixer.php"
}

func (FinalClass) Fix(s *tokens.Stream) bool {
	inner := FinalInternalClass{include: map[string]bool{}, considerAbsent: true}
	return inner.Fix(s)
}
