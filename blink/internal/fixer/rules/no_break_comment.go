package rules

import (
	"regexp"
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ControlStructure/NoBreakCommentFixer.php
//
// NoBreakComment adds a "// no break" comment before an intentional fall-through
// case in a switch, and removes it where there is no fall-through.
type NoBreakComment struct{}

const noBreakCommentText = "no break"

var noBreakCommentRe = regexp.MustCompile(`(?i)^((//|#)\s*no break\s*)|(/\*\*?\s*no break(\s+.*)*\*/)$`)

// structureKinds mirrors self::STRUCTURE_KINDS - control structures whose body
// is skipped whole while scanning a case.
var structureKinds = map[string]bool{
	"for": true, "foreach": true, "while": true, "if": true,
	"elseif": true, "switch": true, "function": true, "match": true,
}

func (NoBreakComment) Name() string {
	return `PhpCsFixer\Fixer\ControlStructure\NoBreakCommentFixer`
}

func (NoBreakComment) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ControlStructure/NoBreakCommentFixer.php"
}

func (NoBreakComment) Fix(s *tokens.Stream) bool {
	hasSwitch, hasEnum := false, false
	for i := 0; i < s.Len(); i++ {
		if s.At(i).Kind == token.Keyword {
			switch strings.ToLower(s.At(i).Value) {
			case "switch":
				hasSwitch = true
			case "enum":
				hasEnum = true
			}
		}
	}
	if !hasSwitch {
		return false
	}

	before := s.Render()
	for i := s.Len() - 1; i >= 0; i-- {
		t := s.At(i)
		if kwIs(t, "default") {
			if n := sigNext(s, i); n >= 0 && isPunctVal(s, n, "=>") {
				continue // "default" of a match expression
			}
		} else if !kwIs(t, "case") || isEnumCase(s, i, hasEnum) {
			continue
		}

		colon := nextPunctOfKind(s, i, ":", ";")
		if colon < 0 {
			continue
		}
		fixNoBreakCase(s, colon)
	}
	return s.Render() != before
}

func fixNoBreakCase(s *tokens.Stream, casePosition int) {
	empty := true
	fallThrough := true
	commentPosition := -1

	for i := casePosition + 1; i < s.Len(); i++ {
		t := s.At(i)

		if t.Kind == token.Keyword {
			lv := strings.ToLower(t.Value)
			if structureKinds[lv] || lv == "else" || lv == "do" || lv == "class" {
				empty = false
				i = noBreakStructureEnd(s, i)
				continue
			}
			switch lv {
			case "break", "continue", "return", "exit", "die", "goto":
				fallThrough = false
				continue
			case "throw":
				prev := sigPrev(s, i)
				if prev == casePosition || (prev >= 0 && isThrowStatementStart(s, prev)) {
					fallThrough = false
				}
				continue
			case "endswitch":
				if commentPosition >= 0 {
					removeNoBreakComment(s, commentPosition)
				}
				return
			case "case", "default":
				handleNextCase(s, i, empty, fallThrough, commentPosition)
				return
			}
		}

		if isPunctVal(s, i, "}") {
			if commentPosition >= 0 {
				removeNoBreakComment(s, commentPosition)
			}
			return
		}

		if isNoBreakCommentToken(t) {
			commentPosition = i
			continue
		}

		if t.Kind != token.Comment && t.Kind != token.DocComment && t.Kind != token.Whitespace {
			empty = false
		}
	}
}

// handleNextCase applies the fall-through decision when the scan reaches the
// following case/default.
func handleNextCase(s *tokens.Stream, casePos int, empty, fallThrough bool, commentPosition int) {
	if !empty && fallThrough {
		if commentPosition >= 0 && getPrevNonWhitespace(s, casePos) != commentPosition {
			removeNoBreakComment(s, commentPosition)
			commentPosition = -1
		}
		if commentPosition < 0 {
			insertNoBreakCommentAt(s, casePos)
		} else {
			ensureNewLineAt(s, commentPosition)
		}
		return
	}
	if commentPosition >= 0 {
		removeNoBreakComment(s, commentPosition)
	}
}

func isThrowStatementStart(s *tokens.Stream, prev int) bool {
	if s.At(prev).Kind == token.OpenTag {
		return true
	}
	if s.At(prev).Kind == token.Punct {
		switch s.At(prev).Value {
		case "{", ";", "}":
			return true
		}
	}
	return false
}

