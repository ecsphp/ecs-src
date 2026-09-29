package rules

import (
	"strings"

	"blink/internal/fixer"
	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Whitespace/StatementIndentationFixer.php
//
// StatementIndentation reindents statement lines to four spaces per brace level.
// It only touches lines that begin a new statement at brace scope (after ";",
// "{" or "}" and not inside "(...)"/"[...]"), so continuation lines - multiline
// arguments, arrays and method chains - keep their own alignment. Heredoc bodies
// and comment interiors are single tokens and are never reindented.
type StatementIndentation struct {
	// stickComment mirrors stick_comment_to_next_continuous_control_statement: a
	// trailing comment of an if/elseif block before else/elseif dedents to the
	// enclosing level. The zero value (false) keeps the current indentation.
	stickComment bool
}

func (StatementIndentation) Name() string {
	return `PhpCsFixer\Fixer\Whitespace\StatementIndentationFixer`
}

func (StatementIndentation) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Whitespace/StatementIndentationFixer.php"
}

func (f StatementIndentation) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["stick_comment_to_next_continuous_control_statement"].(bool); ok {
		f.stickComment = v
	}
	return f
}

func (f StatementIndentation) Fix(s *tokens.Stream) bool {
	changed := false
	brace, paren := 0, 0
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind == token.Punct {
			switch t.Value {
			case "{":
				brace++
			case "}":
				if brace > 0 {
					brace--
				}
			case "(", "[":
				paren++
			case ")", "]":
				if paren > 0 {
					paren--
				}
			}
		}
		if t.Kind != token.Whitespace || !hasNewline(t.Value) || paren != 0 {
			continue
		}
		prev, ok := prevSignificant(s, i)
		if !ok || (prev.Value != ";" && prev.Value != "{" && prev.Value != "}") {
			continue // continuation line - leave its alignment alone
		}
		if nextSignificantValue(s, i) == "" {
			continue
		}

		level := brace
		if nextSignificantValue(s, i) == "}" {
			level-- // a closing brace dedents to the outer level
		} else if f.stickComment && statementIndentationSticksToNext(s, i) {
			level-- // trailing comment counts as the next control block's comment
		}
		if level < 0 {
			level = 0
		}
		target := strings.Repeat("    ", level)

		v := t.Value
		nl := strings.LastIndexByte(v, '\n')
		if v[nl+1:] != target {
			s.SetValue(i, v[:nl+1]+target)
			changed = true
		}
	}
	return changed
}

// statementIndentationSticksToNext reports whether the line starting after the
// whitespace at i is a comment that is the last content of its block before a "}"
// followed by else/elseif - the case stick_comment... dedents by one level.
func statementIndentationSticksToNext(s *tokens.Stream, i int) bool {
	c := nextSignificantIndex(s, i)
	if c < 0 || s.At(c).Kind != token.Comment {
		return false
	}
	// a comment that is the only content of the block keeps the inner indent
	if p := prevSignificantIndex(s, c); p < 0 || (s.At(p).Kind == token.Punct && s.At(p).Value == "{") {
		return false
	}
	brace := nextSignificantIndex(s, c)
	if brace < 0 || s.At(brace).Kind != token.Punct || s.At(brace).Value != "}" {
		return false
	}
	kw := nextSignificantIndex(s, brace)
	if kw < 0 || s.At(kw).Kind != token.Keyword {
		return false
	}
	lw := strings.ToLower(s.At(kw).Value)
	return lw == "else" || lw == "elseif"
}
