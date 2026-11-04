package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/StringNotation/HeredocClosingMarkerFixer.php
//
// HeredocClosingMarker unifies heredoc/nowdoc closing markers to a preferred one
// ("EOD" by default), keeping reserved markers (CSS, HTML, SQL, ...) and adding
// "_" suffixes to avoid colliding with the body.
type HeredocClosingMarker struct {
	ClosingMarker        string
	ReservedMarkers      []string
	ExplicitHeredocStyle bool
}

// heredocReservedMarkers is HeredocClosingMarkerFixer::RESERVED_CLOSING_MARKERS.
var heredocReservedMarkers = []string{
	"CSS", "DIFF", "HTML", "JS", "JSON", "MD", "PHP", "PYTHON", "RST", "TS", "SQL", "XML", "YAML",
}

func (HeredocClosingMarker) Name() string {
	return `PhpCsFixer\Fixer\StringNotation\HeredocClosingMarkerFixer`
}

func (HeredocClosingMarker) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/StringNotation/HeredocClosingMarkerFixer.php"
}

func (f HeredocClosingMarker) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["closing_marker"].(string); ok {
		f.ClosingMarker = v
	}
	if v, ok := config["explicit_heredoc_style"].(bool); ok {
		f.ExplicitHeredocStyle = v
	}
	if v, ok := config["reserved_closing_markers"].([]any); ok {
		markers := make([]string, 0, len(v))
		for _, m := range v {
			if s, ok := m.(string); ok {
				markers = append(markers, s)
			}
		}
		f.ReservedMarkers = markers
	}
	return f
}

func (f HeredocClosingMarker) Fix(s *tokens.Stream) bool {
	closingMarker := f.ClosingMarker
	if closingMarker == "" {
		closingMarker = "EOD"
	}
	reserved := f.ReservedMarkers
	if reserved == nil {
		reserved = heredocReservedMarkers
	}
	reservedMap := make(map[string]string, len(reserved))
	for _, v := range reserved {
		reservedMap[strings.ToUpper(v)] = v
	}

	changed := false
	startIndex := -1
	for i := range s.Len() {
		t := s.At(i)
		if t.StartsHeredoc() {
			if t.Part == token.Whole {
				if f.fixWhole(s, i, closingMarker, reservedMap) {
					changed = true
				}
				startIndex = -1
			} else {
				startIndex = i
			}
			continue
		}
		if startIndex >= 0 && t.Part == token.Tail && t.EndsHeredoc() {
			if f.fixInterpolated(s, startIndex, i, closingMarker, reservedMap) {
				changed = true
			}
			startIndex = -1
		}
	}
	return changed
}

func (f HeredocClosingMarker) fixWhole(s *tokens.Stream, i int, defaultMarker string, reservedMap map[string]string) bool {
	v := s.At(i).Value
	a, b, ok := heredocStartMarkerSpan(v)
	if !ok {
		return false
	}
	bodyStart := encLineEnd(v, 0)
	if bodyStart < 0 || bodyStart > len(v) {
		return false
	}
	endMarkerStart := encHeredocEnd(v, bodyStart)
	body := v[bodyStart:endMarkerStart]
	existing := strings.TrimSpace(v[endMarkerStart:])

	newMarker := heredocComputeNewMarker(existing, reservedMap, defaultMarker, body)
	quote := f.markerQuote(v[a] == '\'')

	newVal := v[:a] + quote + newMarker + quote + v[b:endMarkerStart] + heredocRewriteEndMarker(v[endMarkerStart:], newMarker)
	if newVal == v {
		return false
	}
	s.SetValue(i, newVal)
	return true
}

func (f HeredocClosingMarker) fixInterpolated(s *tokens.Stream, start, end int, defaultMarker string, reservedMap map[string]string) bool {
	headVal := s.At(start).Value
	tailVal := s.At(end).Value
	a, b, ok := heredocStartMarkerSpan(headVal)
	if !ok {
		return false
	}
	headBodyStart := encLineEnd(headVal, 0)
	if headBodyStart < 0 {
		return false
	}
	endMarkerStart := encHeredocEnd(tailVal, 0)

	var body strings.Builder
	body.WriteString(headVal[headBodyStart:])
	for j := start + 1; j < end; j++ {
		body.WriteString(s.At(j).Value)
	}
	body.WriteString(tailVal[:endMarkerStart])
	existing := strings.TrimSpace(tailVal[endMarkerStart:])

	newMarker := heredocComputeNewMarker(existing, reservedMap, defaultMarker, body.String())
	quote := f.markerQuote(false) // interpolated heredocs are never nowdocs

	newHead := headVal[:a] + quote + newMarker + quote + headVal[b:]
	newTail := tailVal[:endMarkerStart] + heredocRewriteEndMarker(tailVal[endMarkerStart:], newMarker)
	if newHead == headVal && newTail == tailVal {
		return false
	}
	s.SetValue(start, newHead)
	s.SetValue(end, newTail)
	return true
}

