package rules

import (
	"blink/internal/fixer"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/DoctrineAnnotation/DoctrineAnnotationBracesFixer.php
//
// DoctrineAnnotationBraces turns `@Foo()` into `@Foo` (syntax "without_braces",
// the default) or `@Foo` into `@Foo()` (syntax "with_braces").
type DoctrineAnnotationBraces struct {
	withBraces  bool
	ignoredTags map[string]bool // nil keeps the default list
}

func (DoctrineAnnotationBraces) Name() string {
	return `PhpCsFixer\Fixer\DoctrineAnnotation\DoctrineAnnotationBracesFixer`
}

func (DoctrineAnnotationBraces) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/DoctrineAnnotation/DoctrineAnnotationBracesFixer.php"
}

func (f DoctrineAnnotationBraces) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["syntax"].(string); ok {
		f.withBraces = v == "with_braces"
	}
	if tags, ok := doctrineIgnoredTagsOption(config); ok {
		f.ignoredTags = tags
	}
	return f
}

func (f DoctrineAnnotationBraces) Fix(s *tokens.Stream) bool {
	ignored := f.ignoredTags
	if ignored == nil {
		ignored = doctrineIgnoredTags
	}
	if f.withBraces {
		return applyToDoctrineAnnotationsWith(s, ignored, doctrineAnnotationAddBraces)
	}
	return applyToDoctrineAnnotationsWith(s, ignored, doctrineAnnotationRemoveBraces)
}

// daNextMeaningful mirrors Doctrine\Annotation\Tokens::getNextMeaningfulToken.
func daNextMeaningful(toks []daToken, index int) int {
	for i := index + 1; i < len(toks); i++ {
		if toks[i].typ != daTNone {
			return i
		}
	}
	return -1
}

// doctrineAnnotationAddBraces mirrors addBracesToAnnotations.
func doctrineAnnotationAddBraces(toks []daToken) ([]daToken, bool) {
	changed := false
	for index := 0; index < len(toks); index++ {
		if toks[index].typ != daTAt {
			continue
		}
		if brace := daNextMeaningful(toks, index+1); brace >= 0 && toks[brace].typ == daTOpenParen {
			continue
		}
		at := min(index+2, len(toks))
		toks = append(toks[:at], append([]daToken{{typ: daTOpenParen, content: "("}, {typ: daTCloseParen, content: ")"}}, toks[at:]...)...)
		changed = true
	}
	return toks, changed
}

// doctrineAnnotationRemoveBraces mirrors removesBracesFromAnnotations.
func doctrineAnnotationRemoveBraces(toks []daToken) ([]daToken, bool) {
	changed := false
	for index := 0; index < len(toks); index++ {
		if toks[index].typ != daTAt {
			continue
		}
		open := daNextMeaningful(toks, index+1)
		if open < 0 || toks[open].typ != daTOpenParen {
			continue
		}
		closing := daNextMeaningful(toks, open)
		if closing < 0 || toks[closing].typ != daTCloseParen {
			continue
		}
		for i := index + 2; i <= closing; i++ {
			toks[i].content = ""
		}
		changed = true
	}
	return toks, changed
}
