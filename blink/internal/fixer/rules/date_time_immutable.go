package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ClassUsage/DateTimeImmutableFixer.php
//
// DateTimeImmutable swaps the mutable date API for the immutable one: the
// "DateTime" class for "DateTimeImmutable", "date_create" for
// "date_create_immutable" and "date_create_from_format" for
// "date_create_immutable_from_format". Risky: it changes behavior when the code
// relies on mutating DateTime objects or overrides the date_create* functions.
type DateTimeImmutable struct{}

func (DateTimeImmutable) Name() string {
	return `PhpCsFixer\Fixer\ClassUsage\DateTimeImmutableFixer`
}

func (DateTimeImmutable) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ClassUsage/DateTimeImmutableFixer.php"
}

// IsCandidate rules out a file holding none of the names the fixer rewrites.
func (DateTimeImmutable) IsCandidate(s *tokens.Stream) bool {
	return s.Index().HasAnyWord("datetime", "date_create", "date_create_from_format")
}

func (DateTimeImmutable) Fix(s *tokens.Stream) bool {
	functionMap := map[string]string{
		"date_create":             "date_create_immutable",
		"date_create_from_format": "date_create_immutable_from_format",
	}

	changed := false
	isInNamespace := false
	isImported := false // e.g. use DateTime;

	for index := 0; index < s.Len(); index++ {
		t := s.At(index)

		if kwIs(t, "namespace") {
			isInNamespace = true
			continue
		}

		if isInNamespace && kwIs(t, "use") {
			nextIndex := aliasNextMeaningful(s, index)
			if nextIndex < 0 || !strings.EqualFold(s.At(nextIndex).Value, "datetime") {
				continue
			}
			nextNextIndex := aliasNextMeaningful(s, nextIndex)
			if nextNextIndex < 0 {
				break
			}
			if isPunctVal(s, nextNextIndex, ";") {
				isImported = true
			}
			index = nextNextIndex
			continue
		}

		if t.Kind != token.Ident {
			continue
		}

		prevIndex := aliasPrevMeaningful(s, index)
		if prevIndex >= 0 && kwIs(s.At(prevIndex), "function") {
			continue
		}

		lower := strings.ToLower(t.Value)

		if lower == "datetime" {
			if dateTimeImmutableFixClassUsage(s, index, isInNamespace, isImported) {
				changed = true
			}
			continue
		}

		if repl, ok := functionMap[lower]; ok && aliasIsGlobalFunctionCall(s, index) {
			s.SetValue(index, repl)
			changed = true
		}
	}

	return changed
}

// dateTimeImmutableFixClassUsage ports DateTimeImmutableFixer::fixClassUsage: it
// rewrites a bare or leading-backslash "DateTime" class reference, inserting a
// leading "\" when inside a namespace with a plain (imported) usage.
func dateTimeImmutableFixClassUsage(s *tokens.Stream, index int, isInNamespace, isImported bool) bool {
	nextIndex := aliasNextMeaningful(s, index)
	if isPunctVal(s, nextIndex, "::") {
		nextNextIndex := aliasNextMeaningful(s, nextIndex)
		if nextNextIndex >= 0 && s.At(nextNextIndex).Kind == token.Ident {
			nextNextNextIndex := aliasNextMeaningful(s, nextNextIndex)
			if !isPunctVal(s, nextNextNextIndex, "(") {
				return false
			}
		}
	}

	isUsedAlone := false                // e.g. new DateTime();
	isUsedWithLeadingBackslash := false // e.g. new \DateTime();

	prevIndex := aliasPrevMeaningful(s, index)
	if isPunctVal(s, prevIndex, `\`) {
		prevPrevIndex := aliasPrevMeaningful(s, prevIndex)
		if prevPrevIndex < 0 || s.At(prevPrevIndex).Kind != token.Ident {
			isUsedWithLeadingBackslash = true
		}
	} else if !isPunctVal(s, prevIndex, "::") && !dateTimeImmutableIsObjectOperator(s, prevIndex) {
		isUsedAlone = true
	}

	if isUsedWithLeadingBackslash || (isUsedAlone && ((isInNamespace && isImported) || !isInNamespace)) {
		s.SetValue(index, "DateTimeImmutable")
		if isInNamespace && isUsedAlone {
			s.InsertAt(index, token.Token{Kind: token.Punct, Value: `\`})
		}
		return true
	}

	return false
}

// dateTimeImmutableIsObjectOperator mirrors Token::isObjectOperator: the "->" and
// nullsafe "?->" member-access operators.
func dateTimeImmutableIsObjectOperator(s *tokens.Stream, i int) bool {
	return isPunctVal(s, i, "->") || isPunctVal(s, i, "?->")
}
