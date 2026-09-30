package rules

import (
	"regexp"
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// Shared base for Symplify's single-task @param/@return/@var doc-block fixers
// (AbstractDocBlockFixer): iterate every comment/doc-comment carrying a
// param/return/var tag, transform it, and set the result (or drop an emptied
// block). Priority -37 upstream.

var (
	docBlockTypeAnnotationRe = regexp.MustCompile(`@(psalm-|phpstan-)?(param|return|var)`)
	docBlockEmptyRe          = regexp.MustCompile(`/\*\*|\*/|\*|\s`)
)

// docBlockFixerCandidate mirrors AbstractDocBlockFixer::isCandidate.
func docBlockFixerCandidate(s *tokens.Stream) bool {
	hasComment := false
	for i := 0; i < s.Len(); i++ {
		if k := s.At(i).Kind; k == token.Comment || k == token.DocComment {
			hasComment = true
			break
		}
	}
	if !hasComment {
		return false
	}
	for i := 0; i < s.Len(); i++ {
		if s.At(i).Kind == token.Keyword && strings.EqualFold(s.At(i).Value, "callable") {
			if i+3 < s.Len() && s.At(i+3).Kind == token.Punct && s.At(i+3).Value == ")" {
				return false
			}
		}
	}
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind == token.Variable || (t.Kind == token.Keyword && strings.EqualFold(t.Value, "function")) {
			return true
		}
	}
	return false
}

func docBlockIsEmpty(content string) bool {
	return docBlockEmptyRe.ReplaceAllString(content, "") == ""
}

// docBlockFixerApply runs transform over each param/return/var comment, mirroring
// AbstractDocBlockFixer::fix (set as doc comment, or drop an emptied block).
func docBlockFixerApply(s *tokens.Stream, transform func(content string, s *tokens.Stream, i int) string) bool {
	if !docBlockFixerCandidate(s) {
		return false
	}
	changed := false
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind != token.Comment && t.Kind != token.DocComment {
			continue
		}
		if !docBlockTypeAnnotationRe.MatchString(t.Value) {
			continue
		}
		newContent := transform(t.Value, s, i)
		if newContent == t.Value {
			continue
		}
		if docBlockIsEmpty(newContent) {
			followWS := i+1 < s.Len() && s.At(i+1).Kind == token.Whitespace
			s.RemoveAt(i)
			if followWS {
				s.RemoveAt(i)
			}
			i--
			changed = true
			continue
		}
		s.Set(i, token.Token{Kind: token.DocComment, Value: newContent})
		changed = true
	}
	return changed
}

// --- DoubleAsteriskInlineVar: "/* @var" -> "/** @var" on a single-asterisk comment ---

type DoubleAsteriskInlineVar struct{}

var doubleAsteriskStartRe = regexp.MustCompile(`^/\*(\n?\s+@(?:psalm-|phpstan-)?var)`)

func (DoubleAsteriskInlineVar) Name() string {
	return `Symplify\CodingStandard\Fixer\Commenting\DoubleAsteriskInlineVarFixer`
}

func (DoubleAsteriskInlineVar) SourceURL() string {
	return "https://github.com/symplify/coding-standard/blob/main/src/Fixer/Commenting/DoubleAsteriskInlineVarFixer.php"
}

func (DoubleAsteriskInlineVar) Fix(s *tokens.Stream) bool {
	return docBlockFixerApply(s, func(content string, s *tokens.Stream, i int) string {
		if s.At(i).Kind != token.Comment {
			return content
		}
		return doubleAsteriskStartRe.ReplaceAllString(content, "/**$1")
	})
}

// --- FixTagTypo: "@returns/@params/@vars" -> singular ---

type FixTagTypo struct{}

var fixTagTypoRe = regexp.MustCompile(`@((?:psalm-|phpstan-)?(?:param|return|var))s\b`)

func (FixTagTypo) Name() string {
	return `Symplify\CodingStandard\Fixer\Commenting\FixTagTypoFixer`
}

func (FixTagTypo) SourceURL() string {
	return "https://github.com/ecsphp/ecs-src/blob/main/packages/coding-standard/src/Fixer/Commenting/FixTagTypoFixer.php"
}

func (FixTagTypo) Fix(s *tokens.Stream) bool {
	return docBlockFixerApply(s, func(content string, _ *tokens.Stream, _ int) string {
		return fixTagTypoRe.ReplaceAllString(content, "@$1")
	})
}

// --- TypeToVarTag: "@type" -> "@var" (+ single-asterisk to double) ---

type TypeToVarTag struct{}

var (
	// only "@type" at the start of a doc line (after an optional /**, /* or *),
	// never inside a description - mirrors TypeToVarTagFixer #81
	typeTagRe                 = regexp.MustCompile(`(?m)^(\s*(?:/\*{1,2}|\*)?\s*)@type\b`)
	typeToVarSingleAsteriskRe = regexp.MustCompile(`^/\*(\n?\s+@var)`)
)

func (TypeToVarTag) Name() string {
	return `Symplify\CodingStandard\Fixer\Commenting\TypeToVarTagFixer`
}

func (TypeToVarTag) SourceURL() string {
	return "https://github.com/ecsphp/ecs-src/blob/main/packages/coding-standard/src/Fixer/Commenting/TypeToVarTagFixer.php"
}

func (TypeToVarTag) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind != token.Comment && t.Kind != token.DocComment {
			continue
		}
		if !typeTagRe.MatchString(t.Value) {
			continue
		}
		nc := typeTagRe.ReplaceAllString(t.Value, "${1}@var")
		nc = typeToVarSingleAsteriskRe.ReplaceAllString(nc, "/**$1")
		s.Set(i, token.Token{Kind: token.DocComment, Value: nc})
		changed = true
	}
	return changed
}

// --- MergeDocBlockStart: single-asterisk "/*" doc block -> "/**", drop empty "*"-only lines ---

type MergeDocBlockStart struct{}

var (
	mergeDocEmptyAsteriskLineRe = regexp.MustCompile(`^\s*\*\s*$`)
	mergeDocPhpdocTagLineRe     = regexp.MustCompile(`\n\s*\*\s*@`)
	mergeDocOpenerRe            = regexp.MustCompile(`^/\*`)
)

func (MergeDocBlockStart) Name() string {
	return `Symplify\CodingStandard\Fixer\Commenting\MergeDocBlockStartFixer`
}

func (MergeDocBlockStart) SourceURL() string {
	return "https://github.com/ecsphp/ecs-src/blob/main/packages/coding-standard/src/Fixer/Commenting/MergeDocBlockStartFixer.php"
}

func (MergeDocBlockStart) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind != token.Comment {
			continue
		}
		if !strings.HasPrefix(t.Value, "/*") || strings.HasPrefix(t.Value, "/**") {
			continue
		}
		if !mergeDocPhpdocTagLineRe.MatchString(t.Value) {
			continue
		}
		lines := strings.Split(t.Value, "\n")
		kept := lines[:0:0]
		removed := false
		for k, line := range lines {
			if k != 0 && mergeDocEmptyAsteriskLineRe.MatchString(line) {
				removed = true
				continue
			}
			kept = append(kept, line)
		}
		if !removed {
			continue
		}
		kept[0] = mergeDocOpenerRe.ReplaceAllString(kept[0], "/**")
		s.Set(i, token.Token{Kind: token.DocComment, Value: strings.Join(kept, "\n")})
		changed = true
	}
	return changed
}
