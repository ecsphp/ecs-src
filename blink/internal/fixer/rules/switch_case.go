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
// ("case 1 :" -> "case 1:"), newlines included.
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
	switches, ok := switchAnalyses(s)
	if !ok {
		// the analyzer throws there, so PHP-CS-Fixer leaves the file as it is
		return false
	}
	changed := false
	// tokens are cleared in place (compacted at the end) so indices stay valid
	for _, sw := range switches {
		if def := sw.def; def >= 0 && def+2 < s.Len() && s.At(def+1).Kind == token.Whitespace &&
			(isPunctVal(s, def+2, ":") || isPunctVal(s, def+2, ";")) {
			clearTokenAt(s, def+1)
			changed = true
		}
		for _, c := range sw.cases {
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

// switchAnalysis is a switch as ControlCaseStructuresAnalyzer reports it: its
// "case" keywords and its "default" keyword (-1 when none).
type switchAnalysis struct {
	cases []int
	def   int
}

// switchAnalyses ports ControlCaseStructuresAnalyzer::findControlStructures for
// T_SWITCH: a switch, match or enum level ends once the braces analyzerBrace
// accepts balance out (endswitch for the alternative syntax), so a switch with
// one in its subject ends there and hands its labels to the enclosing level. ok
// is false where the analyzer throws, as on a second default.
func switchAnalyses(s *tokens.Stream) (switches []switchAnalysis, ok bool) {
	type level struct {
		kind   string
		braces int
		alt    bool
		switchAnalysis
	}
	var stack []*level
	for i := 0; i < s.Len(); i++ {
		kw := ""
		if t := s.At(i); t.Kind == token.Keyword {
			kw = strings.ToLower(t.Value)
		}
		if kw == "switch" || kw == "match" || kw == "enum" {
			l := &level{kind: kw, switchAnalysis: switchAnalysis{def: -1}}
			if kw != "enum" {
				c := -1
				if p := sigNext(s, i); isPunctVal(s, p, "(") {
					c = s.MatchForward(p)
				}
				if c < 0 {
					return nil, false
				}
				l.alt = kw == "switch" && isPunctVal(s, sigNext(s, c), ":")
			}
			stack = append(stack, l)
			continue
		}
		if len(stack) == 0 {
			continue
		}
		top := stack[len(stack)-1]
		switch {
		case isPunctVal(s, i, "{") && analyzerBrace(s, i):
			top.braces++
			continue
		case isPunctVal(s, i, "}") && analyzerBrace(s, s.MatchBackward(i)):
			if top.braces--; top.braces != 0 || top.alt {
				continue
			}
		case kw == "endswitch":
			if !top.alt || top.kind != "switch" || top.braces != 0 {
				return nil, false
			}
		default:
			if top.kind != "switch" {
				continue
			}
			switch kw {
			case "case":
				top.cases = append(top.cases, i)
			case "default":
				if top.def >= 0 {
					return nil, false
				}
				top.def = i
			}
			continue
		}
		if top.kind == "switch" {
			switches = append(switches, top.switchAnalysis)
		}
		stack = stack[:len(stack)-1]
	}
	return switches, true
}

// analyzerBrace reports whether the "{" at i is still a "{" for PHP-CS-Fixer's
// analyzers: BraceTransformer retags a dynamic brace and one after "::" whose
// "}" is followed by ";", "?>" or "::". The braces of an interpolation are part
// of their string token here.
func analyzerBrace(s *tokens.Stream, i int) bool {
	if bosIsDynamicBraceOpen(s, i) {
		return false
	}
	p := sigPrev(s, i)
	if isPunctVal(s, p, "::") {
		if e := s.MatchForward(i); e >= 0 {
			n := sigNext(s, e)
			return !isPunctVal(s, n, ";") && !isPunctVal(s, n, "::") && (n < 0 || s.At(n).Kind != token.CloseTag)
		}
	}
	return true
}

// switchCaseColon mirrors ControlCaseStructuresAnalyzer::findCaseOpen: from the
// "case" keyword, skip parenthesised and braced blocks, pair each "?" with the
// next ":" or ";", and return the first unpaired ":" or ";". A nullable type "?"
// and a return type ":" do not count, as PHP-CS-Fixer retags them
// (CT::T_NULLABLE_TYPE, CT::T_TYPE_COLON).
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
			if !isNullableType(s, k) {
				ternaries++
			}
		case ":", ";":
			if isTypeColon(s, k) {
				continue
			}
			if ternaries == 0 {
				return k
			}
			ternaries--
		}
	}
	return -1
}
