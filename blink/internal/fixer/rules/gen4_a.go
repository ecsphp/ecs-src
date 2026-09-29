package rules

import (
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Comment/SingleLineCommentStyleFixer.php
//
// SingleLineCommentStyle normalizes single-line comments to the "//" form. A "#"
// line comment becomes "//" (an "#[" attribute is never touched), and a genuinely
// single-line "/* ... */" block comment at the end of its line becomes "// ...".
// Multi-line "/* */" blocks and doc comments are left alone.
type SingleLineCommentStyle struct{}

func (SingleLineCommentStyle) Name() string {
	return `PhpCsFixer\Fixer\Comment\SingleLineCommentStyleFixer`
}

func (SingleLineCommentStyle) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Comment/SingleLineCommentStyleFixer.php"
}

func (SingleLineCommentStyle) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		// only plain comments (T_COMMENT); doc blocks keep their "/** */" form
		if s.At(i).Kind != token.Comment {
			continue
		}
		v := s.At(i).Value

		// hash style: "# foo" -> "// foo"; "#[...]" is an attribute, not a comment
		if strings.HasPrefix(v, "#") {
			if len(v) >= 2 && v[1] == '[' {
				continue
			}
			s.SetValue(i, "//"+v[1:])
			changed = true
			continue
		}

		// asterisk style: only a genuinely single-line "/* ... */" block
		if !strings.HasPrefix(v, "/*") || len(v) < 4 {
			continue
		}
		if strings.ContainsAny(v, "\n\r") {
			continue // conservative: never touch a multi-line block comment
		}
		content := v[2 : len(v)-2]
		if strings.Contains(content, "?>") {
			continue
		}
		// the comment must sit at the end of its line, otherwise turning it into
		// a "//" comment would swallow the code that follows on the same line
		if next := i + 1; next < s.Len() {
			nt := s.At(next)
			if nt.Kind != token.Whitespace || !strings.ContainsAny(nt.Value, "\n\r") {
				continue
			}
		}

		inner := strings.Trim(content, " \t\r\n\f\v*")
		nv := "//"
		if inner != "" {
			nv = "// " + inner
		}
		s.SetValue(i, nv)
		changed = true

		// drop the leading indentation the following whitespace inherited from the
		// now-removed closing "*/", matching PHP-CS-Fixer
		if next := i + 1; next < s.Len() {
			nt := s.At(next)
			if trimmed := strings.TrimLeft(nt.Value, " \t"); trimmed != nt.Value {
				s.SetValue(next, trimmed)
			}
		}
	}
	return changed
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ClassNotation/NoNullPropertyInitializationFixer.php
//
// NoNullPropertyInitialization removes an explicit "= null" default from an
// untyped class or trait property: "public $x = null;" -> "public $x;". Typed
// properties, constants and parameter defaults are left untouched.
type NoNullPropertyInitialization struct{}

func (NoNullPropertyInitialization) Name() string {
	return `PhpCsFixer\Fixer\ClassNotation\NoNullPropertyInitializationFixer`
}

func (NoNullPropertyInitialization) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ClassNotation/NoNullPropertyInitializationFixer.php"
}

// propertyModifiers are the keywords that may lead a property declaration. A type
// declaration is not among them, so an untyped property is one whose modifiers are
// followed directly by the variable.
var propertyModifiers = map[string]bool{
	"public": true, "protected": true, "private": true,
	"static": true, "var": true, "final": true,
	"abstract": true, "readonly": true,
}

func (NoNullPropertyInitialization) Fix(s *tokens.Stream) bool {
	var remove []int
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind != token.Keyword {
			continue
		}
		// only class and trait bodies hold properties; enums and interfaces do not
		lw := strings.ToLower(t.Value)
		if lw != "class" && lw != "trait" {
			continue
		}
		open := classBodyBrace(s, i)
		if open < 0 {
			continue
		}
		for _, m := range classMemberStarts(s, open) {
			remove = append(remove, nullDefaultsInProperty(s, m)...)
		}
	}
	if len(remove) == 0 {
		return false
	}
	// remove high-to-low so the lower indices stay valid; ranges never overlap
	for a := 0; a < len(remove); a++ {
		for b := a + 1; b < len(remove); b++ {
			if remove[b] > remove[a] {
				remove[a], remove[b] = remove[b], remove[a]
			}
		}
	}
	for _, idx := range remove {
		s.RemoveAt(idx)
	}
	return true
}

// classBodyBrace returns the index of the "{" opening the body of the class-like
// keyword at kw, or -1 when there is none before the statement ends.
func classBodyBrace(s *tokens.Stream, kw int) int {
	for j := kw + 1; j < s.Len(); j++ {
		if s.At(j).Kind != token.Punct {
			continue
		}
		switch s.At(j).Value {
		case "{":
			return j
		case ";":
			return -1
		}
	}
	return -1
}

// nullDefaultsInProperty returns the token indices to drop so that the untyped
// property declaration starting at member loses its "= null" defaults. It returns
// nil for anything that is not an untyped property (a method, const, or a typed
// property whose type sits between the modifiers and the variable).
func nullDefaultsInProperty(s *tokens.Stream, member int) []int {
	i := member
	// skip leading modifier keywords; a type declaration is not a modifier, so
	// reaching a non-modifier that is not the variable means this is not an
	// untyped property (method, const, typed property, ...)
	for i >= 0 && i < s.Len() {
		t := s.At(i)
		if (t.Kind == token.Keyword || t.Kind == token.Ident) && propertyModifiers[strings.ToLower(t.Value)] {
			i = nextMeaningfulIndex(s, i)
			continue
		}
		break
	}
	if i < 0 || s.At(i).Kind != token.Variable {
		return nil
	}

	var remove []int
	for {
		varIdx := i
		eq := nextMeaningfulIndex(s, varIdx)
		if eq < 0 || s.At(eq).Kind != token.Punct || s.At(eq).Value != "=" {
			return remove
		}
		valIdx := nextMeaningfulIndex(s, eq)
		if valIdx >= 0 && s.At(valIdx).Kind == token.Punct && s.At(valIdx).Value == `\` {
			valIdx = nextMeaningfulIndex(s, valIdx)
		}
		if valIdx < 0 || !isNullLiteral(s.At(valIdx)) {
			return remove
		}
		// clear everything between the variable and the null, keeping newline
		// whitespace and comments so surrounding layout survives
		for k := varIdx + 1; k <= valIdx; k++ {
			tk := s.At(k)
			if tk.Kind == token.Comment || tk.Kind == token.DocComment {
				continue
			}
			if tk.Kind == token.Whitespace && strings.Contains(tk.Value, "\n") {
				continue
			}
			remove = append(remove, k)
		}
		after := nextMeaningfulIndex(s, valIdx)
		if after < 0 || s.At(after).Kind != token.Punct || s.At(after).Value != "," {
			return remove
		}
		next := nextMeaningfulIndex(s, after)
		if next < 0 || s.At(next).Kind != token.Variable {
			return remove
		}
		i = next
	}
}

// isNullLiteral reports whether t is the null constant (case-insensitive).
func isNullLiteral(t token.Token) bool {
	return (t.Kind == token.Ident || t.Kind == token.Keyword) && strings.EqualFold(t.Value, "null")
}

// nextMeaningfulIndex returns the index of the first token after i that is not
// whitespace or a comment, or -1 if there is none.
func nextMeaningfulIndex(s *tokens.Stream, i int) int {
	for j := i + 1; j < s.Len(); j++ {
		k := s.At(j).Kind
		if k == token.Whitespace || k == token.Comment || k == token.DocComment {
			continue
		}
		return j
	}
	return -1
}
