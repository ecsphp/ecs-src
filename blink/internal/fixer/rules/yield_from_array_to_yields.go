package rules

import (
	"slices"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ArrayNotation/YieldFromArrayToYieldsFixer.php
//
// YieldFromArrayToYields unpacks a "yield from [a, b, c]" (or "yield from
// array(...)") statement into a series of "yield a; yield b; yield c;". It is
// risky: yielded keys can differ when several "yield from" share a scope.
type YieldFromArrayToYields struct{}

func (YieldFromArrayToYields) Name() string {
	return `PhpCsFixer\Fixer\ArrayNotation\YieldFromArrayToYieldsFixer`
}

func (YieldFromArrayToYields) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ArrayNotation/YieldFromArrayToYieldsFixer.php"
}

type yfaytJob struct {
	index, start, end int
}

func (YieldFromArrayToYields) Fix(s *tokens.Stream) bool {
	jobs := yfaytCollect(s)
	if len(jobs) == 0 {
		return false
	}

	inserts := map[int][]token.Token{}
	for _, job := range jobs {
		clearTokenAndMergeWS(s, job.index) // clear "yield from"

		if isPunctVal(s, job.start, "(") {
			clearTokenAndMergeWS(s, aliasPrevMeaningful(s, job.start)) // clear "array"
		}
		clearTokenAndMergeWS(s, job.start)
		clearTokenAndMergeWS(s, job.end)

		arrayHasTrailingComma := false
		first := aliasNextMeaningful(s, job.start)
		inserts[first] = yfaytYield()

		for _, commaIndex := range yfaytItemCommas(s, first, aliasPrevMeaningful(s, job.end)) {
			nextItem := aliasNextMeaningful(s, commaIndex)
			if nextItem >= 0 && nextItem < job.end {
				inserts[nextItem] = yfaytYield()
			} else {
				arrayHasTrailingComma = true
			}
			s.SetValue(commaIndex, ";")
		}

		if arrayHasTrailingComma {
			clearTokenAndMergeWS(s, aliasNextMeaningful(s, job.end))
		}
	}

	keys := make([]int, 0, len(inserts))
	for k := range inserts {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	slices.Reverse(keys)
	for _, k := range keys {
		s.InsertSliceAt(k, inserts[k])
	}
	return true
}

func yfaytYield() []token.Token {
	return []token.Token{
		{Kind: token.Keyword, Value: "yield"},
		{Kind: token.Whitespace, Value: " "},
	}
}

func yfaytCollect(s *tokens.Stream) []yfaytJob {
	var jobs []yfaytJob
	for index := 1; index < s.Len(); index++ {
		if !isYieldFrom(s.At(index)) {
			continue
		}

		prev := aliasPrevMeaningful(s, index)
		if !yfaytValidPrev(s, prev) {
			continue
		}

		arrayStart := aliasNextMeaningful(s, index)
		if arrayStart < 0 {
			continue
		}

		var start, end int
		switch {
		case isPunctVal(s, arrayStart, "["):
			start = arrayStart
			end = s.MatchForward(start)
		case kwIs(s.At(arrayStart), "array"):
			start = nextPunctOfKind(s, arrayStart, "(")
			if start < 0 {
				continue
			}
			end = s.MatchForward(start)
		default:
			continue
		}
		if end < 0 {
			continue
		}

		// empty "yield from []"
		if end == aliasNextMeaningful(s, start) {
			continue
		}

		// nested "yield from"
		if yfaytHasNestedYieldFrom(s, start, end) {
			continue
		}

		jobs = append(jobs, yfaytJob{index: index, start: start, end: end})
	}
	return jobs
}

// yfaytValidPrev reports whether a "yield from" preceded by the token at prev
// starts a statement: ";", "{", "}" or an opening tag.
func yfaytValidPrev(s *tokens.Stream, prev int) bool {
	return prev >= 0 && (isPunctVal(s, prev, ";") || isPunctVal(s, prev, "{") ||
		isPunctVal(s, prev, "}") || aliasIsOpenTag(s, prev))
}

func yfaytHasNestedYieldFrom(s *tokens.Stream, start, end int) bool {
	for i := start + 1; i < end; i++ {
		if isYieldFrom(s.At(i)) {
			return true
		}
	}
	return false
}

// yfaytItemCommas yields the index of each top-level comma between start and end
// (inclusive), skipping nested (), [], {} and attribute blocks.
func yfaytItemCommas(s *tokens.Stream, start, end int) []int {
	var commas []int
	for i := start; i >= 0 && i <= end; i++ {
		if blockEnd := aliasBlockEnd(s, i); blockEnd >= 0 {
			i = blockEnd
			continue
		}
		if isPunctVal(s, i, ",") {
			commas = append(commas, i)
		}
	}
	return commas
}
