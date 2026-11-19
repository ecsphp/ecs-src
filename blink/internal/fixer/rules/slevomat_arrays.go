package rules

import (
	"slices"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/Arrays/TrailingArrayCommaSniff.php
//
// SlevomatTrailingArrayComma adds a comma after the last element of a
// multi-line array, unless it ends on the closing bracket's line. With
// enableAfterHeredoc false, an element ending in a heredoc keeps no comma.
type SlevomatTrailingArrayComma struct {
	disableAfterHeredoc bool
}

func (SlevomatTrailingArrayComma) Name() string {
	return `SlevomatCodingStandard\Sniffs\Arrays\TrailingArrayCommaSniff`
}

func (SlevomatTrailingArrayComma) SourceURL() string {
	return fixer.SourceURLFor(SlevomatTrailingArrayComma{}.Name())
}

func (r SlevomatTrailingArrayComma) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["enableAfterHeredoc"].(bool); ok {
		r.disableAfterHeredoc = !v
	}
	return r
}

func (r SlevomatTrailingArrayComma) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		f.eachToken(f.isArrayToken, func(i int) int {
			open, closeIdx, _ := f.arrayOpenClose(i)
			if f.toks[open].line == f.toks[closeIdx].line {
				return 0
			}
			last := f.prevEffective(closeIdx - 1)
			if last == open || f.is(last, ",") || f.toks[closeIdx].line == f.toks[last].line {
				return 0
			}
			if r.disableAfterHeredoc && f.toks[last].heredoc {
				return 0
			}
			f.beginChangeset()
			f.add(last, ",")
			f.endChangeset()
			return 0
		})
	})
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/Arrays/ArrayAccessSniff.php
//
// SlevomatArrayAccess removes the whitespace between a variable or a closing
// "]" and the "[" of an array access.
type SlevomatArrayAccess struct{}

func (SlevomatArrayAccess) Name() string {
	return `SlevomatCodingStandard\Sniffs\Arrays\ArrayAccessSniff`
}

func (SlevomatArrayAccess) SourceURL() string {
	return fixer.SourceURLFor(SlevomatArrayAccess{}.Name())
}

func (SlevomatArrayAccess) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		isAccess := func(i int) bool { return f.is(i, "[") && !f.isShortArrayOpen(i) }
		f.eachToken(isAccess, func(i int) int {
			prev := f.prevNonWhitespace(i - 1)
			if prev < 0 || prev == i-1 {
				return 0
			}
			closesAccess := f.is(prev, "]") && f.toks[prev].closer >= 0 && !f.isShortArrayOpen(f.toks[prev].closer)
			if f.toks[prev].kind == token.Variable || closesAccess {
				f.beginChangeset()
				f.replace(i-1, "")
				f.endChangeset()
			}
			return 0
		})
	})
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/Arrays/MultiLineArrayEndBracketPlacementSniff.php
//
// SlevomatMultiLineArrayEndBracketPlacement breaks the line after the opening
// bracket of a multi-line array whose first element is an array opened on the
// same line but closed on another line than the parent.
type SlevomatMultiLineArrayEndBracketPlacement struct{}

func (SlevomatMultiLineArrayEndBracketPlacement) Name() string {
	return `SlevomatCodingStandard\Sniffs\Arrays\MultiLineArrayEndBracketPlacementSniff`
}

func (SlevomatMultiLineArrayEndBracketPlacement) SourceURL() string {
	return fixer.SourceURLFor(SlevomatMultiLineArrayEndBracketPlacement{}.Name())
}

func (SlevomatMultiLineArrayEndBracketPlacement) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		f.eachToken(f.isArrayToken, func(i int) int {
			open, closeIdx, _ := f.arrayOpenClose(i)
			if f.toks[open].line == f.toks[closeIdx].line {
				return 0
			}
			next := f.nextEffective(open+1, closeIdx)
			if next < 0 {
				return 0
			}
			nestedOpen, nestedClose, ok := f.arrayOpenClose(next)
			if !ok {
				return 0
			}
			if f.toks[open].line != f.toks[nestedOpen].line || f.toks[closeIdx].line == f.toks[nestedClose].line {
				return 0
			}
			f.add(open, f.eol)
			return 0
		})
	})
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/Arrays/SingleLineArrayWhitespaceSniff.php
//
// SlevomatSingleLineArrayWhitespace normalizes the whitespace of a single-line
// array: spacesAroundBrackets spaces inside the brackets, none before and one
// after each comma, and with enableEmptyArrayCheck none inside an empty array.
// A multi-line array is skipped together with the arrays nested in it.
type SlevomatSingleLineArrayWhitespace struct {
	spacesAroundBrackets  int
	enableEmptyArrayCheck bool
}

func (SlevomatSingleLineArrayWhitespace) Name() string {
	return `SlevomatCodingStandard\Sniffs\Arrays\SingleLineArrayWhitespaceSniff`
}

func (SlevomatSingleLineArrayWhitespace) SourceURL() string {
	return fixer.SourceURLFor(SlevomatSingleLineArrayWhitespace{}.Name())
}

