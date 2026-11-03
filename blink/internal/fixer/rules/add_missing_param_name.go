package rules

import (
	"regexp"
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// Symplify: https://github.com/symplify/coding-standard/blob/main/src/Fixer/Commenting/AddMissingParamNameFixer.php
//
// AddMissingParamName adds the parameter variable name to a "@param Type" tag
// that lacks one. It is a line-for-line port of the PHP fixer, regexes
// included, so it shares its quirks: the function is the next "function"/"fn"
// token after the comment, a name counts as present only when a space precedes
// it, and only a "@param" line whose type has no "{", "<" or "$" is completed.
type AddMissingParamName struct{}

func (AddMissingParamName) Name() string {
	return `Symplify\CodingStandard\Fixer\Commenting\AddMissingParamNameFixer`
}

func (AddMissingParamName) SourceURL() string {
	return "https://github.com/symplify/coding-standard/blob/main/src/Fixer/Commenting/AddMissingParamNameFixer.php"
}

var (
	ampTypeAnnotationRe = regexp.MustCompile(`@(psalm-|phpstan-)?(param|return|var)`)
	ampWithoutNameRe    = regexp.MustCompile(`@param ([^${<]*?)( ([^$]*?))?\n`)
	ampWithNameRe       = regexp.MustCompile(`@param(.*?)\$[\w]+(.*?)\n`)
)

func (AddMissingParamName) Fix(s *tokens.Stream) bool {
	if !ampIsCandidate(s) {
		return false
	}
	changed := false
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind != token.DocComment && t.Kind != token.Comment {
			continue
		}
		if !ampTypeAnnotationRe.MatchString(t.Value) {
			continue
		}
		if out := ampProcessDocContent(s, i, t.Value); out != t.Value {
			// the PHP fixer writes the result back as a T_DOC_COMMENT
			s.Set(i, token.Token{Kind: token.DocComment, Value: out, Pos: t.Pos})
			changed = true
		}
	}
	return changed
}

// ampIsCandidate mirrors AbstractDocBlockFixer::isCandidate: a file holding a
// "callable" token whose third next token is ")" is skipped as a whole.
func ampIsCandidate(s *tokens.Stream) bool {
	hasComment, hasFuncOrVar := false, false
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		switch t.Kind {
		case token.DocComment, token.Comment:
			hasComment = true
		case token.Variable:
			hasFuncOrVar = true
		case token.Keyword:
			switch strings.ToLower(t.Value) {
			case "function":
				hasFuncOrVar = true
			case "callable":
				if i+3 < s.Len() && s.At(i+3).Value == ")" {
					return false
				}
			}
		}
	}
	return hasComment && hasFuncOrVar
}

func ampProcessDocContent(s *tokens.Stream, docIndex int, content string) string {
	names := ampRelatedParamNames(s, docIndex)
	if len(names) == 0 {
		return content
	}
	var missing []string
	for _, name := range names {
		if !regexp.MustCompile(` ` + regexp.QuoteMeta(name) + `\b`).MatchString(content) {
			missing = append(missing, name)
		}
	}
	if len(missing) == 0 {
		return content
	}
	lines := ampDocBlockLines(content)
	for _, name := range missing {
		for k, line := range lines {
			if ampShouldSkipLine(line) {
				continue
			}
			lines[k] = ampNewLineContent(name, line)
			break
		}
	}
	return strings.Join(lines, "")
}

