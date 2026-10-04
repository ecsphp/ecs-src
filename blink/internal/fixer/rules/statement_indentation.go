package rules

import (
	"slices"
	"strings"

	"blink/internal/fixer"
	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Whitespace/StatementIndentationFixer.php
//
// StatementIndentation reindents every statement and continuation line to four
// spaces per nesting level, mirroring php-cs-fixer's StatementIndentationFixer
// (bracesFixerCompatibility=false, as ECS configures it). It walks a scope stack
// - block, block_signature and statement scopes - and rewrites the leading
// whitespace of each line to the scope's indent, keeping the extra alignment of
// continuation lines inside multiline calls and arrays. Heredoc bodies and
// comment interiors are single tokens and are never reindented.
type StatementIndentation struct {
	// stickComment mirrors stick_comment_to_next_continuous_control_statement: a
	// trailing comment of an if/elseif block before else/elseif dedents to the
	// enclosing level. The zero value (false) keeps the inner indentation.
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

const stmtIndentUnit = "    "

type stmtScope struct {
	kind          string // "block" | "block_signature" | "statement"
	skip          bool
	endIndex      int
	endInclusive  bool
	initialIndent string
	newIndent     string
	indentedBlock bool
}

func (f StatementIndentation) Fix(s *tokens.Stream) bool {
	n := s.Len()
	if n == 0 {
		return false
	}
	endIndex := n - 1
	if s.At(endIndex).Kind == token.Whitespace {
		endIndex--
	}

	lastIndent := stmtExtractIndent(stmtNewLineContent(s, 0))

	scopes := []stmtScope{{
		kind:          "block",
		endIndex:      endIndex,
		endInclusive:  true,
		initialIndent: lastIndent,
		indentedBlock: false,
	}}

	previousLineInitialIndent := ""
	previousLineNewIndent := ""
	noBracesBlockStarts := map[int]bool{}
	caseBlockStarts := map[int]int{}

	changed := false

	for index := range n {
		t := s.At(index)
		cur := len(scopes) - 1

		if noBracesBlockStarts[index] {
			scopes = append(scopes, stmtScope{
				kind:          "block",
				endIndex:      stmtFindStatementEndIndex(s, index, n-1) + 1,
				endInclusive:  true,
				initialIndent: lastIndent,
				indentedBlock: true,
			})
			cur++
		}

		if _, isCase := caseBlockStarts[index]; stmtIsBlockFirst(s, index) || isCase {
			ei := -1
			eiInclusive := true

			switch {
			case t.Kind == token.Keyword && stmtKwIn(t.Value, "extends", "implements"):
				ei = stmtNextValue(s, index, "{")
			case t.Kind == token.Punct && t.Value == ":":
				if _, ok := caseBlockStarts[index]; ok {
					ei, eiInclusive = f.findCaseBlockEnd(s, index)
				}
			case t.Kind == token.Punct && t.Value == "{":
				ei = s.MatchForward(index)
			case t.Kind == token.Punct && t.Value == "(":
				ei = s.MatchForward(index)
			case t.Kind == token.Punct && t.Value == "[": // destructuring target
				ei = s.MatchForward(index)
			}
			if ei < 0 {
				ei = endIndex
			}

			initialIndent := lastIndent
			if scopes[cur].kind == "block_signature" {
				initialIndent = scopes[cur].initialIndent
			}

			scopes = append(scopes, stmtScope{
				kind:          "block",
				endIndex:      ei,
				endInclusive:  eiInclusive,
				initialIndent: initialIndent,
				indentedBlock: true,
			})
			cur++

			for index >= scopes[cur].endIndex {
				scopes = scopes[:len(scopes)-1]
				cur--
			}
			continue
		}

		if stmtIsArrayOpen(s, index) {
			scopes = append(scopes, stmtScope{
				kind:          "statement",
				skip:          true,
				endIndex:      s.MatchForward(index),
				endInclusive:  true,
				initialIndent: previousLineInitialIndent,
				newIndent:     previousLineNewIndent,
				indentedBlock: false,
			})
			continue
		}

		isProp := stmtIsPropertyStart(s, index)
		if isProp || stmtIsBlockSignatureFirst(s, index) {
			lastWhitespaceIndex := -1
			closingParenthesisIndex := -1
			ternaryLevel := 0
			sigEnd := index
			isNoBrace := stmtIsControlWithoutBracesKw(s, index)

			for e := index + 1; e < n; e++ {
				et := s.At(e)
				if et.Kind == token.Punct && et.Value == "(" {
					c := s.MatchForward(e)
					if c < 0 {
						c = e
					}
					closingParenthesisIndex = c
					e = c
					sigEnd = e
					continue
				}
				if et.Kind == token.Punct && et.Value == "[" && isArrayLiteralOpen(s, e) {
					c := s.MatchForward(e)
					if c < 0 {
						c = e
					}
					e = c
					sigEnd = e
					continue
				}
				if et.Kind == token.Punct && (et.Value == "{" || et.Value == ";" || et.Value == "=>") {
					sigEnd = e
					break
				}
				if et.Kind == token.Keyword && stmtKwIn(et.Value, "implements") {
					sigEnd = e
					break
				}
				if et.Kind == token.Punct && et.Value == "?" {
					ternaryLevel++
					sigEnd = e
					continue
				}
				if et.Kind == token.Punct && et.Value == ":" {
					if ternaryLevel > 0 {
						ternaryLevel--
						sigEnd = e
						continue
					}
					if t.Kind == token.Keyword && stmtKwIn(t.Value, "case", "default") {
						caseBlockStarts[e] = index
					}
					sigEnd = e
					break
				}
				if !isNoBrace {
					sigEnd = e
					continue
				}
				if et.Kind == token.Whitespace {
					lastWhitespaceIndex = e
					continue
				}
				if !stmtIsComment(s, e) {
					start := e
					if lastWhitespaceIndex >= 0 {
						start = lastWhitespaceIndex
					}
					noBracesBlockStarts[start] = true
					if closingParenthesisIndex >= 0 {
						sigEnd = closingParenthesisIndex
					} else {
						sigEnd = index
					}
					break
				}
				sigEnd = e
			}

			scopes = append(scopes, stmtScope{
				kind:          "block_signature",
				endIndex:      sigEnd,
				endInclusive:  true,
				initialIndent: lastIndent,
				indentedBlock: isProp || (t.Kind == token.Keyword && stmtKwIn(t.Value, "extends", "implements", "const", "case")),
			})
			continue
		}

		if t.Kind == token.Keyword && stmtKwIn(t.Value, "function") {
			e := index + 1
			for ; e < n; e++ {
				et := s.At(e)
				if et.Kind == token.Punct && et.Value == "(" {
					c := s.MatchForward(e)
					if c < 0 {
						c = e
					}
					e = c
					continue
				}
				if et.Kind == token.Punct && (et.Value == "{" || et.Value == ";") {
					break
				}
			}
			scopes = append(scopes, stmtScope{
				kind:          "block_signature",
				endIndex:      e,
				endInclusive:  true,
				initialIndent: lastIndent,
				indentedBlock: false,
			})
			continue
		}

		if t.Kind == token.Whitespace {
			content := t.Value
			if !hasNewline(content) {
				continue
			}
			nextTok := index + 1

			if scopes[cur].kind == "block" || scopes[cur].kind == "block_signature" {
				indent := false
				if scopes[cur].indentedBlock {
					indent = f.indentDecision(s, index, scopes[cur])
				}
				previousLineInitialIndent = stmtExtractIndent(content)
				var whitespaces string
				if scopes[cur].skip {
					whitespaces = previousLineInitialIndent
				} else {
					whitespaces = scopes[cur].initialIndent
					if indent {
						whitespaces += stmtIndentUnit
					}
				}
				content = stmtReplaceBlockIndent(content, whitespaces)
				previousLineNewIndent = stmtExtractIndent(content)
			} else {
				content = stmtReplaceStatementIndent(content, scopes[cur].initialIndent, scopes[cur].newIndent)
			}

			lastIndent = stmtExtractIndent(content)

			if content != t.Value {
				s.SetValue(index, content)
				changed = true
			}

			if nextTok < n && stmtIsComment(s, nextTok) {
				newComment := stmtReplaceCommentIndent(s.At(nextTok).Value, previousLineInitialIndent, previousLineNewIndent)
				if newComment != s.At(nextTok).Value {
					s.SetValue(nextTok, newComment)
					changed = true
				}
			}
			continue
		}

		for index >= scopes[cur].endIndex {
			scopes = scopes[:len(scopes)-1]
			if len(scopes) == 0 {
				return changed
			}
			cur--
		}

		if stmtIsComment(s, index) ||
			(t.Kind == token.Punct && (t.Value == ";" || t.Value == "," || t.Value == "}")) ||
			t.Kind == token.OpenTag || t.Kind == token.CloseTag {
			continue
		}

		if scopes[cur].kind != "statement" && scopes[cur].kind != "block_signature" {
			se := stmtFindStatementEndIndex(s, index, scopes[cur].endIndex)
			if se == index {
				continue
			}
			scopes = append(scopes, stmtScope{
				kind:          "statement",
				endIndex:      se,
				endInclusive:  false,
				initialIndent: previousLineInitialIndent,
				newIndent:     previousLineNewIndent,
				indentedBlock: true,
			})
		}
	}
	return changed
}

// indentDecision ports the is_indented_block branch: whether the line starting
// after the whitespace at index gains one extra indent level.
func (f StatementIndentation) indentDecision(s *tokens.Stream, index int, sc stmtScope) bool {
	n := s.Len()
	firstNonWS := -1
	nextNewline := -1
	for j := index + 1; j < n; j++ {
		if s.At(j).Kind != token.Whitespace {
			if firstNonWS < 0 {
				firstNonWS = j
			}
			continue
		}
		if hasNewline(s.At(j).Value) {
			nextNewline = j
			break
		}
	}
	end := sc.endIndex
	if !sc.endInclusive {
		end++
	}
	contentBeforeEnd := (firstNonWS >= 0 && firstNonWS < end) || (nextNewline >= 0 && nextNewline < end)
	if !contentBeforeEnd {
		return false
	}
	// comment directly before "}" gets special handling
	if firstNonWS >= 0 && stmtIsPlainComment(s, firstNonWS) {
		nm := nextMeaningfulIndex(s, firstNonWS)
		if nm >= 0 && s.At(nm).Kind == token.Punct && s.At(nm).Value == "}" {
			pm := prevMeaningfulIndex(s, firstNonWS)
			if pm >= 0 && s.At(pm).Kind == token.Punct && s.At(pm).Value == "{" {
				return true
			}
			nn := nextMeaningfulIndex(s, nm)
			if nn >= 0 && s.At(nn).Kind == token.Keyword && stmtKwIn(s.At(nn).Value, "else", "elseif") {
				return !f.stickComment
			}
			return true
		}
	}
	return true
}

// stmtIsBlockFirst reports whether the token at index opens a block scope: "{",
// a destructuring "[", or a "(" that is not an array(...) opener.
func stmtIsBlockFirst(s *tokens.Stream, index int) bool {
	t := s.At(index)
	if t.Kind != token.Punct {
		return false
	}
	switch t.Value {
	case "{":
		return true
	case "[":
		return isDestructuringAssignOpen(s, index)
	case "(":
		p := prevMeaningfulIndex(s, index)
		if p >= 0 && s.At(p).Kind == token.Keyword && strings.EqualFold(s.At(p).Value, "array") {
			return false
		}
		return true
	}
	return false
}

// stmtIsArrayOpen reports whether the token at index opens an array literal that
// forms a statement scope: "array(" or an array-literal "[".
func stmtIsArrayOpen(s *tokens.Stream, index int) bool {
	t := s.At(index)
	if t.Kind != token.Punct {
		return false
	}
	if t.Value == "(" {
		p := prevMeaningfulIndex(s, index)
		return p >= 0 && s.At(p).Kind == token.Keyword && strings.EqualFold(s.At(p).Value, "array")
	}
	if t.Value == "[" {
		return isArrayLiteralOpen(s, index) && !isDestructuringAssignOpen(s, index)
	}
	return false
}

func stmtIsBlockSignatureFirst(s *tokens.Stream, index int) bool {
	t := s.At(index)
	if t.Kind != token.Keyword {
		return false
	}
	if !stmtKwIn(t.Value, "use", "if", "else", "elseif", "for", "foreach", "while",
		"do", "switch", "case", "default", "try", "class", "interface", "trait",
		"extends", "implements", "const", "match", "enum") {
		return false
	}
	// a contextual keyword used as a name ("Enum::X", "new Match", "$o->enum")
	// is an identifier, not a block signature
	return !stmtKeywordIsIdentifierUse(s, index)
}

// stmtKeywordIsIdentifierUse reports whether the keyword at index is actually a
// class/member name reference rather than a language construct.
func stmtKeywordIsIdentifierUse(s *tokens.Stream, index int) bool {
	if p := prevMeaningfulIndex(s, index); p >= 0 {
		pt := s.At(p)
		if pt.Kind == token.Keyword && stmtKwIn(pt.Value, "new", "function", "const", "instanceof") {
			return true
		}
		if pt.Kind == token.Punct && (pt.Value == "::" || pt.Value == "->" || pt.Value == "?->" || pt.Value == `\`) {
			return true
		}
	}
	// a "::" directly after marks a class-name reference ("Enum::X"); a following
	// "\" is a namespace prefix on an operand (e.g. "case \Foo::BAR:") and does not
	// make the keyword itself a name
	if nmi := nextMeaningfulIndex(s, index); nmi >= 0 {
		nt := s.At(nmi)
		if nt.Kind == token.Punct && nt.Value == "::" {
			return true
		}
	}
	return false
}

func stmtIsControlWithoutBracesKw(s *tokens.Stream, index int) bool {
	t := s.At(index)
	return t.Kind == token.Keyword && stmtKwIn(t.Value, "if", "else", "elseif", "for", "foreach", "while", "do")
}

func stmtKwIn(v string, set ...string) bool {
	return slices.Contains(set, strings.ToLower(v))
}

// stmtIsComment reports whether token i is a comment (line/block/doc), excluding
// a folded attribute token.
func stmtIsComment(s *tokens.Stream, i int) bool {
	if i < 0 || i >= s.Len() {
		return false
	}
	t := s.At(i)
	if t.Kind == token.DocComment {
		return true
	}
	return t.Kind == token.Comment && !isAttributeComment(t)
}

// stmtIsPlainComment reports whether token i is a non-doc line/block comment
// (php-cs-fixer T_COMMENT), excluding attributes and doc comments.
func stmtIsPlainComment(s *tokens.Stream, i int) bool {
	if i < 0 || i >= s.Len() {
		return false
	}
	t := s.At(i)
	return t.Kind == token.Comment && !isAttributeComment(t)
}

// stmtNextValue returns the index of the next token equal to value, or -1.
func stmtNextValue(s *tokens.Stream, from int, value string) int {
	for j := from + 1; j < s.Len(); j++ {
		if s.At(j).Kind == token.Punct && s.At(j).Value == value {
			return j
		}
	}
	return -1
}

// stmtNewLineContent mirrors computeNewLineContent: blink's open/close tags carry
// no whitespace, so it is just the token's own content.
func stmtNewLineContent(s *tokens.Stream, index int) string {
	return s.At(index).Value
}

// stmtExtractIndent returns the horizontal whitespace after the last newline.
func stmtExtractIndent(content string) string {
	nl := strings.LastIndexAny(content, "\n\r")
	if nl < 0 {
		return ""
	}
	rest := content[nl+1:]
	i := 0
	for i < len(rest) && (rest[i] == ' ' || rest[i] == '\t') {
		i++
	}
	return rest[:i]
}

// stmtReplaceBlockIndent replaces the trailing "(newlines)(hspaces)" of content
// with the same newlines followed by whitespaces (Preg '/(\R+)\h*$/').
func stmtReplaceBlockIndent(content, whitespaces string) string {
	e := len(content)
	for e > 0 && (content[e-1] == ' ' || content[e-1] == '\t') {
		e--
	}
	return content[:e] + whitespaces
}

// stmtReplaceStatementIndent replaces a trailing "newline + initial + hspaces"
// with "newline + newIndent + hspaces" (Preg '/(\R)INITIAL(\h*)$/D'), preserving
// extra alignment beyond the scope's initial indent.
func stmtReplaceStatementIndent(content, initial, newIndent string) string {
	nl := strings.LastIndexAny(content, "\n\r")
	if nl < 0 {
		return content
	}
	head := content[:nl+1]
	tail := content[nl+1:] // all horizontal whitespace
	if !strings.HasPrefix(tail, initial) {
		return content
	}
	return head + newIndent + tail[len(initial):]
}

// stmtReplaceCommentIndent reindents the continuation lines of a comment token
// whose first line was reindented (Preg '/(\R)INITIAL(\h*\S+.*)/').
func stmtReplaceCommentIndent(comment, initial, newIndent string) string {
	if initial == newIndent || !strings.Contains(comment, "\n") && !strings.Contains(comment, "\r") {
		return comment
	}
	lines := strings.SplitAfter(comment, "\n")
	for i := 1; i < len(lines); i++ {
		line := lines[i]
		// strip the newline suffix handling: line keeps its trailing "\n"
		body := line
		nlSuffix := ""
		if strings.HasSuffix(body, "\n") {
			body = body[:len(body)-1]
			nlSuffix = "\n"
		}
		if strings.HasPrefix(body, initial) {
			rest := body[len(initial):]
			trimmed := strings.TrimLeft(rest, " \t")
			if trimmed != "" { // only lines with meaningful content
				lines[i] = newIndent + rest + nlSuffix
			}
		}
	}
	return strings.Join(lines, "")
}

// stmtFindStatementEndIndex ports findStatementEndIndex: the last meaningful
// token of the statement starting at index, within the parent scope.
func stmtFindStatementEndIndex(s *tokens.Stream, index, parentScopeEndIndex int) int {
	endIndex := -1
	ifLevel := 0
	doWhileLevel := 0
	for se := index; se <= parentScopeEndIndex && se < s.Len(); se++ {
		et := s.At(se)
		if et.Kind == token.Keyword && stmtKwIn(et.Value, "if") {
			p := prevMeaningfulIndex(s, se)
			prevIsElse := p >= 0 && s.At(p).Kind == token.Keyword && stmtKwIn(s.At(p).Value, "else")
			if !prevIsElse {
				ifLevel++
				continue
			}
		}
		if et.Kind == token.Keyword && stmtKwIn(et.Value, "do") {
			doWhileLevel++
			continue
		}
		if et.Kind == token.Punct && (et.Value == "(" || et.Value == "{" || (et.Value == "[" && isArrayLiteralOpen(s, se))) {
			c := s.MatchForward(se)
			if c >= 0 {
				se = c
				et = s.At(se)
			}
		}
		isStatementEnd := (et.Kind == token.Punct && (et.Value == ";" || et.Value == "," || et.Value == "}")) || et.Kind == token.CloseTag
		if !isStatementEnd {
			continue
		}
		cont := nextMeaningfulIndex(s, se)
		if ifLevel > 0 && cont >= 0 && s.At(cont).Kind == token.Keyword && stmtKwIn(s.At(cont).Value, "else", "elseif") {
			if stmtKwIn(s.At(cont).Value, "else") {
				nn := nextMeaningfulIndex(s, cont)
				nextIsIf := nn >= 0 && s.At(nn).Kind == token.Keyword && stmtKwIn(s.At(nn).Value, "if")
				if !nextIsIf {
					ifLevel--
				}
			}
			se = cont
			continue
		}
		if doWhileLevel > 0 && cont >= 0 && s.At(cont).Kind == token.Keyword && stmtKwIn(s.At(cont).Value, "while") {
			doWhileLevel--
			se = cont
			continue
		}
		endIndex = prevSignificantIndex(s, se)
		break
	}
	if endIndex >= 0 {
		return endIndex
	}
	return prevMeaningfulIndex(s, parentScopeEndIndex)
}

// findCaseBlockEnd ports findCaseBlockEnd: the end of a case/default body.
func (f StatementIndentation) findCaseBlockEnd(s *tokens.Stream, index int) (int, bool) {
	n := s.Len()
	for ; index < n; index++ {
		t := s.At(index)
		if t.Kind == token.Keyword && stmtKwIn(t.Value, "switch") {
			op := nextMeaningfulIndex(s, index)
			if op >= 0 && s.At(op).Kind == token.Punct && s.At(op).Value == "(" {
				cp := s.MatchForward(op)
				if cp >= 0 {
					brace := nextMeaningfulIndex(s, cp)
					if brace >= 0 && s.At(brace).Kind == token.Punct && s.At(brace).Value == "{" {
						if be := s.MatchForward(brace); be >= 0 {
							index = be
						}
					}
				}
			}
			continue
		}
		if t.Kind == token.Punct && t.Value == "{" {
			if be := s.MatchForward(index); be >= 0 {
				index = be
			}
			continue
		}
		if t.Kind == token.Keyword && stmtKwIn(t.Value, "case", "default") {
			return index, true
		}
		if t.Kind == token.Punct && t.Value == "}" {
			return prevSignificantIndex(s, index), false
		}
	}
	return n - 1, false
}

// stmtIsPropertyStart ports isPropertyStart: the last modifier before a typed or
// named property declaration.
func stmtIsPropertyStart(s *tokens.Stream, index int) bool {
	ni := nextMeaningfulIndex(s, index)
	if ni < 0 {
		return false
	}
	nt := s.At(ni)
	if nt.Kind == token.Keyword && stmtPropertyKeyword(nt.Value) {
		return false
	}
	if nt.Kind == token.Keyword && stmtKwIn(nt.Value, "const", "function") {
		return false
	}
	// walk back over the modifier chain, tracking a visibility modifier and the
	// token that precedes the chain
	i := index
	foundVisibility := false
	chainStart := index
	for i >= 0 && s.At(i).Kind == token.Keyword && stmtPropertyKeyword(s.At(i).Value) {
		if stmtKwIn(s.At(i).Value, "var", "public", "protected", "private") {
			foundVisibility = true
		}
		chainStart = i
		i = prevMeaningfulIndex(s, i)
	}
	if !foundVisibility {
		return false
	}
	// a promoted constructor parameter ("(private A $a, protected B $b)") is not a
	// class property: its modifier chain is preceded by "(" or ","
	before := prevMeaningfulIndex(s, chainStart)
	if before >= 0 && s.At(before).Kind == token.Punct && (s.At(before).Value == "(" || s.At(before).Value == ",") {
		return false
	}
	return true
}

func stmtPropertyKeyword(v string) bool {
	return stmtKwIn(v, "var", "public", "protected", "private", "static", "readonly")
}
