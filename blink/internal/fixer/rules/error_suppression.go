package rules

import (
	"slices"
	"strings"

	"blink/internal/fixer"
	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/LanguageConstruct/ErrorSuppressionFixer.php
//
// ErrorSuppression adds "@" to trigger_error(..., E_USER_DEPRECATED) calls and,
// with noise_remaining_usages, removes it everywhere else.
type ErrorSuppression struct {
	configured                  bool
	muteDeprecationError        bool     // default true
	noiseRemainingUsages        bool     // default false
	noiseRemainingUsagesExclude []string // lowercased
}

func (ErrorSuppression) Name() string {
	return `PhpCsFixer\Fixer\LanguageConstruct\ErrorSuppressionFixer`
}

func (ErrorSuppression) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/LanguageConstruct/ErrorSuppressionFixer.php"
}

func (f ErrorSuppression) WithConfig(config map[string]any) fixer.Fixer {
	f.configured = true
	f.muteDeprecationError = true
	if v, ok := config["mute_deprecation_error"].(bool); ok {
		f.muteDeprecationError = v
	}
	if v, ok := config["noise_remaining_usages"].(bool); ok {
		f.noiseRemainingUsages = v
	}
	f.noiseRemainingUsagesExclude = nil
	for _, name := range singleElementConfigStrings(config["noise_remaining_usages_exclude"]) {
		f.noiseRemainingUsagesExclude = append(f.noiseRemainingUsagesExclude, strings.ToLower(name))
	}
	return f
}

func (f ErrorSuppression) Fix(s *tokens.Stream) bool {
	mute := !f.configured || f.muteDeprecationError
	noise := f.noiseRemainingUsages

	var analysis gfcAnalysis
	changed := false
	for index := s.Len() - 1; index >= 0; index-- {
		t := s.At(index)
		if noise && t.Kind == token.Punct && t.Value == "@" {
			clearTokenAt(s, index)
			changed = true
			continue
		}
		// without noise_remaining_usages, only trigger_error calls can change
		if !noise && (t.Kind != token.Ident || !strings.EqualFold(t.Value, "trigger_error")) {
			continue
		}
		if !analysis.isGlobalFunctionCall(s, index) {
			continue
		}
		functionIndex, startIndex := index, index
		prev := sigPrev(s, index)
		if isPunctVal(s, prev, `\`) {
			startIndex = prev
			prev = sigPrev(s, startIndex)
		}
		index = prev

		if errorSuppressionIsDeprecationCall(s, functionIndex) {
			if !mute || isPunctVal(s, prev, "@") {
				continue
			}
			s.InsertAt(startIndex, token.Token{Kind: token.Punct, Value: "@"})
			analysis.invalidate()
			changed = true
			continue
		}
		if !isPunctVal(s, prev, "@") {
			continue
		}
		// the exclude list is lowercased, the call name is compared as written
		if noise && !slices.Contains(f.noiseRemainingUsagesExclude, s.At(functionIndex).Value) {
			clearTokenAt(s, prev)
			changed = true
		}
	}
	if changed {
		compactEmptyTokens(s)
	}
	return changed
}

// errorSuppressionIsDeprecationCall reports whether the global call at index is
// trigger_error(..., E_USER_DEPRECATED).
func errorSuppressionIsDeprecationCall(s *tokens.Stream, index int) bool {
	if !strings.EqualFold(s.At(index).Value, "trigger_error") {
		return false
	}
	open := -1
	for j := index + 1; j < s.Len(); j++ {
		if gfcIsTString(s, j) || isPunctVal(s, j, "(") {
			open = j
			break
		}
	}
	closeIdx := s.MatchForward(open)
	if closeIdx < 0 {
		return false
	}
	prev := sigPrev(s, closeIdx)
	if isPunctVal(s, prev, ",") {
		prev = sigPrev(s, prev)
	}
	return gfcIsTString(s, prev) && s.At(prev).Value == "E_USER_DEPRECATED"
}