// ampIsFunctionImport reports whether the "function" keyword at i is part of
// an import ("use function f;", "use Foo\{function f}"), which PHP-CS-Fixer's
// ImportTransformer retypes as CT::T_FUNCTION_IMPORT.
func ampIsFunctionImport(s *tokens.Stream, i int) bool {
	if p := prevMeaningfulIndex(s, i); p >= 0 && isKeywordValue(s.At(p), "use") {
		return true
	}
	for j := i + 1; j < s.Len(); j++ {
		t := s.At(j)
		if t.Kind != token.Punct {
			continue
		}
		switch t.Value {
		case "=", "(":
			return false
		case "&":
			if p := prevMeaningfulIndex(s, j); p >= 0 && isKeywordValue(s.At(p), "function", "fn") {
				return false
			}
		case "}":
			if o := s.MatchBackward(j); o >= 0 {
				if p := prevMeaningfulIndex(s, o); p >= 0 && isPunctVal(s, p, `\`) {
					return true
				}
			}
		}
	}
	return false
}

// ampRelatedParamNames resolves the argument names of the next "function" or
// "fn" token (compared case-sensitively, as Tokens::getNextTokenOfKind does)
// after the comment, in FunctionsAnalyzer::getFunctionArguments order.
func ampRelatedParamNames(s *tokens.Stream, docIndex int) []string {
	fn := -1
	for j := docIndex + 1; j < s.Len(); j++ {
		if t := s.At(j); t.Kind == token.Keyword && (t.Value == "function" || t.Value == "fn") {
			if t.Value == "function" && ampIsFunctionImport(s, j) {
				continue
			}
			fn = j
			break
		}
	}
	if fn < 0 {
		return nil
	}
	open := nextPunctOfKind(s, fn, "(")
	if open < 0 {
		return nil
	}
	closeIdx := s.MatchForward(open)
	if closeIdx < 0 {
		return nil
	}
	if nextMeaningfulIndex(s, open) == closeIdx {
		return nil
	}
	var names []string
	seen := map[string]bool{}
	add := func(start, end int) {
		name := ampArgumentName(s, start, end)
		if !seen[name] {
			seen[name] = true
			names = append(names, name)
		}
	}
	start := open + 1
	k := open + 1
	for ; k < closeIdx; k++ {
		t := s.At(k)
		if t.Kind == token.AttributeStart {
			if end := s.AttributeEnd(k); end > k {
				k = end
			}
			continue
		}
		if t.Kind == token.Punct && (t.Value == "(" || t.Value == "[" || t.Value == "{") {
			if end := s.MatchForward(k); end > k {
				k = end
			}
			continue
		}
		if isPunctVal(s, k, ",") {
			if nextMeaningfulIndex(s, k) == closeIdx {
				break // trailing comma
			}
			add(start, k-1)
			start = k + 1
		}
	}
	add(start, k-1)
	return names
}

// ampArgumentName returns the first variable of an argument, skipping its
// attributes and property-hook body; "" when it has none (PHP's null key).
func ampArgumentName(s *tokens.Stream, start, end int) string {
	for k := start; k <= end; k++ {
		t := s.At(k)
		if t.Kind == token.AttributeStart {
			if e := s.AttributeEnd(k); e > k {
				k = e
			}
			continue
		}
		if t.Kind == token.Punct && t.Value == "{" {
			if e := s.MatchForward(k); e > k {
				k = e
			}
			continue
		}
		if t.Kind == token.Variable {
			return t.Value
		}
	}
	return ""
}

// ampDocBlockLines splits a comment the way PhpCsFixer\DocBlock\DocBlock does
// ('/([^\n\r]+\R*)/'): each line keeps its line ending and the blank lines
// that follow it.
func ampDocBlockLines(content string) []string {
	isNewline := func(b byte) bool { return b == '\n' || b == '\r' }
	isR := func(b byte) bool { return isNewline(b) || b == '\v' || b == '\f' || b == 0x85 }
	var lines []string
	for i := 0; i < len(content); {
		j := i
		for j < len(content) && !isNewline(content[j]) {
			j++
		}
		if j == i {
			// line endings before any text stay a piece of their own
			for j < len(content) && isNewline(content[j]) {
				j++
			}
		} else {
			for j < len(content) && isR(content[j]) {
				j++
			}
		}
		lines = append(lines, content[i:j])
		i = j
	}
	return lines
}

func ampShouldSkipLine(line string) bool {
	if !strings.Contains(line, "@param ") {
		return true
	}
	if ampWithNameRe.MatchString(line) {
		return true
	}
	return !ampWithoutNameRe.MatchString(line)
}

func ampNewLineContent(name, line string) string {
	missingDollar := regexp.MustCompile(`(@param\s+([\w\|\[\]\\]+\s)?)(` + regexp.QuoteMeta(strings.TrimLeft(name, "$")) + `)`)
	if missingDollar.MatchString(line) {
		return missingDollar.ReplaceAllString(line, "${1}$$${3}")
	}
	return ampWithoutNameRe.ReplaceAllString(line, "@param ${1} "+strings.ReplaceAll(name, "$", "$$")+"${2}\n")
}
