package rules

import (
	"regexp"
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// --- AddMissingVarName: "/** @var Type */" with no name -> add the next variable's name ---

type AddMissingVarName struct{}

var varWithoutNameRe = regexp.MustCompile(`^(/\*\* @(?:psalm-|phpstan-)?var )([\\\w|\[\]&-]+)(\s+\*/)$`)

func (AddMissingVarName) Name() string {
	return `Symplify\CodingStandard\Fixer\Commenting\AddMissingVarNameFixer`
}

func (AddMissingVarName) SourceURL() string {
	return "https://github.com/symplify/coding-standard/blob/main/src/Fixer/Commenting/AddMissingVarNameFixer.php"
}

func (AddMissingVarName) Fix(s *tokens.Stream) bool {
	return docBlockFixerApply(s, func(content string, s *tokens.Stream, i int) string {
		m := varWithoutNameRe.FindStringSubmatch(content)
		if m == nil {
			return content
		}
		varTok := addMissingNextVariable(s, i)
		if varTok == "" {
			return content
		}
		return m[1] + m[2] + " " + varTok + m[3]
	})
}

func addMissingNextVariable(s *tokens.Stream, i int) string {
	n := sigNext(s, i)
	if n < 0 || s.At(n).Kind != token.Variable {
		return ""
	}
	if a := sigNext(s, n); a >= 0 && s.At(a).Kind == token.Punct && (s.At(a).Value == "->" || s.At(a).Value == "?->") {
		return ""
	}
	return s.At(n).Value
}

// --- SingleLineInlineVarDocBlock: collapse a multiline inline @var above a variable ---

type SingleLineInlineVarDocBlock struct{}

var (
	singleLineAsteriskStartRe = regexp.MustCompile(`^/\*\s+\*(\s+@(?:psalm-|phpstan-)?var)`)
	singleLineSpaceRe         = regexp.MustCompile(`\s+`)
	singleLineAsteriskLeftRe  = regexp.MustCompile(`(\*\*)(\s+\*)`)
)

func (SingleLineInlineVarDocBlock) Name() string {
	return `Symplify\CodingStandard\Fixer\Commenting\SingleLineInlineVarDocBlockFixer`
}

func (SingleLineInlineVarDocBlock) SourceURL() string {
	return "https://github.com/symplify/coding-standard/blob/main/src/Fixer/Commenting/SingleLineInlineVarDocBlockFixer.php"
}

func (SingleLineInlineVarDocBlock) Fix(s *tokens.Stream) bool {
	return docBlockFixerApply(s, func(content string, s *tokens.Stream, i int) string {
		if !singleLineIsVariableComment(s, i) {
			return content
		}
		if strings.Count(content, "\n") > 2 {
			return content
		}
		content = singleLineAsteriskStartRe.ReplaceAllString(content, "/**$1")
		content = singleLineSpaceRe.ReplaceAllString(content, " ")
		return singleLineAsteriskLeftRe.ReplaceAllString(content, "$1")
	})
}

func singleLineIsVariableComment(s *tokens.Stream, i int) bool {
	n := sigNext(s, i)
	if n < 0 {
		return false
	}
	nn := sigNext(s, n+2)
	if nn < 0 {
		return false
	}
	if s.At(nn).Kind == token.Keyword {
		switch strings.ToLower(s.At(nn).Value) {
		case "static", "function":
			return false
		}
	}
	return s.At(n).Kind == token.Variable
}

// --- RemoveSuperfluousReturnName: "@return Type $var" -> "@return Type" ---

type RemoveSuperfluousReturnName struct{}

var (
	returnVarNameRe   = regexp.MustCompile(`(@(?:psalm-|phpstan-)?return)(\s+[|\\\w]+)?(\s+)(\$[\w]+)`)
	docVariableNameRe = regexp.MustCompile(`\$\w+`)
)

func (RemoveSuperfluousReturnName) Name() string {
	return `Symplify\CodingStandard\Fixer\Commenting\RemoveSuperfluousReturnNameFixer`
}

func (RemoveSuperfluousReturnName) SourceURL() string {
	return "https://github.com/symplify/coding-standard/blob/main/src/Fixer/Commenting/RemoveSuperfluousReturnNameFixer.php"
}

func (RemoveSuperfluousReturnName) Fix(s *tokens.Stream) bool {
	return docBlockFixerApply(s, func(content string, _ *tokens.Stream, _ int) string {
		lines := splitDocLines(content)
		for idx, line := range lines {
			m := returnVarNameRe.FindStringSubmatch(line)
			if m == nil {
				continue
			}
			if m[4] == "$this" {
				continue
			}
			if len(docVariableNameRe.FindAllString(line, -1)) >= 2 {
				continue
			}
			lines[idx] = returnVarNameRe.ReplaceAllString(line, "$1$2")
		}
		return strings.Join(lines, "")
	})
}

// --- RemoveSuperfluousVarName: property "@var Type $prop" -> "@var Type" ($this -> self) ---

type RemoveSuperfluousVarName struct{}

var varVariableNameRe = regexp.MustCompile(`(@(?:psalm-|phpstan-)?var)(\s+[|\\\w]+)?(\s+)(\$[\w]+)`)

func (RemoveSuperfluousVarName) Name() string {
	return `Symplify\CodingStandard\Fixer\Commenting\RemoveSuperfluousVarNameFixer`
}

func (RemoveSuperfluousVarName) SourceURL() string {
	return "https://github.com/symplify/coding-standard/blob/main/src/Fixer/Commenting/RemoveSuperfluousVarNameFixer.php"
}

func (RemoveSuperfluousVarName) Fix(s *tokens.Stream) bool {
	return docBlockFixerApply(s, func(content string, s *tokens.Stream, i int) string {
		n := sigNext(s, i)
		if n < 0 || s.At(n).Kind != token.Keyword {
			return content
		}
		switch strings.ToLower(s.At(n).Value) {
		case "public", "protected", "private", "static":
		default:
			return content
		}
		lines := splitDocLines(content)
		for idx, line := range lines {
			m := varVariableNameRe.FindStringSubmatch(line)
			if m == nil {
				continue
			}
			if strings.HasSuffix(m[4], "$this") && m[4] == "$this" {
				lines[idx] = varVariableNameRe.ReplaceAllString(line, "$1 self")
			} else {
				lines[idx] = varVariableNameRe.ReplaceAllString(line, "$1$2")
			}
		}
		return strings.Join(lines, "")
	})
}
