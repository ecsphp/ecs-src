package rules

import (
	"strings"

	"blink/internal/fixer"
	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Alias/ModernizeStrposFixer.php
//
// ModernizeStrpos replaces "strpos($h, $n) === 0" with "str_starts_with($h, $n)"
// and "strpos($h, $n) !== false" with "str_contains($h, $n)" ("!" is added for
// the opposite operators).
type ModernizeStrpos struct {
	// ModernizeStripos also converts stripos() calls, wrapping their arguments in
	// strtolower() (default false).
	ModernizeStripos bool
}

func (f ModernizeStrpos) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["modernize_stripos"].(bool); ok {
		f.ModernizeStripos = v
	}
	return f
}

func (ModernizeStrpos) Name() string {
	return `PhpCsFixer\Fixer\Alias\ModernizeStrposFixer`
}

func (ModernizeStrpos) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Alias/ModernizeStrposFixer.php"
}

var modernizeStrposReplacements = []struct {
	operator, operand, replacement string
	negate                         bool
}{
	{"===", "0", "str_starts_with", false},
	{"!==", "0", "str_starts_with", true},
	{"!==", "false", "str_contains", false},
	{"===", "false", "str_contains", true},
}

func (f ModernizeStrpos) Fix(s *tokens.Stream) bool {
	compactEmptyTokens(s)
	changed := false
	for index := s.Len() - 1; index > 0; index-- {
		t := s.At(index)
		if t.Kind != token.Ident {
			continue
		}
		isStrpos := strings.EqualFold(t.Value, "strpos")
		if !isStrpos && !(f.ModernizeStripos && strings.EqualFold(t.Value, "stripos")) {
			continue
		}
		if !aliasIsGlobalFunctionCall(s, index) {
			continue
		}

		// assert called with 2 arguments
		openIndex := aliasNextMeaningful(s, index)
		closeIndex := s.MatchForward(openIndex)
		if closeIndex < 0 || len(aliasArguments(s, openIndex, closeIndex)) != 2 {
			continue
		}

		// check if part of a condition and fix if needed
		operator, operand, ok := modernizeStrposCompareTokens(s, index, -1) // look behind
		if !ok {
			operator, operand, ok = modernizeStrposCompareTokens(s, closeIndex, 1) // look ahead
		}
		if ok {
			modernizeStrposFixCall(s, index, operator, operand, !isStrpos)
			changed = true
		}
	}
	if changed {
		aliasRelex(s)
	}
	return changed
}