func isNoBreakCommentToken(t token.Token) bool {
	if t.Kind != token.Comment && t.Kind != token.DocComment {
		return false
	}
	return noBreakCommentRe.MatchString(t.Value)
}

// noBreakStructureEnd mirrors getStructureEnd: the index of the token closing
// the control structure that starts at position.
func noBreakStructureEnd(s *tokens.Stream, position int) int {
	initial := strings.ToLower(s.At(position).Value)

	if structureKinds[initial] {
		if op := nextPunctOfKind(s, position, "("); op >= 0 {
			position = s.MatchForward(op)
		}
	} else if initial == "class" {
		op := sigNext(s, position)
		if op >= 0 && isPunctVal(s, op, "(") {
			position = s.MatchForward(op)
		}
	}

	if initial == "function" {
		position = nextPunctOfKind(s, position, "{")
	} else {
		position = sigNext(s, position)
	}
	if position < 0 {
		return s.Len() - 1
	}

	if !isPunctVal(s, position, "{") {
		return nextPunctOfKind(s, position, ";")
	}

	position = s.MatchForward(position)

	if initial == "do" {
		if op := nextPunctOfKind(s, position, "("); op >= 0 {
			position = s.MatchForward(op)
		}
		return nextPunctOfKind(s, position, ";")
	}
	return position
}

func insertNoBreakCommentAt(s *tokens.Stream, casePosition int) {
	newlinePosition := ensureNewLineAt(s, casePosition)
	content := s.At(newlinePosition).Value
	nbNewlines := strings.Count(content, "\n")

	if s.At(newlinePosition).Kind == token.OpenTag && hasNewline(content) {
		nbNewlines++
	} else if newlinePosition-1 >= 0 && s.At(newlinePosition-1).Kind == token.OpenTag && hasNewline(s.At(newlinePosition-1).Value) {
		nbNewlines++
		if !hasNewline(content) {
			s.SetValue(newlinePosition, "\n"+content)
			content = s.At(newlinePosition).Value
		}
	}

	if nbNewlines > 1 {
		head, tail := splitTrailingNewline(content)
		indent := detectIndent(s, newlinePosition-1)
		s.SetValue(newlinePosition, head+"\n"+indent)
		newlinePosition++
		s.InsertAt(newlinePosition, token.Token{Kind: token.Whitespace, Value: tail})
	}

	s.InsertAt(newlinePosition, token.Token{Kind: token.Comment, Value: "// " + noBreakCommentText})
	ensureNewLineAt(s, newlinePosition)
}

// ensureNewLineAt guarantees a newline before position and returns the index of
// the newline token.
func ensureNewLineAt(s *tokens.Stream, position int) int {
	content := "\n" + detectIndent(s, position)
	if position-1 < 0 {
		return position
	}
	ws := s.At(position - 1)

	if ws.Kind != token.Whitespace {
		if ws.Kind == token.OpenTag {
			content = stripNewlines(content)
			if !hasNewline(ws.Value) {
				s.SetValue(position-1, replaceTrailingSpace(ws.Value, "\n"))
			}
		}
		if content != "" {
			s.InsertAt(position, token.Token{Kind: token.Whitespace, Value: content})
			return position
		}
		return position - 1
	}

	if position-2 >= 0 && s.At(position-2).Kind == token.OpenTag && hasNewline(s.At(position-2).Value) {
		content = strings.TrimPrefix(content, "\n")
	}
	if !hasNewline(ws.Value) {
		s.SetValue(position-1, content)
	}
	return position - 1
}

func removeNoBreakComment(s *tokens.Stream, commentPosition int) {
	var whitespacePosition int
	prevNW := getPrevNonWhitespace(s, commentPosition)
	afterOpenTag := prevNW >= 0 && s.At(prevNW).Kind == token.OpenTag

	if afterOpenTag {
		whitespacePosition = commentPosition + 1
	} else {
		whitespacePosition = commentPosition - 1
	}

	if whitespacePosition >= 0 && whitespacePosition < s.Len() && s.At(whitespacePosition).Kind == token.Whitespace {
		v := s.At(whitespacePosition).Value
		if afterOpenTag {
			v = stripLeadingNewlineIndent(v)
		} else {
			v = stripTrailingNewlineIndent(v)
		}
		if v == "" {
			s.RemoveAt(whitespacePosition)
			if whitespacePosition < commentPosition {
				commentPosition--
			}
		} else {
			s.SetValue(whitespacePosition, v)
		}
	}

	clearAndMergeWhitespace(s, commentPosition)
}

