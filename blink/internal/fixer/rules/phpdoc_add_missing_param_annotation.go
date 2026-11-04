package rules

import (
	"slices"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/PhpdocAddMissingParamAnnotationFixer.php
//
// PhpdocAddMissingParamAnnotation adds a missing @param line for each function
// argument not already documented. By default (only_untyped) it only adds ones
// for arguments without a native type declaration.
type PhpdocAddMissingParamAnnotation struct {
	onlyUntyped *bool
}

func (PhpdocAddMissingParamAnnotation) Name() string {
	return `PhpCsFixer\Fixer\Phpdoc\PhpdocAddMissingParamAnnotationFixer`
}

func (PhpdocAddMissingParamAnnotation) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/PhpdocAddMissingParamAnnotationFixer.php"
}

func (f PhpdocAddMissingParamAnnotation) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["only_untyped"].(bool); ok {
		f.onlyUntyped = &v
	}
	return f
}

// addParamArg is one parsed function argument.
type addParamArg struct {
	name string // including the leading "$"
	typ  string // native type text ("" when untyped)
	def  string // default value text ("" when none)
}

func (f PhpdocAddMissingParamAnnotation) Fix(s *tokens.Stream) bool {
	onlyUntyped := f.onlyUntyped == nil || *f.onlyUntyped
	changed := false
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind != token.DocComment {
			continue
		}
		content := t.Value
		if strings.Contains(strings.ToLower(content), "inheritdoc") {
			continue
		}
		if !strings.Contains(content, "\n") {
			continue // one-line phpdoc, no place for new annotations
		}

		j := nextMeaningfulIndex(s, i)
		for j >= 0 && addParamIsModifier(s.At(j)) {
			j = nextMeaningfulIndex(s, j)
		}
		if j < 0 || s.At(j).Kind != token.Keyword || !strings.EqualFold(s.At(j).Value, "function") {
			continue
		}
		open := nextSignificantIndex(s, j)
		for open >= 0 && (s.At(open).Kind != token.Punct || s.At(open).Value != "(") {
			open = nextSignificantIndex(s, open)
		}
		if open < 0 {
			continue
		}
		close := s.MatchForward(open)
		if close < 0 {
			continue
		}

		args := parseAddParamArgs(s, open, close)
		wanted := args[:0:0]
		for _, a := range args {
			if !onlyUntyped || a.typ == "" {
				wanted = append(wanted, a)
			}
		}
		if len(wanted) == 0 {
			continue
		}

		lines := splitDocLines(content)
		documented := map[string]bool{}
		lastParamLine := -1
		for _, an := range docAnnotations(lines) {
			if an.name != "param" {
				continue
			}
			if name := addParamFirstVar(strings.Join(lines[an.start:an.end+1], "")); name != "" {
				documented[name] = true
			}
			lastParamLine = max(lastParamLine, an.end)
		}

		lineEnding := "\n"
		if strings.Contains(content, "\r\n") {
			lineEnding = "\r\n"
		}
		var newLines []string
		for _, a := range wanted {
			if documented[a.name] {
				continue
			}
			typ := a.typ
			if typ == "" {
				typ = "mixed"
			}
			if !strings.HasPrefix(typ, "?") && strings.EqualFold(a.def, "null") {
				typ = "null|" + typ
			}
			newLines = append(newLines, indentOf(lines[len(lines)-1])+"* @param "+typ+" "+a.name+lineEnding)
		}
		if len(newLines) == 0 {
			continue
		}

		pos := len(lines) - 1
		if lastParamLine > 0 {
			pos = lastParamLine + 1
		}
		lines = slices.Insert(lines, pos, newLines...)
		s.SetValue(i, strings.Join(lines, ""))
		changed = true
	}
	return changed
}

// addParamIsModifier reports whether a token is a method modifier skipped while
// looking for the "function" keyword.
func addParamIsModifier(t token.Token) bool {
	if t.Kind != token.Keyword {
		return false
	}
	switch strings.ToLower(t.Value) {
	case "abstract", "final", "private", "protected", "public", "static":
		return true
	}
	return false
}

// indentOf returns the leading horizontal whitespace of a line.
func indentOf(line string) string {
	i := 0
	for i < len(line) && (line[i] == ' ' || line[i] == '\t') {
		i++
	}
	return line[:i]
}

// addParamFirstVar ports Preg::match('/^[^$]+(\$\w+).*$/s', ...): the first
// "$name" after at least one non-"$" character.
func addParamFirstVar(content string) string {
	d := strings.IndexByte(content, '$')
	if d <= 0 {
		return ""
	}
	j := d + 1
	for j < len(content) && addParamIsWordByte(content[j]) {
		j++
	}
	if j == d+1 {
		return ""
	}
	return content[d:j]
}

func addParamIsWordByte(b byte) bool {
	return b == '_' || ('0' <= b && b <= '9') || ('a' <= b && b <= 'z') || ('A' <= b && b <= 'Z')
}

// parseAddParamArgs ports ArgumentsAnalyzer::getArguments plus
// prepareArgumentInformation: ordered arguments with name, native type and
// default value text.
func parseAddParamArgs(s *tokens.Stream, open, close int) []addParamArg {
	var args []addParamArg
	cur := addParamArg{}
	sawName := false
	hasTok := false
	depth := 0
	flush := func() {
		if hasTok {
			args = append(args, cur)
		}
		cur = addParamArg{}
		sawName = false
		hasTok = false
	}
	for k := open + 1; k < close; k++ {
		t := s.At(k)
		switch t.Kind {
		case token.AttributeStart:
			if end := s.AttributeEnd(k); end >= 0 {
				k = end
			}
			continue
		case token.Whitespace, token.Comment, token.DocComment:
			continue
		}
		if t.Kind == token.Keyword {
			switch strings.ToLower(t.Value) {
			case "public", "private", "protected", "readonly":
				hasTok = true
				continue
			}
		}
		if t.Kind == token.Punct {
			switch t.Value {
			case "(", "[", "{":
				depth++
			case ")", "]", "}":
				depth--
			case ",":
				if depth == 0 {
					flush()
					continue
				}
			case "=":
				if depth == 0 {
					hasTok = true
					continue
				}
			}
		}
		hasTok = true
		if t.Kind == token.Variable {
			if depth == 0 {
				sawName = true
				cur.name = t.Value
			} else if sawName {
				cur.def += t.Value
			}
			continue
		}
		if sawName {
			cur.def += t.Value
			continue
		}
		if t.Kind == token.Punct && t.Value == "&" {
			continue
		}
		if t.Kind == token.Punct && t.Value == "..." {
			if cur.typ == "" {
				cur.typ = "array"
			} else {
				cur.typ += "[]"
			}
			continue
		}
		cur.typ += t.Value
	}
	flush()
	return args
}