func (r SlevomatSingleLineArrayWhitespace) WithConfig(config map[string]any) fixer.Fixer {
	r.spacesAroundBrackets = configInt(config, "spacesAroundBrackets", r.spacesAroundBrackets)
	if v, ok := config["enableEmptyArrayCheck"].(bool); ok {
		r.enableEmptyArrayCheck = v
	}
	return r
}

func (r SlevomatSingleLineArrayWhitespace) Fix(s *tokens.Stream) bool {
	spaces := strings.Repeat(" ", r.spacesAroundBrackets)
	return slvRun(s, func(f *slvFile) {
		f.eachToken(f.isArrayToken, func(i int) int {
			open, closeIdx, _ := f.arrayOpenClose(i)
			if f.toks[open].line != f.toks[closeIdx].line {
				return closeIdx
			}
			if f.nextNonWhitespace(open+1, closeIdx+1) == closeIdx {
				if r.enableEmptyArrayCheck && closeIdx-open != 1 {
					f.replace(open+1, "")
				}
				return closeIdx + 1
			}

			if l := f.whitespaceLength(open + 1); l != r.spacesAroundBrackets {
				if l == 0 {
					f.add(open, spaces)
				} else {
					f.replace(open+1, spaces)
				}
			}
			if l := f.whitespaceLength(closeIdx - 1); l != r.spacesAroundBrackets {
				if l == 0 {
					f.addBefore(closeIdx, spaces)
				} else {
					f.replace(closeIdx-1, spaces)
				}
			}

			for j := open + 1; j < closeIdx; j++ {
				if f.is(j, "(") && f.toks[j].closer >= 0 {
					j = f.toks[j].closer
					continue
				}
				if _, nestedClose, ok := f.arrayOpenClose(j); ok {
					j = nestedClose
					continue
				}
				if !f.is(j, ",") {
					continue
				}
				if f.nextNonWhitespace(j+1, closeIdx) < 0 {
					return open + 1
				}
				if f.toks[j-1].kind == token.Whitespace && !f.is(j-2, ",") {
					f.replace(j-1, "")
				}
				if f.toks[j+1].kind != token.Whitespace {
					f.add(j, " ")
				} else if f.toks[j+1].length != 1 {
					f.replace(j+1, " ")
				}
			}
			return open + 1
		})
	})
}

func (f *slvFile) whitespaceLength(i int) int {
	if f.toks[i].kind == token.Whitespace {
		return f.toks[i].length
	}
	return 0
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/Arrays/AlphabeticallySortedByKeysSniff.php
//
// SlevomatAlphabeticallySortedByKeys sorts the elements of a multi-line array
// whose elements all have a key (spread elements aside) by key, in natural
// case-insensitive order.
type SlevomatAlphabeticallySortedByKeys struct{}

func (SlevomatAlphabeticallySortedByKeys) Name() string {
	return `SlevomatCodingStandard\Sniffs\Arrays\AlphabeticallySortedByKeysSniff`
}

func (SlevomatAlphabeticallySortedByKeys) SourceURL() string {
	return fixer.SourceURLFor(SlevomatAlphabeticallySortedByKeys{}.Name())
}

func (SlevomatAlphabeticallySortedByKeys) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		f.eachToken(f.isArrayToken, func(i int) int {
			open, closeIdx, _ := f.arrayOpenClose(i)
			if f.toks[open].line == f.toks[closeIdx].line {
				return 0
			}
			kvs := f.parseArray(open, closeIdx)
			for _, kv := range kvs {
				if !kv.unpacking && !kv.hasKey {
					return 0
				}
			}
			if slvSortedByKey(kvs) {
				return 0
			}
			start, end := kvs[0].start, kvs[len(kvs)-1].end
			indent, hasIndent := slvCommonIndent(kvs)
			sorted := slices.Clone(kvs)
			slices.SortStableFunc(sorted, func(a, b slvKeyValue) int { return strnatcasecmp(a.key, b.key) })
			var b strings.Builder
			for _, kv := range sorted {
				b.WriteString(f.keyValueContent(kv, indent, hasIndent))
				b.WriteString(f.eol)
			}
			f.beginChangeset()
			f.change(start, end, b.String())
			f.endChangeset()
			return 0
		})
	})
}

// slvKeyValue is slevomat's ArrayKeyValue: one element of an array.
type slvKeyValue struct {
	start, end int
	key        string
	hasKey     bool
	indent     string
	hasIndent  bool
	comma      int
	unpacking  bool
}

