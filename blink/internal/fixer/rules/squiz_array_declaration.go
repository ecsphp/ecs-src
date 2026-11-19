package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/Arrays/ArrayDeclarationSniff.php
//
// SquizArrayDeclaration lays out array declarations: a single-line array
// keeps one value, a multi-line one puts each value on its own line ending in
// a comma, with keys, double arrows and values aligned.
type SquizArrayDeclaration struct{}

func (SquizArrayDeclaration) Name() string {
	return `PHP_CodeSniffer\Standards\Squiz\Sniffs\Arrays\ArrayDeclarationSniff`
}

func (SquizArrayDeclaration) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/Arrays/ArrayDeclarationSniff.php"
}

func (SquizArrayDeclaration) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		a := &squizArrays{f: f, sc: newPhpcsScope(f)}
		for i := range f.toks {
			if a.isArray(i) || f.isShortArrayOpen(i) {
				a.process(i)
			}
		}
	})
}

type squizArrays struct {
	f  *slvFile
	sc *phpcsScope
}

type squizArrayEntry struct {
	value, index, arrow int // -1 when unset
	indexLength         int
}

// isArray tells T_ARRAY: the "array" keyword that owns parentheses.
func (a *squizArrays) isArray(i int) bool {
	return a.f.isKeyword(i, "array") && a.sc.pCloser[i] >= 0
}

func (a *squizArrays) depth(i int) int {
	n := 0
	for p := a.sc.nested[i]; p >= 0; p = a.sc.nested[p] {
		n++
	}
	return n
}

// nextNot is File::findNext with exclude=true: the first token in
// [start, end) that skip rejects, or -1.
func (a *squizArrays) nextNot(skip func(int) bool, start, end int) int {
	end = min(end, len(a.f.toks))
	for i := max(start, 0); i < end; i++ {
		if !skip(i) {
			return i
		}
	}
	return -1
}

// prevNot is File::findPrevious with exclude=true: the first token in
// [end, start] going down that skip rejects, or -1.
func (a *squizArrays) prevNot(skip func(int) bool, start, end int) int {
	for i := min(start, len(a.f.toks)-1); i >= max(end, 0); i-- {
		if !skip(i) {
			return i
		}
	}
	return -1
}

func (a *squizArrays) isCast(i int) bool {
	t := a.f.toks[i]
	return t.kind == token.Punct && len(t.orig) > 1 && t.orig[0] == '('
}

func (a *squizArrays) isClosureOrFn(i int) bool {
	return a.f.isKeyword(i, "function") && a.f.isClosure(i) || a.sc.code[i] == "fn"
}

func (a *squizArrays) isEndHeredoc(i int) bool { return a.sc.code[i] == "end_heredoc" }

func (a *squizArrays) process(stackPtr int) {
	f, sc := a.f, a.sc
	if f.is(stackPtr, "[") {
		if p := sc.nested[stackPtr]; p >= 0 {
			if owner := sc.pOwner[p]; owner >= 0 && sc.code[owner] == "foreach" {
				closer := f.toks[p].closer
				for j := p + 1; j < closer; j++ {
					if f.isKeyword(j, "as") {
						if j < stackPtr {
							return
						}
						break
					}
				}
			}
		}
	}

	var arrayStart, arrayEnd int
	if !f.is(stackPtr, "[") {
		if f.toks[stackPtr].orig != strings.ToLower(f.toks[stackPtr].orig) {
			f.replaceToken(stackPtr, "array")
		}
		arrayEnd = sc.pCloser[stackPtr]
		arrayStart = f.toks[arrayEnd].closer
		if arrayStart != stackPtr+1 {
			next := a.nextNot(f.isWS, stackPtr+1, arrayStart)
			if next < 0 || !f.isCommentToken(next) {
				f.beginChangeset()
				for i := stackPtr + 1; i < arrayStart; i++ {
					f.replaceToken(i, "")
				}
				f.endChangeset()
			}
		}
	} else {
		arrayStart, arrayEnd = stackPtr, f.toks[stackPtr].closer
	}

	if a.nextNot(f.isWS, arrayStart+1, arrayEnd+1) == arrayEnd {
		if arrayEnd-arrayStart != 1 {
			f.beginChangeset()
			for i := arrayStart + 1; i < arrayEnd; i++ {
				f.replaceToken(i, "")
			}
			f.endChangeset()
		}
		return
	}

	if f.toks[arrayStart].line == f.toks[arrayEnd].line {
		a.singleLine(stackPtr, arrayStart, arrayEnd)
	} else {
		a.multiLine(stackPtr, arrayStart, arrayEnd)
	}
}