func (f HeredocClosingMarker) markerQuote(isNowdoc bool) string {
	switch {
	case isNowdoc:
		return "'"
	case f.ExplicitHeredocStyle:
		return `"`
	default:
		return ""
	}
}

// heredocStartMarkerSpan returns the byte range [a,b) of the marker following
// "<<<" (and horizontal whitespace) in a heredoc Head/Whole value: the optional
// quote, the label and the optional closing quote. It mirrors the "\K" region of
// '/<<<\h*\K["\']?[^\s"\']+["\']?/'.
func heredocStartMarkerSpan(v string) (a, b int, ok bool) {
	prefix, rest := token.SplitBinaryPrefix(v)
	if !strings.HasPrefix(rest, "<<<") {
		return 0, 0, false
	}
	p := len(prefix) + 3
	for p < len(v) && (v[p] == ' ' || v[p] == '\t') {
		p++
	}
	a = p
	if p < len(v) && (v[p] == '"' || v[p] == '\'') {
		p++
	}
	labelStart := p
	for p < len(v) && v[p] != '"' && v[p] != '\'' && !isRegexSpace(v[p]) {
		p++
	}
	if p == labelStart {
		return 0, 0, false
	}
	if p < len(v) && (v[p] == '"' || v[p] == '\'') {
		p++
	}
	return a, p, true
}

// heredocComputeNewMarker ports HeredocClosingMarkerFixer's marker selection: a
// reserved marker (matched case-insensitively, trailing "_" stripped) is kept in
// its canonical casing, otherwise the preferred marker is used, with "_" appended
// until it no longer collides with the body.
func heredocComputeNewMarker(existing string, reservedMap map[string]string, defaultMarker, body string) string {
	up := strings.ToUpper(existing)
	var newMarker string
	for {
		newMarker = reservedMap[up]
		if newMarker != "" {
			break
		}
		if !strings.HasSuffix(up, "_") {
			break
		}
		up = up[:len(up)-1]
	}
	if newMarker == "" {
		newMarker = defaultMarker
	}
	for heredocMarkerCollides(body, newMarker) {
		newMarker += "_"
	}
	return newMarker
}

// heredocMarkerCollides emulates preg_match('~(^|[\r\n])\s*MARKER(?!\w)~', body):
// the marker appears at the start of a line (only whitespace before it back to a
// line break or the start of body) and is not followed by a word character.
func heredocMarkerCollides(body, marker string) bool {
	for from := 0; ; {
		rel := strings.Index(body[from:], marker)
		if rel < 0 {
			return false
		}
		idx := from + rel
		after := idx + len(marker)
		if after < len(body) && isWordByte(body[after]) {
			from = idx + 1
			continue
		}
		k := idx
		sawBreak := false
		for k > 0 && isRegexSpace(body[k-1]) {
			k--
			if body[k] == '\r' || body[k] == '\n' {
				sawBreak = true
			}
		}
		if k == 0 || sawBreak {
			return true
		}
		from = idx + 1
	}
}

// heredocRewriteEndMarker replaces each run of non-whitespace in the end-marker
// content with the new marker, keeping the indentation (Preg::replace('/\S+/')).
func heredocRewriteEndMarker(end, newMarker string) string {
	var b strings.Builder
	b.Grow(len(end))
	for i := 0; i < len(end); {
		if isRegexSpace(end[i]) {
			b.WriteByte(end[i])
			i++
			continue
		}
		b.WriteString(newMarker)
		for i < len(end) && !isRegexSpace(end[i]) {
			i++
		}
	}
	return b.String()
}

func isRegexSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\v' || c == '\f'
}

func isWordByte(c byte) bool {
	return c == '_' || (c >= '0' && c <= '9') || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}
