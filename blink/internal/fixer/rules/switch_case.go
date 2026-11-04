package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

var caseKeywords = map[string]bool{"case": true, "default": true}

// caseTerminator finds the ":" (or ";") that ends the case/default label at i,
// skipping ternary "?:" pairs and "::". It stops at a block brace for safety.
func caseTerminator(s *tokens.Stream, i int) (idx int, isSemicolon, ok bool) {
	ternary := 0
	for k := i + 1; k < s.Len(); k++ {
		t := s.At(k)
		if t.Kind != token.Punct {
			continue
		}
		switch t.Value {
		case "?":
			ternary++
		case ":":
			if ternary > 0 {
				ternary--
			} else {
				return k, false, true
			}
		case ";":
			if ternary == 0 {
				return k, true, true
			}
		case "{", "}":
			return 0, false, false
		}
	}
	return 0, false, false
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ControlStructure/SwitchCaseSemicolonToColonFixer.php
//
// SwitchCaseSemicolonToColon turns a "case 1;" terminator into "case 1:". Enum
// cases are left alone.
type SwitchCaseSemicolonToColon struct{}

func (SwitchCaseSemicolonToColon) Name() string {
	return `PhpCsFixer\Fixer\ControlStructure\SwitchCaseSemicolonToColonFixer`
}

func (SwitchCaseSemicolonToColon) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ControlStructure/SwitchCaseSemicolonToColonFixer.php"
}

// IsCandidate rules out a file without the trigger: only the label of a "case" or "default" is fixed.
func (SwitchCaseSemicolonToColon) IsCandidate(s *tokens.Stream) bool {
	return s.Index().HasAnyWord("case", "default")
}

func (SwitchCaseSemicolonToColon) Fix(s *tokens.Stream) bool {
	changed := false
	for i := range s.Len() {
		t := s.At(i)
		if t.Kind != token.Keyword || !caseKeywords[strings.ToLower(t.Value)] || inClassLikeBody(s, i) {
			continue
		}
		if idx, isSemicolon, ok := caseTerminator(s, i); ok && isSemicolon {
			s.SetValue(idx, ":")
			changed = true
		}
	}
	return changed
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ControlStructure/SwitchCaseSpaceFixer.php
//
// SwitchCaseSpace removes whitespace between a case/default label and its colon
// ("case 1 :" -> "case 1:"), newlines included. Like PHP-CS-Fixer, it skips a
// whole switch whose "default" is not directly followed by whitespace and ":"
// or ";", even a tidy "default:".
type SwitchCaseSpace struct{}

func (SwitchCaseSpace) Name() string {
	return `PhpCsFixer\Fixer\ControlStructure\SwitchCaseSpaceFixer`
}

func (SwitchCaseSpace) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ControlStructure/SwitchCaseSpaceFixer.php"
}

// IsCandidate rules out a file without the trigger: only the labels of a "switch" are fixed.
func (SwitchCaseSpace) IsCandidate(s *tokens.Stream) bool {
	return s.Index().HasWord("switch")
}

func (SwitchCaseSpace) Fix(s *tokens.Stream) bool {
	changed := false
	// tokens are cleared in place (compacted at the end) so indices stay valid
	for i := 0; i < s.Len(); i++ {
		if !kwIs(s.At(i), "switch") {
			continue
		}
		cases, def, ok := switchLabels(s, i)
		if !ok {
			continue
		}
		if def >= 0 {
			if def+2 >= s.Len() || s.At(def+1).Kind != token.Whitespace ||
				(!isPunctVal(s, def+2, ":") && !isPunctVal(s, def+2, ";")) {
				continue
			}
			clearTokenAt(s, def+1)
			changed = true
		}
		for _, c := range cases {
			colon := switchCaseColon(s, c)
			if colon < 0 {
				continue
			}
			v := colon - 1
			for v >= 0 && s.At(v).Kind == token.Whitespace {
				v--
			}
			if v == colon-1 || v < 0 || s.At(v).Kind == token.Comment || s.At(v).Kind == token.DocComment {
				continue
			}
			clearTokenAt(s, v+1)
			changed = true
		}
	}
	if changed {
		s.ClearEmptyTokens()
	}
	return changed
}

// switchLabels returns the "case" keywords and the "default" keyword (-1 when
// none) of the switch at sw, skipping nested switch and match constructs, like
// PHP-CS-Fixer's ControlCaseStructuresAnalyzer.
func switchLabels(s *tokens.Stream, sw int) (cases []int, def int, ok bool) {
	def = -1
	open, end, ok := switchBody(s, sw)
	if !ok {
		return nil, -1, false
	}
	for k := open + 1; k < end; k++ {
		t := s.At(k)
		if t.Kind != token.Keyword {
			continue
		}
		switch strings.ToLower(t.Value) {
		case "switch":
			if _, e, ok := switchBody(s, k); ok {
				k = e
			}
		case "match":
			if p := sigNext(s, k); isPunctVal(s, p, "(") {
				if c := s.MatchForward(p); c >= 0 {
					if b := sigNext(s, c); isPunctVal(s, b, "{") {
						if e := s.MatchForward(b); e >= 0 {
							k = e
						}
					}
				}
			}
		case "case":
			cases = append(cases, k)
		case "default":
			if def < 0 {
				def = k
			}
		}
	}
	return cases, def, true
}

// switchBody returns the index of the "{" or ":" opening the body of the switch
// at sw and of its "}" or "endswitch".
func switchBody(s *tokens.Stream, sw int) (open, end int, ok bool) {
	p := sigNext(s, sw)
	if !isPunctVal(s, p, "(") {
		return 0, 0, false
	}
	c := s.MatchForward(p)
	if c < 0 {
		return 0, 0, false
	}
	open = sigNext(s, c)
	switch {
	case isPunctVal(s, open, "{"):
		if end = s.MatchForward(open); end < 0 {
			return 0, 0, false
		}
		return open, end, true
	case isPunctVal(s, open, ":"):
		depth := 0
		for k := open + 1; k < s.Len(); k++ {
			switch {
			case kwIs(s.At(k), "switch"):
				// a nested braced switch is skipped whole; an alternative one nests
				if _, e, ok := switchBody(s, k); ok && !kwIs(s.At(e), "endswitch") {
					k = e
				} else {
					depth++
				}
			case kwIs(s.At(k), "endswitch"):
				if depth == 0 {
					return open, k, true
				}
				depth--
			}
		}
	}
	return 0, 0, false
}

// switchCaseColon mirrors ControlCaseStructuresAnalyzer::findCaseOpen: from the
// "case" keyword, skip parenthesised and braced blocks, pair each "?" with the
// next ":" or ";", and return the first unpaired ":" or ";".
func switchCaseColon(s *tokens.Stream, c int) int {
	ternaries := 0
	for k := c; k < s.Len(); k++ {
		t := s.At(k)
		if t.Kind != token.Punct {
			continue
		}
		switch t.Value {
		case "(", "{":
			if e := s.MatchForward(k); e >= 0 {
				k = e
				continue
			}
			return -1
		case "?":
			ternaries++
		case ":", ";":
			if ternaries == 0 {
				return k
			}
			ternaries--
		}
	}
	return -1
}