func (a *squizArrays) singleLine(stackPtr, arrayStart, arrayEnd int) {
	f, sc := a.f, a.sc
	valueCount := 0
	var commas []int
	for i := arrayStart + 1; i < arrayEnd; i++ {
		if f.is(i, "(") {
			if f.toks[i].closer > i {
				i = f.toks[i].closer
			}
			continue
		}
		if f.is(i, ",") {
			if a.nextNot(f.isWS, i+1, arrayEnd) >= 0 {
				valueCount++
				commas = append(commas, i)
			} else {
				f.replaceToken(i, "")
			}
		}
	}

	for arrow := arrayStart + 1; arrow < arrayEnd; arrow++ {
		if sc.code[arrow] != "=>" {
			continue
		}
		if !f.isWS(arrow - 1) {
			f.addContentBefore(arrow, " ")
		} else if f.toks[arrow-1].length != 1 {
			f.replaceToken(arrow-1, " ")
		}
		if !f.isWS(arrow + 1) {
			f.addContent(arrow, " ")
		} else if f.toks[arrow+1].length != 1 {
			f.replaceToken(arrow+1, " ")
		}
	}

	if valueCount == 0 {
		return
	}
	p := sc.nested[stackPtr]
	if p < 0 || f.toks[f.toks[p].closer].line != f.toks[stackPtr].line {
		f.beginChangeset()
		f.addContent(arrayStart, f.eol)
		if f.isWS(arrayEnd - 1) {
			f.replaceToken(arrayEnd-1, f.eol)
		} else {
			f.addContentBefore(arrayEnd, f.eol)
		}
		f.endChangeset()
		return
	}
	for _, comma := range commas {
		if !f.isWS(comma + 1) {
			f.addContent(comma, " ")
		} else if f.toks[comma+1].length != 1 {
			f.replaceToken(comma+1, " ")
		}
		if f.isWS(comma - 1) {
			f.replaceToken(comma-1, "")
		}
	}
}