// modernizeStrposFixCall rewrites the call; modernizeStrposCompareTokens has
// checked the operator and the operand ("0" or "false").
func modernizeStrposFixCall(s *tokens.Stream, functionIndex, operatorIndex, operandIndex int, isCaseInsensitive bool) {
	isFalse := s.At(operandIndex).Kind == token.Ident
	for _, r := range modernizeStrposReplacements {
		if s.At(operatorIndex).Value != r.operator || (r.operand == "false") != isFalse {
			continue
		}

		clearTokenAndMergeWS(s, operatorIndex)
		clearTokenAndMergeWS(s, operandIndex)
		clearTokenAndMergeWS(s, functionIndex)

		if r.negate {
			negateInsertIndex := functionIndex
			if p := aliasPrevMeaningful(s, functionIndex); isPunctVal(s, p, `\`) {
				negateInsertIndex = p
			}
			s.InsertAt(negateInsertIndex, aliasPunct("!"))
			functionIndex++
		}

		s.InsertAt(functionIndex, token.Token{Kind: token.Ident, Value: r.replacement})
		if isCaseInsensitive {
			modernizeStrposWrapArgumentsWithStrToLower(s, functionIndex)
		}
		return
	}
}

// modernizeStrposWrapArgumentsWithStrToLower wraps both arguments of the call
// at functionIndex in strtolower(), with PHP-CS-Fixer's exact index arithmetic.
func modernizeStrposWrapArgumentsWithStrToLower(s *tokens.Stream, functionIndex int) {
	shouldAddNamespace := functionIndex > 0 && isPunctVal(s, functionIndex-1, `\`)
	ns := 0
	if shouldAddNamespace {
		ns = 1
	}
	openIndex := aliasNextMeaningful(s, functionIndex)
	closeIndex := s.MatchForward(openIndex)
	arguments := aliasArguments(s, openIndex, closeIndex)
	if len(arguments) == 0 {
		return
	}
	first, last := arguments[0], arguments[len(arguments)-1]

	firstArgumentIndexStart := first[0]
	firstArgumentIndexEnd := first[1] + 3 + ns

	isSecondArgumentTokenWhiteSpace := s.At(last[0]).Kind == token.Whitespace
	ws := 0
	secondArgumentIndexStart := last[0]
	if isSecondArgumentTokenWhiteSpace {
		ws = 1
		secondArgumentIndexStart = aliasNextMeaningful(s, last[0])
	}
	secondArgumentIndexStart += 3 + ns
	secondArgumentIndexEnd := last[1] + 6 + ns + ws

	if shouldAddNamespace {
		s.InsertAt(firstArgumentIndexStart, aliasPunct(`\`))
		firstArgumentIndexStart++
	}
	s.InsertSliceAt(firstArgumentIndexStart, []token.Token{{Kind: token.Ident, Value: "strtolower"}, aliasPunct("(")})
	aliasInsertClamped(s, firstArgumentIndexEnd, aliasPunct(")"))

	if shouldAddNamespace {
		aliasInsertClamped(s, secondArgumentIndexStart, aliasPunct(`\`))
		secondArgumentIndexStart++
	}
	aliasInsertClamped(s, secondArgumentIndexStart, token.Token{Kind: token.Ident, Value: "strtolower"}, aliasPunct("("))
	aliasInsertClamped(s, secondArgumentIndexEnd, aliasPunct(")"))
}

// aliasInsertClamped inserts ts at i, appending when i is past the end (as
// Tokens::insertAt does).
func aliasInsertClamped(s *tokens.Stream, i int, ts ...token.Token) {
	s.InsertSliceAt(min(i, s.Len()), ts)
}

// modernizeStrposCompareTokens finds the "=== 0" / "!== false" style comparison
// next to offsetIndex in direction dir, returning the operator and operand.
func modernizeStrposCompareTokens(s *tokens.Stream, offsetIndex, dir int) (operatorIndex, operandIndex int, ok bool) {
	operatorIndex = aliasMeaningfulSibling(s, offsetIndex, dir)
	if isPunctVal(s, operatorIndex, `\`) {
		operatorIndex = aliasMeaningfulSibling(s, operatorIndex, dir)
	}
	if !isPunctVal(s, operatorIndex, "===") && !isPunctVal(s, operatorIndex, "!==") {
		return 0, 0, false
	}
	operandIndex = aliasMeaningfulSibling(s, operatorIndex, dir)
	if operandIndex < 0 {
		return 0, 0, false
	}
	operand := s.At(operandIndex)
	if !(operand.Kind == token.Number && operand.Value == "0") && !(operand.Kind == token.Ident && strings.EqualFold(operand.Value, "false")) {
		return 0, 0, false
	}
	if p := aliasMeaningfulSibling(s, operandIndex, dir); p >= 0 && modernizeStrposIsOfHigherPrecedence(s.At(p)) {
		return 0, 0, false
	}
	return operatorIndex, operandIndex, true
}

func modernizeStrposIsOfHigherPrecedence(t token.Token) bool {
	switch t.Kind {
	case token.Keyword:
		return strings.EqualFold(t.Value, "instanceof")
	case token.Punct:
		switch t.Value {
		case "--", "++", ">=", "<=", "**", "<<", ">>", "!", "%", "*", "+", "-", ".", "/", "<", ">", "~":
			return true
		}
	}
	return false
}