// parseArray is ArrayHelper::parse.
func (f *slvFile) parseArray(open, closeIdx int) []slvKeyValue {
	firstNextLine := f.firstOnNextLine(open + 1)
	firstEffective := f.nextEffective(open+1, len(f.toks))
	start := firstEffective
	if firstNextLine >= 0 && firstNextLine < firstEffective {
		start = firstNextLine
	}
	end := start
	indentation := ""
	if f.toks[open].line < f.toks[firstEffective].line {
		indentation = f.indentation(firstEffective)
	}

	var kvs []slvKeyValue
	for i := start; i < closeIdx; i++ {
		if _, nestedClose, ok := f.arrayOpenClose(i); ok {
			i = nestedClose
			continue
		}
		if sc := f.scopeCloser(i); sc > i {
			i = sc - 1
			continue
		}
		if f.is(i, "(") && f.toks[i].closer > i {
			i = f.toks[i].closer - 1
			continue
		}
		next := f.nextEffective(i+1, len(f.toks))
		if next == closeIdx {
			end = f.valueEnd(i, closeIdx, indentation)
			break
		}
		if !f.is(i, ",") {
			end = i
			continue
		}
		end = f.valueEnd(i, closeIdx, indentation)
		kvs = append(kvs, f.keyValue(start, end))
		start = end + 1
		i = end
	}
	return append(kvs, f.keyValue(start, end))
}

// valueEnd is ArrayHelper::getValueEndPointer: an element owns the comments
// after it on its line and the more indented comment lines below.
func (f *slvFile) valueEnd(end, closeIdx int, indentation string) int {
	next := f.nextEffective(end+1, closeIdx+1)
	if f.toks[next].line == f.toks[end].line {
		return next - 1
	}
	for i := end + 1; i < next; i++ {
		if f.toks[i].line == f.toks[end].line {
			end = i
			continue
		}
		nonWhitespace := f.nextNonWhitespace(i, len(f.toks))
		if f.toks[nonWhitespace].kind != token.Comment {
			break
		}
		if indentation == f.indentation(nonWhitespace) {
			end = i - 1
			break
		}
		i = f.lastOnLine(i)
		end = i
	}
	return end
}

// keyValue is ArrayKeyValue's constructor.
func (f *slvFile) keyValue(start, end int) slvKeyValue {
	kv := slvKeyValue{start: start, end: end, comma: -1}
	arrow := -1
	firstNonWhitespace := -1
	var key strings.Builder
	for i := start; i <= end; i++ {
		if _, nestedClose, ok := f.arrayOpenClose(i); ok {
			i = nestedClose
			continue
		}
		if f.is(i, "=>") && f.isDoubleArrow(i) {
			if f.inClosureBody(i) {
				continue
			}
			arrow = i
			continue
		}
		if f.is(i, ",") {
			kv.comma = i
			continue
		}
		if f.is(i, "...") {
			kv.unpacking = true
			continue
		}
		if arrow >= 0 {
			continue
		}
		if firstNonWhitespace < 0 && f.toks[i].kind != token.Whitespace {
			firstNonWhitespace = i
		}
		if f.toks[i].kind != token.Comment {
			key.WriteString(f.toks[i].content)
		}
	}
	if firstNonWhitespace >= 0 && f.firstNonWhitespaceOnLine(firstNonWhitespace) == firstNonWhitespace {
		kv.indent = f.content(f.firstOnLine(firstNonWhitespace), firstNonWhitespace-1)
		kv.hasIndent = true
	}
	if arrow >= 0 {
		kv.key = strings.Trim(key.String(), " \t\n\r\x00\x0B")
		kv.hasKey = true
	}
	return kv
}

// keyValueContent is ArrayKeyValue::getContent with $normalize true: the
// element with a comma after it, trimmed and re-indented.
func (f *slvFile) keyValueContent(kv slvKeyValue, indent string, hasIndent bool) string {
	addComma := -1
	if kv.comma < 0 {
		addComma = f.prevEffective(kv.end)
	}
	var b strings.Builder
	for i := kv.start; i <= kv.end; i++ {
		b.WriteString(f.toks[i].content)
		if i == addComma {
			b.WriteString(",")
		}
	}
	content := strings.TrimLeft(b.String(), " \t")
	content = strings.TrimRight(content, " \t\n\r\x00\x0B")
	if hasIndent && !strings.HasPrefix(content, f.eol) {
		content = indent + content
	}
	return content
}

// slvCommonIndent is ArrayHelper::getIndentation: the most frequent element
// indentation, the first one seen on a tie.
func slvCommonIndent(kvs []slvKeyValue) (string, bool) {
	type count struct {
		indent    string
		hasIndent bool
		n         int
	}
	var counts []count
	for _, kv := range kvs {
		found := false
		for c := range counts {
			if counts[c].hasIndent == kv.hasIndent && counts[c].indent == kv.indent {
				counts[c].n++
				found = true
				break
			}
		}
		if !found {
			counts = append(counts, count{kv.indent, kv.hasIndent, 1})
		}
	}
	best := counts[0]
	for _, c := range counts[1:] {
		if c.n > best.n {
			best = c
		}
	}
	return best.indent, best.hasIndent
}

// slvSortedByKey is ArrayHelper::isSortedByKey.
func slvSortedByKey(kvs []slvKeyValue) bool {
	previous := ""
	for _, kv := range kvs {
		if kv.unpacking {
			continue
		}
		if strnatcasecmp(previous, kv.key) == 1 {
			return false
		}
		previous = kv.key
	}
	return true
}