// isEnumCase reports whether the "case" at caseIndex belongs to an enum.
func isEnumCase(s *tokens.Stream, caseIndex int, hasEnum bool) bool {
	if !hasEnum {
		return false
	}
	prev := caseIndex
	for {
		prev = prevOfKinds(s, prev)
		if prev < 0 {
			return false
		}
		if isPunctVal(s, prev, "}") {
			prev = s.MatchBackward(prev)
			if prev < 0 {
				return false
			}
			continue
		}
		return kwIs(s.At(prev), "enum")
	}
}

// prevOfKinds finds the previous "}", "enum" or "switch" before idx.
func prevOfKinds(s *tokens.Stream, idx int) int {
	for j := idx - 1; j >= 0; j-- {
		if isPunctVal(s, j, "}") {
			return j
		}
		if s.At(j).Kind == token.Keyword {
			switch strings.ToLower(s.At(j).Value) {
			case "enum", "switch":
				return j
			}
		}
	}
	return -1
}

func kwIs(t token.Token, name string) bool {
	return t.Kind == token.Keyword && strings.EqualFold(t.Value, name)
}

func isPunctVal(s *tokens.Stream, i int, v string) bool {
	return i >= 0 && i < s.Len() && s.At(i).Kind == token.Punct && s.At(i).Value == v
}

// nextPunctOfKind scans forward from idx for a punctuation token matching one of
// vals and returns its index, or -1.
func nextPunctOfKind(s *tokens.Stream, idx int, vals ...string) int {
	for j := idx + 1; j < s.Len(); j++ {
		if s.At(j).Kind != token.Punct {
			continue
		}
		for _, v := range vals {
			if s.At(j).Value == v {
				return j
			}
		}
	}
	return -1
}

func getPrevNonWhitespace(s *tokens.Stream, i int) int {
	for j := i - 1; j >= 0; j-- {
		if s.At(j).Kind != token.Whitespace {
			return j
		}
	}
	return -1
}

func prevWhitespace(s *tokens.Stream, i int) int {
	for j := i - 1; j >= 0; j-- {
		if s.At(j).Kind == token.Whitespace {
			return j
		}
	}
	return -1
}

// detectIndent mirrors WhitespacesAnalyzer::detectIndent.
func detectIndent(s *tokens.Stream, index int) string {
	idx := index
	for {
		wi := prevWhitespace(s, idx)
		if wi < 0 {
			return ""
		}
		w := s.At(wi).Value
		if strings.Contains(w, "\n") {
			parts := strings.Split(w, "\n")
			return parts[len(parts)-1]
		}
		if wi-1 >= 0 {
			prev := s.At(wi - 1)
			if (prev.Kind == token.OpenTag || prev.Kind == token.Comment) && strings.HasSuffix(prev.Value, "\n") {
				parts := strings.Split(w, "\n")
				return parts[len(parts)-1]
			}
		}
		idx = wi
	}
}

func splitTrailingNewline(content string) (head, tail string) {
	nl := strings.LastIndexByte(content, '\n')
	if nl < 0 {
		return content, ""
	}
	return content[:nl], content[nl:]
}

func stripNewlines(v string) string {
	return strings.NewReplacer("\r", "", "\n", "").Replace(v)
}

func replaceTrailingSpace(v, repl string) string {
	trimmed := strings.TrimRight(v, " \t\r\n")
	if trimmed == v {
		return v
	}
	return trimmed + repl
}

func stripLeadingNewlineIndent(v string) string {
	i := 0
	if i < len(v) && (v[i] == '\n' || v[i] == '\r') {
		for i < len(v) && (v[i] == '\n' || v[i] == '\r') {
			i++
		}
		for i < len(v) && (v[i] == ' ' || v[i] == '\t') {
			i++
		}
		return v[i:]
	}
	return v
}

func stripTrailingNewlineIndent(v string) string {
	end := len(v)
	j := end
	for j > 0 && (v[j-1] == ' ' || v[j-1] == '\t') {
		j--
	}
	if j > 0 && (v[j-1] == '\n' || v[j-1] == '\r') {
		for j > 0 && (v[j-1] == '\n' || v[j-1] == '\r') {
			j--
		}
		return v[:j]
	}
	return v
}

// clearAndMergeWhitespace removes the token at i and merges surrounding
// whitespace, mirroring Tokens::clearTokenAndMergeSurroundingWhitespace on the
// rendered bytes.
func clearAndMergeWhitespace(s *tokens.Stream, i int) {
	s.RemoveAt(i)
}
