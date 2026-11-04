package rules

import (
	"math"
	"strconv"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/lexer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/FunctionNotation/CombineNestedDirnameFixer.php
//
// CombineNestedDirname folds nested global dirname() calls into one call with
// the $levels argument: "dirname(dirname($path))" becomes "dirname($path, 2)",
// summing any integer levels the calls already pass. Risky: dirname may be
// overridden.
type CombineNestedDirname struct{}

func (CombineNestedDirname) Name() string {
	return `PhpCsFixer\Fixer\FunctionNotation\CombineNestedDirnameFixer`
}

func (CombineNestedDirname) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/FunctionNotation/CombineNestedDirnameFixer.php"
}

// dirnameInfo describes one dirname() call of a nested chain.
type dirnameInfo struct {
	indices        []int // tokens removed when the call is folded into its parent
	levels         int
	end            int // the closing ")"
	secondArgument int // the levels literal, or -1
}

func (CombineNestedDirname) Fix(s *tokens.Stream) bool {
	changed := false
	for index := s.Len() - 1; index >= 0; index-- {
		info, ok := getDirnameInfo(s, index, -1)
		if !ok {
			continue
		}
		prev := prevMeaningfulIndex(s, info.indices[0])
		if !isPunctVal(s, prev, "(") {
			continue
		}
		prev = prevMeaningfulIndex(s, prev)
		firstArgumentEnd := info.end
		chain := []dirnameInfo{info}
		for {
			outer, ok := getDirnameInfo(s, prev, firstArgumentEnd)
			if !ok {
				break
			}
			chain = append(chain, outer)
			prev = prevMeaningfulIndex(s, outer.indices[0])
			if !isPunctVal(s, prev, "(") {
				break
			}
			prev = prevMeaningfulIndex(s, prev)
			firstArgumentEnd = outer.end
		}
		if len(chain) > 1 {
			combineDirnames(s, chain)
			changed = true
		}
		index = prev
	}
	return changed
}

// getDirnameInfo ports CombineNestedDirnameFixer::getDirnameInfo. With
// firstArgumentEnd >= 0 the call's first argument is known to end there (it is
// the nested dirname call).
func getDirnameInfo(s *tokens.Stream, index, firstArgumentEnd int) (dirnameInfo, bool) {
	if index < 0 || s.At(index).Kind != token.Ident || !strings.EqualFold(s.At(index).Value, "dirname") {
		return dirnameInfo{}, false
	}
	if !aliasIsGlobalFunctionCall(s, index) {
		return dirnameInfo{}, false
	}
	info := dirnameInfo{secondArgument: -1}
	if prev := prevMeaningfulIndex(s, index); isPunctVal(s, prev, `\`) {
		info.indices = append(info.indices, prev)
	}
	info.indices = append(info.indices, index)

	next := nextMeaningfulIndex(s, index) // "("
	info.indices = append(info.indices, next)
	if firstArgumentEnd >= 0 {
		next = nextMeaningfulIndex(s, firstArgumentEnd)
	} else {
		next = nextMeaningfulIndex(s, next)
		if isPunctVal(s, next, ")") {
			return dirnameInfo{}, false
		}
		for next >= 0 && !isPunctVal(s, next, ",") && !isPunctVal(s, next, ")") {
			if e := aliasBlockEnd(s, next); e >= 0 {
				next = e
			}
			next = nextMeaningfulIndex(s, next)
		}
		if next < 0 {
			return dirnameInfo{}, false
		}
	}
	info.indices = append(info.indices, next)

	if isPunctVal(s, next, ",") {
		next = nextMeaningfulIndex(s, next)
		info.indices = append(info.indices, next)
	}
	if isPunctVal(s, next, ")") {
		info.levels = 1
		info.end = next
		return info, true
	}
	if next < 0 || s.At(next).Kind != token.Number || !lexer.IsIntegerLiteral(s.At(next).Value) {
		return dirnameInfo{}, false
	}
	info.secondArgument = next
	info.levels = phpIntCast(s.At(next).Value)

	next = nextMeaningfulIndex(s, next)
	if isPunctVal(s, next, ",") {
		info.indices = append(info.indices, next)
		next = nextMeaningfulIndex(s, next)
	}
	if !isPunctVal(s, next, ")") {
		return dirnameInfo{}, false
	}
	info.indices = append(info.indices, next)
	info.end = next
	return info, true
}

// combineDirnames folds the inner calls of chain (innermost first) into the
// outermost one, which gets the summed levels.
func combineDirnames(s *tokens.Stream, chain []dirnameInfo) {
	outer := chain[len(chain)-1]
	levels := phpIntSum{n: outer.levels}
	for _, inner := range chain[:len(chain)-1] {
		levels.add(inner.levels)
		for _, i := range inner.indices {
			removeWhitespaceSafely(s, i, -1)
			clearTokenAndMergeWS(s, i)
		}
	}
	levelsToken := token.Token{Kind: token.Number, Value: levels.String()}
	if outer.secondArgument >= 0 {
		levelsToken.Pos = s.At(outer.secondArgument).Pos
		s.Set(outer.secondArgument, levelsToken)
		return
	}
	var items []token.Token
	if !isPunctVal(s, prevMeaningfulIndex(s, outer.end), ",") {
		items = []token.Token{{Kind: token.Punct, Value: ","}, {Kind: token.Whitespace, Value: " "}}
	}
	items = append(items, levelsToken)
	s.InsertSliceAt(outer.end, items)
}

// phpIntCast mirrors PHP's (int) cast of the literal's text: it reads the
// leading decimal digits ("0x1" gives 0, "1_000" gives 1) and caps at PHP_INT_MAX.
func phpIntCast(v string) int {
	n := 0
	for i := 0; i < len(v) && v[i] >= '0' && v[i] <= '9'; i++ {
		d := int(v[i] - '0')
		if n > (math.MaxInt-d)/10 {
			return math.MaxInt
		}
		n = n*10 + d
	}
	return n
}

// phpIntSum adds levels the way PHP's "+=" does: an int sum that overflows
// becomes a float, and stays one.
type phpIntSum struct {
	n       int
	f       float64
	isFloat bool
}

func (p *phpIntSum) add(v int) {
	switch {
	case p.isFloat:
		p.f += float64(v)
	case (v > 0 && p.n > math.MaxInt-v) || (v < 0 && p.n < math.MinInt-v):
		p.f, p.isFloat = float64(p.n)+float64(v), true
	default:
		p.n += v
	}
}

// String renders the sum as PHP's (string) cast does: an int in decimal, a
// float with 14 significant digits ("9.2233720368548E+18", "1.0E+19").
func (p *phpIntSum) String() string {
	if !p.isFloat {
		return strconv.Itoa(p.n)
	}
	s := strconv.FormatFloat(p.f, 'G', 14, 64)
	if mant, exp, ok := strings.Cut(s, "E"); ok && !strings.Contains(mant, ".") {
		s = mant + ".0E" + exp
	}
	return s
}