func (a *squizArrays) multiLine(stackPtr, arrayStart, arrayEnd int) {
	f, sc := a.f, a.sc
	col := func(i int) int { return f.toks[i].column }
	line := func(i int) int { return f.toks[i].line }
	spaces := func(n int) string { return strings.Repeat(" ", max(n, 0)) }
	keywordStart := col(stackPtr)

	lastContent := a.prevNot(f.isWS, arrayEnd-1, arrayStart)
	if line(lastContent) == line(arrayEnd) {
		f.addContentBefore(arrayEnd, f.eol)
	} else if col(arrayEnd) != keywordStart {
		expected, found := keywordStart-1, col(arrayEnd)-1
		if found == 0 {
			f.addContent(arrayEnd-1, spaces(expected))
		} else {
			f.replaceToken(arrayEnd-1, spaces(expected))
		}
	}

	keyUsed, singleUsed := false, false
	var indices []squizArrayEntry
	maxLength := 0
	lastToken := stackPtr
	if !f.is(stackPtr, "[") {
		lastToken = arrayStart
	}

	for next := stackPtr + 1; next < arrayEnd; next++ {
		if f.is(next, "(") && sc.pOwner[next] != stackPtr {
			if f.toks[next].closer > next {
				next = f.toks[next].closer
			}
			continue
		}

		if a.isArray(next) || f.isShortArrayOpen(next) || a.isClosureOrFn(next) || sc.code[next] == "match" {
			if sc.code[lastToken] != "=>" {
				indices = append(indices, squizArrayEntry{value: next, index: -1, arrow: -1})
				lastToken = next
			}
			switch {
			case a.isArray(next):
				next = sc.pCloser[next]
			case f.is(next, "["):
				next = f.toks[next].closer
			default:
				if sc.closer[next] < 0 {
					continue
				}
				next = sc.closer[next]
			}
			next = a.nextNot(f.isWS, next+1, len(f.toks))
			if next < 0 {
				break
			}
			if !f.is(next, ",") {
				next--
			} else {
				lastToken = next
			}
			continue
		}

		if sc.code[next] != "=>" && !f.is(next, ",") {
			continue
		}

		if f.is(next, ",") {
			stackPtrCount := a.depth(stackPtr)
			commaCount := a.depth(next)
			if commaCount > 0 && !f.is(stackPtr, "[") {
				commaCount--
			}
			if commaCount > stackPtrCount {
				continue
			}
			if keyUsed && f.is(lastToken, ",") {
				next = a.nextNot(f.isEmpty, lastToken+1, len(f.toks))
				if next < 0 || !f.is(next, "...") {
					return
				}
			}
			if !keyUsed {
				if f.isWS(next - 1) {
					prev := a.prevNot(f.isEmpty, next-1, 0)
					if !a.isEndHeredoc(prev) || line(next-1) == line(next) {
						if prev == a.prevNot(f.isWS, next-1, 0) {
							f.replaceToken(next-1, "")
						}
					}
				}
				valueContent := a.nextNot(f.isEmpty, lastToken+1, next)
				indices = append(indices, squizArrayEntry{value: valueContent, index: -1, arrow: -1})
				unpack := a.prevNot(f.isEmpty, next-2, 0)
				if unpack < 0 || !f.is(unpack, "...") {
					singleUsed = true
				}
			}
			lastToken = next
			continue
		}

		if singleUsed {
			return
		}
		keyUsed = true
		indexEnd := a.prevNot(f.isWS, next-1, arrayStart)
		indexStart := sc.startOfStatement(indexEnd)
		entry := squizArrayEntry{arrow: next, index: indexEnd, indexLength: f.toks[indexEnd].length}
		if indexStart != indexEnd {
			entry.index, entry.indexLength = indexStart, 0
			for i := indexStart; i <= indexEnd; i++ {
				entry.indexLength += f.toks[i].length
			}
		}
		maxLength = max(maxLength, entry.indexLength)
		entry.value = a.nextNot(f.isEmpty, next+1, arrayEnd)
		indices = append(indices, entry)
		lastToken = next
	}

	singleValue := false
	if len(indices) == 0 {
		singleValue = true
	} else if len(indices) == 1 && f.is(lastToken, ",") {
		skip := func(i int) bool { return f.isEmpty(i) || f.is(i, ",") }
		if a.nextNot(skip, max(indices[0].value, 0)+1, arrayEnd) < 0 {
			singleValue = true
		}
	}
	if singleValue && a.nextNot(func(i int) bool { return !f.toks[i].heredoc }, arrayStart+1, arrayEnd-1) < 0 {
		notComment := func(i int) bool { return f.toks[i].kind != token.Comment && !phpcsIsAnnotation(f, i) }
		if a.nextNot(notComment, arrayStart+1, arrayEnd) < 0 {
			f.beginChangeset()
			for i := arrayStart + 1; i < arrayEnd && f.isWS(i); i++ {
				f.replaceToken(i, "")
			}
			for i := arrayEnd - 1; i > arrayStart && f.isWS(i); i-- {
				f.replaceToken(i, "")
			}
			f.endChangeset()
		}
		return
	}

	if !keyUsed && len(indices) > 0 {
		lastIndex := max(indices[len(indices)-1].value, 0)
		trailing := max(a.prevNot(f.isEmpty, arrayEnd-1, lastIndex), 0)
		if !f.is(trailing, ",") {
			f.addContent(trailing, ",")
		}
		ignore := func(i int) bool { return f.isWS(i) || f.is(i, ",") || a.isCast(i) }
		for _, v := range indices {
			valuePointer := v.value
			if valuePointer <= 0 {
				continue
			}
			if a.isClosureOrFn(valuePointer) {
				if st := a.prevNot(ignore, valuePointer-1, arrayStart+1); st >= 0 && f.isKeyword(st, "static") {
					valuePointer = st
				}
			}
			previous := a.prevNot(ignore, valuePointer-1, arrayStart+1)
			if previous < 0 {
				previous = stackPtr
			}
			prevIsWS := f.isWS(valuePointer - 1)
			if line(previous) == line(valuePointer) {
				if prevIsWS {
					f.replaceToken(valuePointer-1, f.eol)
				} else {
					f.addContentBefore(valuePointer, f.eol)
				}
			} else if prevIsWS {
				first := f.firstNonWhitespaceOnLine(valuePointer)
				if found := col(first) - 1; found != keywordStart {
					if found == 0 {
						f.addContent(first-1, spaces(keywordStart))
					} else {
						f.replaceToken(first-1, spaces(keywordStart))
					}
				}
			}
		}
	}

	indicesStart := keywordStart + 1
	for _, index := range indices {
		valuePointer := index.value
		if valuePointer < 0 || index.index < 0 {
			continue
		}
		indexPointer := index.index
		indexLine := line(indexPointer)
		previous := a.prevNot(func(i int) bool { return f.isWS(i) || f.is(i, ",") }, indexPointer-1, arrayStart+1)
		if previous < 0 {
			previous = stackPtr
		}
		if line(previous) == indexLine {
			if f.isWS(indexPointer - 1) {
				f.replaceToken(indexPointer-1, f.eol)
			} else {
				f.addContentBefore(indexPointer, f.eol)
			}
			continue
		}

		if col(indexPointer) != indicesStart && indexPointer-1 != arrayStart {
			expected, found := indicesStart-1, col(indexPointer)-1
			if found == 0 || !f.isWS(indexPointer-1) {
				f.addContent(indexPointer-1, spaces(expected))
			} else {
				f.replaceToken(indexPointer-1, spaces(expected))
			}
		}

		arrowStart := col(indexPointer) + maxLength + 1
		if col(index.arrow) != arrowStart {
			base := index.indexLength + col(indexPointer)
			expected, found := arrowStart-base, col(index.arrow)-base
			if found == 0 {
				f.addContent(index.arrow-1, spaces(expected))
			} else {
				f.replaceToken(index.arrow-1, spaces(expected))
			}
			continue
		}

		valueStart := arrowStart + 3
		if col(valuePointer) != valueStart {
			base := f.toks[index.arrow].length + col(index.arrow)
			expected, found := valueStart-base, col(valuePointer)-base
			switch {
			case found < 0:
				prev := a.prevNot(f.isWS, valuePointer-1, 0)
				f.beginChangeset()
				for i := prev + 1; i < valuePointer; i++ {
					f.replaceToken(i, "")
				}
				f.replaceToken(valuePointer-1, spaces(expected))
				f.endChangeset()
			case found == 0:
				f.addContent(valuePointer-1, spaces(expected))
			default:
				f.replaceToken(valuePointer-1, spaces(expected))
			}
		}

		nextComma := -1
		var valueEnd int
		end := sc.endOfStatement(valuePointer)
		if f.is(end, ",") {
			valueEnd = a.prevNot(f.isEmpty, end-1, valuePointer)
			nextComma = end
		} else {
			valueEnd = end
			if n := a.nextNot(f.isEmpty, end+1, arrayEnd); n >= 0 && f.is(n, ",") {
				nextComma = n
			}
		}
		valueLine := line(valueEnd)
		if a.isEndHeredoc(valueEnd) {
			valueLine++
		}

		if nextComma < 0 || line(nextComma) != valueLine {
			i := valuePointer + 1
			for ; i <= arrayEnd; i++ {
				if line(i) > valueLine {
					break
				}
			}
			f.beginChangeset()
			f.addContentBefore(i-1, ",")
			if nextComma >= 0 {
				f.replaceToken(nextComma, "")
			}
			f.endChangeset()
		}

		if nextComma >= 0 && f.isWS(nextComma-1) {
			if prev := a.prevNot(f.isEmpty, nextComma-1, 0); !a.isEndHeredoc(prev) {
				f.replaceToken(nextComma-1, "")
			}
		}
	}
}
